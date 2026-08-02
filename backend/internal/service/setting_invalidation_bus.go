package service

import "context"

// SettingsInvalidationBus publishes and subscribes to settings-change
// notifications over Redis Pub/Sub so the local onUpdate callback (frontend
// HTML cache invalidation + CSP refresh) converges across replicas.
//
// The bus intentionally carries no user, request, or settings data: a fixed
// static channel plus a fixed static expected payload is all that is ever
// published, and subscribers ignore anything unexpected. This keeps the
// notification a pure "something changed, re-read from the DB" signal.
type SettingsInvalidationBus interface {
	// PublishSettingsInvalidation notifies sibling replicas that settings
	// changed. Callers treat failure as best-effort (log only): the settings
	// write that triggered the notification has already committed.
	PublishSettingsInvalidation(ctx context.Context) error

	// SubscribeSettingsInvalidation blocks until ctx is done. For every
	// validated settings-changed notification it invokes handler (never for
	// unexpected payloads). It returns ctx.Err() on cancellation or a non-nil
	// error when the subscription fails or its channel closes, so callers can
	// retry with backoff. It must close any resources it opened on the way out.
	SubscribeSettingsInvalidation(ctx context.Context, handler func()) error
}

type settingsSubscriptionReadyKey struct{}

func withSettingsSubscriptionReady(ctx context.Context, ready func()) context.Context {
	return context.WithValue(ctx, settingsSubscriptionReadyKey{}, ready)
}

// NotifySettingsSubscriptionReady lets bus implementations report that the
// Redis server acknowledged the subscription without widening the bus API.
func NotifySettingsSubscriptionReady(ctx context.Context) {
	if ready, ok := ctx.Value(settingsSubscriptionReadyKey{}).(func()); ok && ready != nil {
		ready()
	}
}

// SettingsInvalidationSubscriberHealth exposes subscriber connectivity for
// observability and tests.
type SettingsInvalidationSubscriberHealth struct {
	Connected bool   `json:"connected"`
	Failures  uint64 `json:"failures"`
}
