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

// settingsBusRepoStub is a minimal SettingRepository for invalidation-bus tests.
type settingsBusRepoStub struct {
	mu      sync.Mutex
	updates map[string]string
	values  map[string]string
	err     error
}

func (s *settingsBusRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: v}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *settingsBusRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingsBusRepoStub) Set(ctx context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *settingsBusRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func (s *settingsBusRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if s.updates == nil {
		s.updates = map[string]string{}
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	for k, v := range settings {
		s.updates[k] = v
		s.values[k] = v
	}
	return nil
}

func (s *settingsBusRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}

func (s *settingsBusRepoStub) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}

// settingsInvalidationBusStub records publishes and lets tests drive the
// subscription side manually.
type settingsInvalidationBusStub struct {
	mu          sync.Mutex
	published   int
	publishErr  error
	subscribeFn func(ctx context.Context, handler func()) error
}

func (b *settingsInvalidationBusStub) PublishSettingsInvalidation(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published++
	return b.publishErr
}

func (b *settingsInvalidationBusStub) SubscribeSettingsInvalidation(ctx context.Context, handler func()) error {
	if b.subscribeFn != nil {
		return b.subscribeFn(ctx, handler)
	}
	return nil
}

func (b *settingsInvalidationBusStub) publishCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.published
}

func TestSettingService_LocalUpdateInvokesCallbackAndPublisher(t *testing.T) {
	repo := &settingsBusRepoStub{}
	bus := &settingsInvalidationBusStub{}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSettingsInvalidationBus(bus)

	var callbackCalls int64
	svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

	err := svc.UpdateSettings(context.Background(), &SystemSettings{CompactHomeEnabled: true})
	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingKeyCompactHomeEnabled])
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls))
	require.Equal(t, 1, bus.publishCount())
}

func TestSettingService_PublisherFailureDoesNotFailWrite(t *testing.T) {
	repo := &settingsBusRepoStub{}
	bus := &settingsInvalidationBusStub{publishErr: errors.New("redis unavailable")}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSettingsInvalidationBus(bus)

	var callbackCalls int64
	svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

	err := svc.UpdateSettings(context.Background(), &SystemSettings{CompactHomeEnabled: true})
	require.NoError(t, err, "a publish failure must not turn a committed update into an error")
	require.Equal(t, "true", repo.updates[SettingKeyCompactHomeEnabled])
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls), "local callback still runs")
	require.Equal(t, 1, bus.publishCount())
}

func TestSettingService_NilBusLocalOnlyBehavior(t *testing.T) {
	repo := &settingsBusRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	var callbackCalls int64
	svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

	require.NotPanics(t, func() {
		svc.StartSettingsInvalidationSubscriber(context.Background())
		svc.StopSettingsInvalidationSubscriber()
		require.NoError(t, svc.UpdateSettings(context.Background(), &SystemSettings{CompactHomeEnabled: true}))
	})
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls))
	require.NotPanics(t, func() { svc.StopSettingsInvalidationSubscriber() })
}

func TestSettingService_RemoteEventInvokesCallbackWithoutPublishLoop(t *testing.T) {
	ready := make(chan struct{})
	var remoteHandler func()
	bus := &settingsInvalidationBusStub{subscribeFn: func(ctx context.Context, handler func()) error {
		remoteHandler = handler
		NotifySettingsSubscriptionReady(ctx)
		close(ready)
		<-ctx.Done()
		return ctx.Err()
	}}
	repo := &settingsBusRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSettingsInvalidationBus(bus)

	var callbackCalls int64
	svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

	svc.StartSettingsInvalidationSubscriber(context.Background())
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("subscriber did not connect")
	}
	require.True(t, svc.SettingsInvalidationSubscriberHealth().Connected)

	// A remote event re-runs the local callback exactly once and never republishes.
	require.NotNil(t, remoteHandler)
	remoteHandler()
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls))
	require.Equal(t, 0, bus.publishCount(), "remote events must never be republished")

	svc.StopSettingsInvalidationSubscriber()
	require.False(t, svc.SettingsInvalidationSubscriberHealth().Connected)
}

func TestSettingService_SubscriberRetriesInitialFailureAndStops(t *testing.T) {
	ready := make(chan struct{})
	var calls int
	var mu sync.Mutex
	bus := &settingsInvalidationBusStub{subscribeFn: func(ctx context.Context, _ func()) error {
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		if call == 1 {
			return errors.New("redis starting")
		}
		NotifySettingsSubscriptionReady(ctx)
		close(ready)
		<-ctx.Done()
		return ctx.Err()
	}}
	repo := &settingsBusRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSettingsInvalidationBus(bus)

	svc.StartSettingsInvalidationSubscriber(context.Background())
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("subscriber did not retry after initial failure")
	}
	require.Eventually(t, func() bool {
		return svc.SettingsInvalidationSubscriberHealth().Connected
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, uint64(1), svc.SettingsInvalidationSubscriberHealth().Failures)
	require.NotPanics(t, func() {
		svc.StopSettingsInvalidationSubscriber()
		svc.StopSettingsInvalidationSubscriber()
	})
	require.False(t, svc.SettingsInvalidationSubscriberHealth().Connected)
}

func TestSettingService_UpdateAuthSourceDefaultSettingsNotifies(t *testing.T) {
	repo := &settingsBusRepoStub{}
	bus := &settingsInvalidationBusStub{}
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSettingsInvalidationBus(bus)

	var callbackCalls int64
	svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

	err := svc.UpdateAuthSourceDefaultSettings(context.Background(), &AuthSourceDefaultSettings{
		ForceEmailOnThirdPartySignup: true,
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingKeyForceEmailOnThirdPartySignup])
	require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls))
	require.Equal(t, 1, bus.publishCount())
}

func TestSettingService_PaymentEnabledUpdateNotifiesOnlyOnCommittedToggle(t *testing.T) {
	repo := &settingsBusRepoStub{}
	configSvc := NewPaymentConfigService(nil, repo, nil)

	var notified int64
	configSvc.SetOnPaymentEnabledUpdate(func() { atomic.AddInt64(&notified, 1) })

	enabled := true
	err := configSvc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{Enabled: &enabled})
	require.NoError(t, err)
	require.Equal(t, "true", repo.updates[SettingPaymentEnabled])
	require.Equal(t, int64(1), atomic.LoadInt64(&notified), "committed payment_enabled toggle notifies")

	// A write that does not touch SettingPaymentEnabled must not notify.
	minAmount := 5.0
	err = configSvc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{MinAmount: &minAmount})
	require.NoError(t, err)
	require.Equal(t, int64(1), atomic.LoadInt64(&notified), "non-payment-enabled write must not notify")
}

func TestSettingService_PaymentEnabledWriteFailureDoesNotNotify(t *testing.T) {
	repo := &settingsBusRepoStub{err: errors.New("db down")}
	configSvc := NewPaymentConfigService(nil, repo, nil)

	var notified int64
	configSvc.SetOnPaymentEnabledUpdate(func() { atomic.AddInt64(&notified, 1) })

	enabled := true
	err := configSvc.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{Enabled: &enabled})
	require.Error(t, err)
	require.Equal(t, int64(0), atomic.LoadInt64(&notified), "uncommitted write must not notify")
}
