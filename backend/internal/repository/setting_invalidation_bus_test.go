package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSettingsInvalidationBus_PublishRoundTrips(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = client.Close() }()
	bus := NewSettingsInvalidationBus(client)

	require.NoError(t, bus.PublishSettingsInvalidation(context.Background()))
}

func TestSettingsInvalidationBus_SubscriberFiltersUnexpectedPayloadsAndStops(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = client.Close() }()
	bus := NewSettingsInvalidationBus(client)
	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan string, 8)
	returned := make(chan error, 1)
	go func() {
		returned <- bus.SubscribeSettingsInvalidation(ctx, func() { received <- settingsInvalidatePayload })
	}()

	// The static expected payload is delivered. The loop may publish several
	// times before the subscription is live (pre-subscription messages are
	// dropped by Redis), so drain residual duplicates after the handshake.
	require.Eventually(t, func() bool {
		require.NoError(t, client.Publish(context.Background(), settingsInvalidateChannel, settingsInvalidatePayload).Err())
		select {
		case payload := <-received:
			return payload == settingsInvalidatePayload
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	// Let any in-flight valid deliveries land, then drain them so the garbage
	// phase below observes a quiescent channel.
	time.Sleep(50 * time.Millisecond)
	for {
		select {
		case <-received:
		default:
			goto drained
		}
	}
drained:

	// Unexpected payloads on the same channel are filtered out: no data ever
	// reaches the handler, so a stray/garbage message cannot trigger work.
	require.NoError(t, client.Publish(context.Background(), settingsInvalidateChannel, "garbage").Err())
	require.NoError(t, client.Publish(context.Background(), settingsInvalidateChannel, "").Err())
	require.NoError(t, client.Publish(context.Background(), settingsInvalidateChannel, settingsInvalidatePayload+"x").Err())
	select {
	case payload := <-received:
		t.Fatalf("unexpected payload delivered: %q", payload)
	case <-time.After(150 * time.Millisecond):
	}

	select {
	case err := <-returned:
		t.Fatalf("subscriber returned while connection was active: %v", err)
	default:
	}
	cancel()
	select {
	case err := <-returned:
		require.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
	case <-time.After(time.Second):
		t.Fatal("subscriber did not stop after context cancellation")
	}
}
