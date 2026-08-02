//go:build unit

package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// settingsReReadFailRepoStub commits SetMultiple but fails the post-commit
// GetAllSettings re-read (or reports a canceled context), so tests can verify
// that a committed settings write still notifies exactly once.
type settingsReReadFailRepoStub struct {
	updates map[string]string
}

func (s *settingsReReadFailRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingsReReadFailRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingsReReadFailRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *settingsReReadFailRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingsReReadFailRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for k, v := range settings {
		s.updates[k] = v
	}
	return nil
}

func (s *settingsReReadFailRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("re-read failed")
}

func (s *settingsReReadFailRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

// TestSettingService_PartialUpdateReReadFailureStillNotifies pins the Oracle
// gate finding: a settings write that committed must notify the local onUpdate
// callback and the cross-replica publisher exactly once, even when the
// post-commit cache re-read fails or its request context was canceled. The
// re-read failure is logged/best-effort, so the update method still returns nil
// (the write already committed; only persistence/validation failures surface as
// errors), and it must never skip the notification.
func TestSettingService_PartialUpdateReReadFailureStillNotifies(t *testing.T) {
	run := func(t *testing.T, svc *SettingService, repo *settingsReReadFailRepoStub, bus *settingsInvalidationBusStub, ctx context.Context) {
		t.Helper()

		var callbackCalls int64
		svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

		err := svc.UpdateSettingsOmitting(
			ctx,
			&SystemSettings{CompactHomeEnabled: true},
			OmittedSettingKeys{SettingKeySiteName: {}},
		)
		require.NoError(t, err, "a committed write must succeed even when the post-commit re-read fails")
		require.Equal(t, "true", repo.updates[SettingKeyCompactHomeEnabled], "commit must have succeeded")
		require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls), "local callback runs exactly once")
		require.Equal(t, 1, bus.publishCount(), "cross-replica publish happens exactly once")
	}

	t.Run("re_read_error", func(t *testing.T) {
		repo := &settingsReReadFailRepoStub{}
		bus := &settingsInvalidationBusStub{}
		svc := NewSettingService(repo, &config.Config{})
		svc.SetSettingsInvalidationBus(bus)

		run(t, svc, repo, bus, context.Background())
	})

	t.Run("canceled_context", func(t *testing.T) {
		repo := &settingsReReadFailRepoStub{}
		bus := &settingsInvalidationBusStub{}
		svc := NewSettingService(repo, &config.Config{})
		svc.SetSettingsInvalidationBus(bus)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		run(t, svc, repo, bus, ctx)
	})

	t.Run("with_auth_source_defaults", func(t *testing.T) {
		repo := &settingsReReadFailRepoStub{}
		bus := &settingsInvalidationBusStub{}
		svc := NewSettingService(repo, &config.Config{})
		svc.SetSettingsInvalidationBus(bus)

		var callbackCalls int64
		svc.SetOnUpdateCallback(func() { atomic.AddInt64(&callbackCalls, 1) })

		err := svc.UpdateSettingsWithAuthSourceDefaultsOmitting(
			context.Background(),
			&SystemSettings{CompactHomeEnabled: true},
			&AuthSourceDefaultSettings{},
			OmittedSettingKeys{SettingKeySiteName: {}},
		)
		require.NoError(t, err, "a committed write must succeed even when the post-commit re-read fails")
		require.Equal(t, "true", repo.updates[SettingKeyCompactHomeEnabled], "commit must have succeeded")
		require.Equal(t, int64(1), atomic.LoadInt64(&callbackCalls), "local callback runs exactly once")
		require.Equal(t, 1, bus.publishCount(), "cross-replica publish happens exactly once")
	})
}
