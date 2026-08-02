package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// settingsInvalidationPublishTimeout bounds the best-effort Redis PUBLISH so a
// slow or unreachable Redis can never block a settings write that already
// committed.
const settingsInvalidationPublishTimeout = 2 * time.Second

const settingsInvalidationRetryMaxBackoff = 30 * time.Second

// SetSettingsInvalidationBus injects the optional Redis Pub/Sub settings
// invalidation bus. Nil-safe: without a bus the service simply falls back to
// local-only invalidation (single-replica behavior).
func (s *SettingService) SetSettingsInvalidationBus(bus SettingsInvalidationBus) {
	if s == nil {
		return
	}
	s.settingsInvalidationBus = bus
}

// StartSettingsInvalidationSubscriber starts the Pub/Sub subscriber that
// re-invokes the local onUpdate callback when another replica commits a
// settings write. It must only be called after SetOnUpdateCallback has been
// wired (i.e. after SetupRouter). Nil-safe: no-op when the service or the bus
// is nil. The goroutine retries with bounded exponential backoff and stops on
// context cancellation.
func (s *SettingService) StartSettingsInvalidationSubscriber(ctx context.Context) {
	if s == nil || s.settingsInvalidationBus == nil {
		return
	}
	s.settingsInvalidationStart.Do(func() {
		subscriberCtx, cancel := context.WithCancel(ctx)
		subscriberCtx = withSettingsSubscriptionReady(subscriberCtx, func() {
			s.settingsInvalidationConnected.Store(true)
		})
		s.settingsInvalidationCancel = cancel
		s.settingsInvalidationWG.Add(1)
		go func() {
			defer s.settingsInvalidationWG.Done()
			backoff := time.Second
			for {
				err := s.settingsInvalidationBus.SubscribeSettingsInvalidation(subscriberCtx, func() {
					s.handleRemoteSettingsInvalidation()
				})
				wasConnected := s.settingsInvalidationConnected.Swap(false)
				if subscriberCtx.Err() != nil {
					return
				}
				if wasConnected {
					backoff = time.Second
				}
				s.settingsInvalidationFailures.Add(1)
				if err == nil {
					err = errors.New("settings invalidation subscription closed")
				}
				slog.Warn("failed to start settings invalidation subscriber; retrying",
					"error", err, "retry_in", backoff)
				timer := time.NewTimer(backoff)
				select {
				case <-subscriberCtx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				if backoff < settingsInvalidationRetryMaxBackoff {
					backoff *= 2
					if backoff > settingsInvalidationRetryMaxBackoff {
						backoff = settingsInvalidationRetryMaxBackoff
					}
				}
			}
		}()
	})
}

// handleRemoteSettingsInvalidation re-runs the local onUpdate callback only.
// A remote event must never be republished — the originating replica already
// published, and republishing would create an infinite cross-replica loop.
func (s *SettingService) handleRemoteSettingsInvalidation() {
	if s == nil {
		return
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
}

// StopSettingsInvalidationSubscriber cancels the subscriber context and waits
// for the goroutine to exit. Idempotent and nil-safe; must be called from
// server cleanup before Redis is closed.
func (s *SettingService) StopSettingsInvalidationSubscriber() {
	if s == nil {
		return
	}
	s.settingsInvalidationStop.Do(func() {
		if s.settingsInvalidationCancel != nil {
			s.settingsInvalidationCancel()
		}
		s.settingsInvalidationWG.Wait()
	})
}

// SettingsInvalidationSubscriberHealth reports subscriber connectivity and the
// number of retry cycles observed since start.
func (s *SettingService) SettingsInvalidationSubscriberHealth() SettingsInvalidationSubscriberHealth {
	if s == nil {
		return SettingsInvalidationSubscriberHealth{}
	}
	return SettingsInvalidationSubscriberHealth{
		Connected: s.settingsInvalidationConnected.Load(),
		Failures:  s.settingsInvalidationFailures.Load(),
	}
}

// NotifySettingsChanged is the centralized post-commit settings notification:
// it runs the local onUpdate callback (frontend HTML cache invalidation + CSP
// refresh) and best-effort publishes a settings-changed notification to
// sibling replicas. Publish failure is logged and must not turn a committed
// update into an error. Writes that bypass refreshCachedSettings (auth-source
// defaults, payment-enabled toggles) route through here too. Remote events
// must never call it (see handleRemoteSettingsInvalidation).
func (s *SettingService) NotifySettingsChanged() {
	if s == nil {
		return
	}
	if s.onUpdate != nil {
		s.onUpdate() // Invalidate cache after settings update
	}
	s.publishSettingsInvalidation()
}

func (s *SettingService) publishSettingsInvalidation() {
	if s == nil || s.settingsInvalidationBus == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), settingsInvalidationPublishTimeout)
	defer cancel()
	if err := s.settingsInvalidationBus.PublishSettingsInvalidation(ctx); err != nil {
		slog.Warn("failed to publish settings invalidation to sibling replicas", "error", err)
	}
}
