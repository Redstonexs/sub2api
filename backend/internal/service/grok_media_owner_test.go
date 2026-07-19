//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type grokMediaOwnerCacheStub struct {
	bindings map[string]int64
	ttls     map[string]time.Duration
}

func (s *grokMediaOwnerCacheStub) GetSessionAccountID(_ context.Context, _ int64, sessionHash string) (int64, error) {
	if accountID, ok := s.bindings[sessionHash]; ok {
		return accountID, nil
	}
	return 0, errors.New("session binding not found")
}

func (s *grokMediaOwnerCacheStub) SetSessionAccountID(_ context.Context, _ int64, sessionHash string, accountID int64, ttl time.Duration) error {
	if s.bindings == nil {
		s.bindings = make(map[string]int64)
	}
	if s.ttls == nil {
		s.ttls = make(map[string]time.Duration)
	}
	s.bindings[sessionHash] = accountID
	s.ttls[sessionHash] = ttl
	return nil
}

func (s *grokMediaOwnerCacheStub) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *grokMediaOwnerCacheStub) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func TestOpenAIGatewayServiceGrokMediaVideoRequestOwner(t *testing.T) {
	groupID := int64(44)
	cache := &grokMediaOwnerCacheStub{}
	svc := &OpenAIGatewayService{cache: cache}

	require.NoError(t, svc.BindGrokMediaVideoRequestOwner(context.Background(), &groupID, "video-request-1", 101))
	require.True(t, svc.IsGrokMediaVideoRequestOwnedBy(context.Background(), &groupID, "video-request-1", 101))
	require.False(t, svc.IsGrokMediaVideoRequestOwnedBy(context.Background(), &groupID, "video-request-1", 202))
	require.False(t, svc.IsGrokMediaVideoRequestOwnedBy(context.Background(), &groupID, "unknown-video-request", 101))
	require.Equal(t, grokMediaVideoRequestOwnerTTL, cache.ttls[grokMediaVideoRequestOwnerSessionHash("video-request-1")])
	require.NotEqual(t, GrokMediaVideoRequestSessionHash("video-request-1", 101, 1), grokMediaVideoRequestOwnerSessionHash("video-request-1"))
}

func TestGrokMediaVideoRequestOwnerSessionHashUsesSHA256(t *testing.T) {
	require.Equal(t, "grok-video-owner:7ca13cd22a497db592e8115ad6661c8f743d6280382c2806b115171ce259442b", grokMediaVideoRequestOwnerSessionHash("video-request-1"))
}
