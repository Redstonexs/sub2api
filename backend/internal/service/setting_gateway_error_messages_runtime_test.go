//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoadGatewayErrorMessages_AppliesDBOverride(t *testing.T) {
	repo := &settingsBusRepoStub{values: map[string]string{
		SettingKeyGatewayErrorMessages: `{"429":"Please retry later"}`,
	}}
	cfg := &config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
	svc := NewSettingService(repo, cfg)

	require.NoError(t, svc.LoadGatewayErrorMessages(context.Background()))
	require.Equal(t, "Please retry later", config.GatewayErrorMessage(cfg, 429, "fallback"), "DB override is applied")
	require.Equal(t, "static 503", config.GatewayErrorMessage(cfg, 503, "fallback"), "static config is untouched for codes the DB does not override")
}

func TestLoadGatewayErrorMessages_MissingKeyClearsPriorOverride(t *testing.T) {
	repo := &settingsBusRepoStub{} // gateway_error_messages key absent
	cfg := &config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
	svc := NewSettingService(repo, cfg)

	cfg.SetGatewayErrorMessages(map[string]string{"429": "stale override"})
	require.NoError(t, svc.LoadGatewayErrorMessages(context.Background()))
	require.Equal(t, "fallback", config.GatewayErrorMessage(cfg, 429, "fallback"), "missing DB key clears the prior live override")
	require.Equal(t, "static 503", config.GatewayErrorMessage(cfg, 503, "fallback"))
}

func TestLoadGatewayErrorMessages_BlankAndEmptyObjectClearOverride(t *testing.T) {
	for _, raw := range []string{"", "   ", "{}"} {
		t.Run("raw="+raw, func(t *testing.T) {
			repo := &settingsBusRepoStub{values: map[string]string{SettingKeyGatewayErrorMessages: raw}}
			cfg := &config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
			svc := NewSettingService(repo, cfg)

			cfg.SetGatewayErrorMessages(map[string]string{"429": "stale override"})
			require.NoError(t, svc.LoadGatewayErrorMessages(context.Background()))
			require.Equal(t, "fallback", config.GatewayErrorMessage(cfg, 429, "fallback"), "blank/empty stored value clears the prior live override")
			require.Equal(t, "static 503", config.GatewayErrorMessage(cfg, 503, "fallback"))
		})
	}
}

func TestLoadGatewayErrorMessages_InvalidJSONClearsOverride(t *testing.T) {
	repo := &settingsBusRepoStub{values: map[string]string{SettingKeyGatewayErrorMessages: "not json"}}
	cfg := &config.Config{}
	svc := NewSettingService(repo, cfg)

	cfg.SetGatewayErrorMessages(map[string]string{"429": "stale override"})
	require.NoError(t, svc.LoadGatewayErrorMessages(context.Background()))
	require.Equal(t, "fallback", config.GatewayErrorMessage(cfg, 429, "fallback"), "invalid JSON is treated as no override, like parseSettings")
}

type gatewayErrorMessagesLoadErrorRepo struct {
	SettingRepository
}

func (r *gatewayErrorMessagesLoadErrorRepo) GetValue(context.Context, string) (string, error) {
	return "", errors.New("db down")
}

func TestLoadGatewayErrorMessages_ReadErrorKeepsPriorOverride(t *testing.T) {
	repo := &gatewayErrorMessagesLoadErrorRepo{}
	cfg := &config.Config{}
	svc := NewSettingService(repo, cfg)

	cfg.SetGatewayErrorMessages(map[string]string{"429": "prior override"})
	err := svc.LoadGatewayErrorMessages(context.Background())
	require.Error(t, err, "a real DB error must surface")
	require.Equal(t, "prior override", config.GatewayErrorMessage(cfg, 429, "fallback"), "a read error must never clear the prior live override")
}

// relayedSettingsInvalidationBus routes publishes from one replica to the other
// replica's subscriber handler, mimicking a single Redis Pub/Sub channel shared
// by two gateway replicas. It records publish counts so tests can assert the
// remote-events-never-republish invariant.
type relayedSettingsInvalidationBus struct {
	mu        sync.Mutex
	remote    *relayedSettingsInvalidationBus
	handler   func()
	published int
}

