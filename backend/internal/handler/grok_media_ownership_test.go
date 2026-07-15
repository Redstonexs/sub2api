//go:build unit

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type grokMediaOwnerHandlerCache struct {
	bindings map[string]int64
}

func (s *grokMediaOwnerHandlerCache) GetSessionAccountID(_ context.Context, _ int64, sessionHash string) (int64, error) {
	if accountID, ok := s.bindings[sessionHash]; ok {
		return accountID, nil
	}
	return 0, errors.New("session binding not found")
}

func (s *grokMediaOwnerHandlerCache) SetSessionAccountID(_ context.Context, _ int64, sessionHash string, accountID int64, _ time.Duration) error {
	if s.bindings == nil {
		s.bindings = make(map[string]int64)
	}
	s.bindings[sessionHash] = accountID
	return nil
}

func (s *grokMediaOwnerHandlerCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *grokMediaOwnerHandlerCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

type grokMediaOwnerHandlerAccountRepo struct {
	service.AccountRepository
	account service.Account
}

func (r *grokMediaOwnerHandlerAccountRepo) GetByID(_ context.Context, accountID int64) (*service.Account, error) {
	if accountID != r.account.ID {
		return nil, errors.New("account not found")
	}
	account := r.account
	return &account, nil
}

func (r *grokMediaOwnerHandlerAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	if platform != r.account.Platform {
		return nil, nil
	}
	return []service.Account{r.account}, nil
}

type grokMediaOwnerHandlerUpstream struct {
	calls int
}

func (u *grokMediaOwnerHandlerUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	body := `{"id":"video-request-owner","status":"completed"}`
	if req.Method == http.MethodPost {
		body = `{"request_id":"video-request-owner"}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func (u *grokMediaOwnerHandlerUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func newGrokMediaOwnerHandlerForTest(t *testing.T, groupID int64, cache service.GatewayCache, upstream service.HTTPUpstream) (*OpenAIGatewayHandler, *service.OpenAIGatewayService) {
	t.Helper()
	cfg := &config.Config{RunMode: config.RunModeSimple}
	accountRepo := &grokMediaOwnerHandlerAccountRepo{account: service.Account{
		ID:          88,
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"api_key": "test-key", "base_url": "https://grok.example.test/v1"},
	}}
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, upstream, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	return NewOpenAIGatewayHandler(
		gatewayService,
		service.NewConcurrencyService(nil),
		billingCacheService,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
		nil,
		nil,
		nil,
		nil,
		cfg,
	), gatewayService
}

func setGrokMediaOwnerTestIdentity(c *gin.Context, groupID, userID int64) {
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      userID + 300,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			Platform:             service.PlatformGrok,
			AllowImageGeneration: true,
		},
		User: &service.User{ID: userID},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
}

func TestGrokVideoStatusRejectsAnotherUsersRequestBeforeScheduling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(71)
	cache := &grokMediaOwnerHandlerCache{}
	h, gatewayService := newGrokMediaOwnerHandlerForTest(t, groupID, cache, &grokMediaOwnerHandlerUpstream{})
	require.NoError(t, gatewayService.BindGrokMediaVideoRequestOwner(context.Background(), &groupID, "video-request-owner", 101))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/video-request-owner", nil)
	c.Params = gin.Params{{Key: "request_id", Value: "video-request-owner"}}
	setGrokMediaOwnerTestIdentity(c, groupID, 202)

	h.GrokVideoStatus(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "not_found_error", gjson.GetBytes(recorder.Body.Bytes(), "error.type").String())
}

func TestGrokVideoGenerationBindsOwnerForStatusPolling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(72)
	cache := &grokMediaOwnerHandlerCache{}
	upstream := &grokMediaOwnerHandlerUpstream{}
	h, gatewayService := newGrokMediaOwnerHandlerForTest(t, groupID, cache, upstream)

	generationRecorder := httptest.NewRecorder()
	generationContext, _ := gin.CreateTestContext(generationRecorder)
	generationContext.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", strings.NewReader(`{"model":"grok-imagine-video-1.5","prompt":"test"}`))
	generationContext.Request.Header.Set("Content-Type", "application/json")
	setGrokMediaOwnerTestIdentity(generationContext, groupID, 101)

	h.GrokVideoGeneration(generationContext)

	require.Equal(t, http.StatusOK, generationRecorder.Code)
	require.True(t, gatewayService.IsGrokMediaVideoRequestOwnedBy(context.Background(), &groupID, "video-request-owner", 101))
	require.False(t, gatewayService.IsGrokMediaVideoRequestOwnedBy(context.Background(), &groupID, "video-request-owner", 202))

	attackerRecorder := httptest.NewRecorder()
	attackerContext, _ := gin.CreateTestContext(attackerRecorder)
	attackerContext.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/video-request-owner", nil)
	attackerContext.Params = gin.Params{{Key: "request_id", Value: "video-request-owner"}}
	setGrokMediaOwnerTestIdentity(attackerContext, groupID, 202)

	h.GrokVideoStatus(attackerContext)

	require.Equal(t, http.StatusNotFound, attackerRecorder.Code)
	require.Equal(t, 1, upstream.calls)

	ownerRecorder := httptest.NewRecorder()
	ownerContext, _ := gin.CreateTestContext(ownerRecorder)
	ownerContext.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/video-request-owner", nil)
	ownerContext.Params = gin.Params{{Key: "request_id", Value: "video-request-owner"}}
	setGrokMediaOwnerTestIdentity(ownerContext, groupID, 101)

	h.GrokVideoStatus(ownerContext)

	require.Equal(t, http.StatusOK, ownerRecorder.Code)
	require.Equal(t, "completed", gjson.GetBytes(ownerRecorder.Body.Bytes(), "status").String())
	require.Equal(t, 2, upstream.calls)
}
