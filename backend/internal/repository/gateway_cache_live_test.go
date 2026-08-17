package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGatewayCacheLiveCallIdentityAndController(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)
	otherInstance, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)
	record := &service.LiveCallRecord{
		CallID:                "call_secret",
		CallHash:              HashLiveCallID("call_secret"),
		AccountID:             11,
		APIKeyID:              22,
		UserID:                33,
		GroupID:               44,
		LeaseID:               "lease",
		Model:                 "gpt-live-test",
		AttestationCiphertext: "encrypted-attestation",
		CreatedAt:             time.Now(),
		ExpiresAt:             time.Now().Add(time.Hour),
		Controller:            service.LiveControllerPending,
	}
	require.NoError(t, cache.SaveLiveCall(context.Background(), record, time.Hour))

	loaded, err := otherInstance.GetLiveCall(context.Background(), record.CallHash)
	require.NoError(t, err)
	require.Equal(t, record.CallID, loaded.CallID)
	require.Equal(t, record.AccountID, loaded.AccountID)
	require.Equal(t, record.AttestationCiphertext, loaded.AttestationCiphertext)

	claimed, err := cache.ClaimLiveController(context.Background(), record.CallHash, service.LiveControllerObserver, "observer-1")
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = cache.ClaimLiveController(context.Background(), record.CallHash, service.LiveControllerProxy, "proxy-1")
	require.NoError(t, err)
	require.True(t, claimed)
	controller, err := cache.GetLiveController(context.Background(), record.CallHash)
	require.NoError(t, err)
	require.Equal(t, service.LiveControllerProxy, controller)

	released, err := cache.ReleaseLiveController(context.Background(), record.CallHash, "proxy-1")
	require.NoError(t, err)
	require.True(t, released)
	closed, err := cache.MarkLiveCallClosed(context.Background(), record.CallHash, time.Hour)
	require.NoError(t, err)
	require.True(t, closed)
	closed, err = cache.MarkLiveCallClosed(context.Background(), record.CallHash, time.Hour)
	require.NoError(t, err)
	require.False(t, closed)
}

// The admission-time QoS snapshot must survive the Redis store round-trip:
// the observer path re-reads the record from the store before finalizing the
// usage row, so a snapshot that is not persisted would be lost.
func TestGatewayCacheLiveCallGroupQoSRecordRoundTrip(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)
	otherInstance, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)

	record := &service.LiveCallRecord{
		CallID:     "call_qos_roundtrip",
		CallHash:   HashLiveCallID("call_qos_roundtrip"),
		AccountID:  11,
		APIKeyID:   22,
		UserID:     33,
		GroupID:    44,
		LeaseID:    "lease-qos",
		Model:      "gpt-live-qos",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		Controller: service.LiveControllerPending,
		GroupQoSRecord: &service.GroupQoSRecordSnapshot{
			Tier:    2,
			Window:  "weekly",
			Effects: service.GroupQoSEffectRPM,
		},
	}
	require.NoError(t, cache.SaveLiveCall(context.Background(), record, time.Hour))

	loaded, err := otherInstance.GetLiveCall(context.Background(), record.CallHash)
	require.NoError(t, err)
	require.NotNil(t, loaded.GroupQoSRecord, "snapshot must survive the store round-trip")
	require.Equal(t, 2, loaded.GroupQoSRecord.Tier)
	require.Equal(t, "weekly", loaded.GroupQoSRecord.Window)
	require.Equal(t, service.GroupQoSEffectRPM, loaded.GroupQoSRecord.Effects)

	// A record saved without a snapshot loads with a nil snapshot.
	plain := &service.LiveCallRecord{
		CallID:    "call_plain",
		CallHash:  HashLiveCallID("call_plain"),
		AccountID: 11,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, cache.SaveLiveCall(context.Background(), plain, time.Hour))
	loadedPlain, err := otherInstance.GetLiveCall(context.Background(), plain.CallHash)
	require.NoError(t, err)
	require.Nil(t, loadedPlain.GroupQoSRecord)
}