func (b *relayedSettingsInvalidationBus) PublishSettingsInvalidation(context.Context) error {
	b.mu.Lock()
	b.published++
	b.mu.Unlock()
	remote := b.remote
	if remote == nil {
		return nil
	}
	remote.mu.Lock()
	h := remote.handler
	remote.mu.Unlock()
	if h != nil {
		h()
	}
	return nil
}

func (b *relayedSettingsInvalidationBus) SubscribeSettingsInvalidation(ctx context.Context, handler func()) error {
	b.mu.Lock()
	b.handler = handler
	b.mu.Unlock()
	NotifySettingsSubscriptionReady(ctx)
	<-ctx.Done()
	return ctx.Err()
}

func (b *relayedSettingsInvalidationBus) publishCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.published
}

// TestSettingService_RemoteEventConvergesGatewayErrorMessagesWithoutPublishing
// proves the second replica applies the gateway_error_messages override when it
// receives a remote invalidation event, and that it never republishes the event
// (no cross-replica loop). Replica A commits the setting on the shared DB;
// replica B, subscribed to the shared bus, converges its config snapshot before
// running its UI/CSP callback.
func TestSettingService_RemoteEventConvergesGatewayErrorMessagesWithoutPublishing(t *testing.T) {
	repo := &settingsBusRepoStub{}
	busA := &relayedSettingsInvalidationBus{}
	busB := &relayedSettingsInvalidationBus{}
	busA.remote = busB
	busB.remote = busA

	cfgA := &config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
	cfgB := &config.Config{Gateway: config.GatewayConfig{ErrorMessages: map[string]string{"503": "static 503"}}}
	svcA := NewSettingService(repo, cfgA)
	svcA.SetSettingsInvalidationBus(busA)
	svcB := NewSettingService(repo, cfgB)
	svcB.SetSettingsInvalidationBus(busB)

	var callbackACalls, callbackBCalls int64
	svcA.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackACalls, 1) })
	svcB.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackBCalls, 1) })

	// Replica B subscribes to the shared channel and registers its handler.
	svcB.StartSettingsInvalidationSubscriber(context.Background())
	require.Eventually(t, func() bool {
		return svcB.SettingsInvalidationSubscriberHealth().Connected
	}, time.Second, 10*time.Millisecond)
	t.Cleanup(svcB.StopSettingsInvalidationSubscriber)

	// Replica A commits the gateway_error_messages setting.
	require.NoError(t, svcA.UpdateSettings(context.Background(), &SystemSettings{
		GatewayErrorMessages: map[string]string{"429": "Too many requests"},
	}))

	// Replica A applied the override locally after its own write and published once.
	require.Equal(t, "Too many requests", config.GatewayErrorMessage(cfgA, 429, "fallback"))
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackACalls))
	require.Equal(t, 1, busA.publishCount(), "replica A publishes exactly once after its commit")

	// Replica B converges via the remote event: re-reads the DB, applies the
	// override to its config snapshot, runs its callback, and never republishes.
	require.Eventually(t, func() bool {
		return config.GatewayErrorMessage(cfgB, 429, "fallback") == "Too many requests"
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, "static 503", config.GatewayErrorMessage(cfgB, 503, "fallback"), "static config fallback still applies on the second replica")
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackBCalls), "second replica callback runs exactly once")
	require.Equal(t, 0, busB.publishCount(), "remote events must never be republished")

	// Clearing on replica A propagates to replica B: the empty stored map clears
	// B's live override too, so the static fallback resumes.
	require.NoError(t, svcA.UpdateSettings(context.Background(), &SystemSettings{
		GatewayErrorMessages: map[string]string{},
	}))
	require.Eventually(t, func() bool {
		return config.GatewayErrorMessage(cfgB, 429, "fallback") == "fallback"
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, "static 503", config.GatewayErrorMessage(cfgB, 503, "fallback"))
	require.Equal(t, 0, busB.publishCount(), "convergence on clear also never republishes")
}
