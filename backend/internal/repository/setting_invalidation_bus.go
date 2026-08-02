package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	// settingsInvalidateChannel is the fixed static channel for settings-change
	// notifications. Nothing dynamic (user/request/settings data) is ever
	// placed in a message: replicas only need "something changed".
	settingsInvalidateChannel = "settings:invalidate"
	// settingsInvalidatePayload is the only accepted payload. Subscribers
	// filter out anything else, so the bus is a pure re-read signal.
	settingsInvalidatePayload = "changed"
)

// settingsInvalidationBus implements service.SettingsInvalidationBus on top of
// Redis Pub/Sub, mirroring the API-key auth-cache invalidation pattern but with
// a static payload and no message data.
type settingsInvalidationBus struct {
	rdb *redis.Client
}

// NewSettingsInvalidationBus creates the Redis-backed settings invalidation bus.
func NewSettingsInvalidationBus(rdb *redis.Client) service.SettingsInvalidationBus {
	return &settingsInvalidationBus{rdb: rdb}
}

// PublishSettingsInvalidation publishes a static notification to all instances.
func (b *settingsInvalidationBus) PublishSettingsInvalidation(ctx context.Context) error {
	return b.rdb.Publish(ctx, settingsInvalidateChannel, settingsInvalidatePayload).Err()
}

// SubscribeSettingsInvalidation subscribes to settings-change notifications,
// invoking handler only for the static expected payload. It verifies the
// subscription, blocks until ctx is done, closes the pubsub cleanly, and
// returns failures so the service-level retry/backoff loop can react.
func (b *settingsInvalidationBus) SubscribeSettingsInvalidation(ctx context.Context, handler func()) error {
	pubsub := b.rdb.Subscribe(ctx, settingsInvalidateChannel)

	// Verify subscription is working.
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return fmt.Errorf("subscribe to settings invalidation: %w", err)
	}

	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Warning: failed to close settings invalidation pubsub: %v", err)
		}
	}()
	service.NotifySettingsSubscriptionReady(ctx)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return errors.New("settings invalidation pubsub channel closed")
			}
			if msg != nil && msg.Payload == settingsInvalidatePayload {
				handler()
			}
		}
	}
}
