//go:build embed

package web

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestInjectSiteTitle(t *testing.T) {
	t.Run("replaces_title_with_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), "<title>MyCustomSite - AI API Gateway</title>")
		assert.NotContains(t, string(result), "Sub2API")
	})

	t.Run("returns_unchanged_when_site_name_empty", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":""}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_site_name_missing", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"other_field":"value"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_invalid_json", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{invalid json}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_no_title_tag", func(t *testing.T) {
		html := []byte(`<html><head></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("returns_unchanged_when_title_has_attributes", func(t *testing.T) {
		// The function looks for "<title>" literally, so attributes are not supported
		// This is acceptable since index.html uses plain <title> without attributes
		html := []byte(`<html><head><title lang="en">Sub2API</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"NewSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		// Should return unchanged since <title> with attributes is not matched
		assert.Equal(t, string(html), string(result))
	})

	t.Run("escapes_html_in_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API - AI API Gateway</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"</title><script>alert(1)</script><title>"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.NotContains(t, string(result), "<script>")
		assert.Contains(t, string(result), "&lt;/title&gt;&lt;script&gt;alert(1)&lt;/script&gt;&lt;title&gt;")
	})

	t.Run("escapes_ampersand_in_site_name", func(t *testing.T) {
		html := []byte(`<html><head><title>Sub2API</title></head><body></body></html>`)
		settingsJSON := []byte(`{"site_name":"A&B"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), "<title>A&amp;B - AI API Gateway</title>")
	})

	t.Run("preserves_rest_of_html", func(t *testing.T) {
		html := []byte(`<html><head><meta charset="UTF-8"><title>Sub2API</title><script src="app.js"></script></head><body><div id="app"></div></body></html>`)
		settingsJSON := []byte(`{"site_name":"TestSite"}`)

		result := injectSiteTitle(html, settingsJSON)

		assert.Contains(t, string(result), `<meta charset="UTF-8">`)
		assert.Contains(t, string(result), `<script src="app.js"></script>`)
		assert.Contains(t, string(result), `<div id="app"></div>`)
		assert.Contains(t, string(result), "<title>TestSite - AI API Gateway</title>")
	})
}

func TestInjectSiteFavicon(t *testing.T) {
	t.Run("replaces_favicon_with_site_logo", func(t *testing.T) {
		html := []byte(`<html><head><link rel="icon" type="image/png" href="/logo.png" /></head></html>`)
		settingsJSON := []byte(`{"site_logo":"https://example.com/custom-logo.png"}`)

		result := injectSiteFavicon(html, settingsJSON)

		assert.Contains(t, string(result), `<link rel="icon" href="https://example.com/custom-logo.png" />`)
		assert.NotContains(t, string(result), `/logo.png`)
	})

	t.Run("supports_relative_and_data_image_urls", func(t *testing.T) {
		html := []byte(`<link rel="icon" href="/logo.png" />`)

		assert.Contains(t, string(injectSiteFavicon(html, []byte(`{"site_logo":"/uploads/logo.svg"}`))), `/uploads/logo.svg`)
		assert.Contains(t, string(injectSiteFavicon(html, []byte(`{"site_logo":"data:image/png;base64,abc"}`))), `data:image/png;base64,abc`)
	})

	t.Run("rejects_unsafe_logo_urls", func(t *testing.T) {
		html := []byte(`<link rel="icon" href="/logo.png" />`)

		result := injectSiteFavicon(html, []byte(`{"site_logo":"javascript:alert(1)"}`))

		assert.Equal(t, string(html), string(result))
	})

	t.Run("escapes_logo_url_for_html", func(t *testing.T) {
		html := []byte(`<link rel="icon" href="/logo.png" />`)

		result := injectSiteFavicon(html, []byte(`{"site_logo":"https://example.com/logo.png?a=1&b=2"}`))

		assert.Contains(t, string(result), `a=1&amp;b=2`)
	})
}

func TestReplaceNoncePlaceholder(t *testing.T) {
	t.Run("replaces_single_placeholder", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">console.log('test');</script>`)
		nonce := "abc123xyz"

		result := replaceNoncePlaceholder(html, nonce)

		expected := `<script nonce="abc123xyz">console.log('test');</script>`
		assert.Equal(t, expected, string(result))
	})

	t.Run("replaces_multiple_placeholders", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">a</script><script nonce="__CSP_NONCE_VALUE__">b</script>`)
		nonce := "nonce123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, 2, strings.Count(string(result), `nonce="nonce123"`))
		assert.NotContains(t, string(result), NonceHTMLPlaceholder)
	})

	t.Run("handles_empty_nonce", func(t *testing.T) {
		html := []byte(`<script nonce="__CSP_NONCE_VALUE__">test</script>`)
		nonce := ""

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, `<script nonce="">test</script>`, string(result))
	})

	t.Run("no_placeholder_returns_unchanged", func(t *testing.T) {
		html := []byte(`<script>console.log('test');</script>`)
		nonce := "abc123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Equal(t, string(html), string(result))
	})

	t.Run("handles_empty_html", func(t *testing.T) {
		html := []byte(``)
		nonce := "abc123"

		result := replaceNoncePlaceholder(html, nonce)

		assert.Empty(t, result)
	})
}

func TestNonceHTMLPlaceholder(t *testing.T) {
	t.Run("constant_value", func(t *testing.T) {
		assert.Equal(t, "__CSP_NONCE_VALUE__", NonceHTMLPlaceholder)
	})
}

// mockSettingsProvider implements PublicSettingsProvider for testing
type mockSettingsProvider struct {
	settings any
	err      error
	called   int
}

func (m *mockSettingsProvider) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	m.called++
	return m.settings, m.err
}

func TestFrontendServer_InjectSettings(t *testing.T) {
	t.Run("injects_settings_with_nonce_placeholder", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"key": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{"test":"data"}`)
		result := server.injectSettings(settingsJSON)

		// Should contain the script with nonce placeholder
		assert.Contains(t, string(result), `<script nonce="__CSP_NONCE_VALUE__">`)
		assert.Contains(t, string(result), `window.__APP_CONFIG__={"test":"data"};`)
		assert.Contains(t, string(result), `CAP_SCRIPT_NONCE="__CSP_NONCE_VALUE__"`)
		assert.Contains(t, string(result), `CAP_CSS_NONCE="__CSP_NONCE_VALUE__"`)
		assert.Contains(t, string(result), `</script></head>`)
	})

	t.Run("injects_before_head_close", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"key": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{}`)
		result := server.injectSettings(settingsJSON)

		// Script should be injected before </head>
		headCloseIndex := bytes.Index(result, []byte("</head>"))
		scriptIndex := bytes.Index(result, []byte(`<script nonce="`))

		assert.True(t, scriptIndex < headCloseIndex, "script should be before </head>")
	})

	t.Run("handles_complex_settings", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]any{
				"nested": map[string]any{
					"array": []int{1, 2, 3},
				},
			},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		settingsJSON := []byte(`{"nested":{"array":[1,2,3]},"special":"<>&"}`)
		result := server.injectSettings(settingsJSON)

		assert.Contains(t, string(result), `window.__APP_CONFIG__={"nested":{"array":[1,2,3]},"special":"<>&"};`)
	})
}

func TestFrontendServer_ServeIndexHTML(t *testing.T) {
	t.Run("serves_html_with_nonce", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Create a gin context with nonce
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		// Set nonce in context (simulating SecurityHeaders middleware)
		testNonce := "test-nonce-12345"
		c.Set(middleware.CSPNonceKey, testNonce)

		server.serveIndexHTML(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

		body := w.Body.String()
		// Nonce placeholder should be replaced
		assert.NotContains(t, body, NonceHTMLPlaceholder)
		assert.Contains(t, body, `nonce="`+testNonce+`"`)
	})

	t.Run("caches_html_content", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// First request
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c1.Set(middleware.CSPNonceKey, "nonce1")

		server.serveIndexHTML(c1)
		assert.Equal(t, 1, provider.called)

		// Second request - should use cache
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c2.Set(middleware.CSPNonceKey, "nonce2")

		server.serveIndexHTML(c2)
		// Settings provider should not be called again
		assert.Equal(t, 1, provider.called)

		// But nonce should be different
		assert.Contains(t, w2.Body.String(), `nonce="nonce2"`)
	})

	t.Run("sets_etag_header_when_csp_disabled", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		server.serveIndexHTML(c)

		etag := w.Header().Get("ETag")
		assert.NotEmpty(t, etag)
		assert.True(t, strings.HasPrefix(etag, `"`))
		assert.True(t, strings.HasSuffix(etag, `"`))
	})

	t.Run("returns_304_for_matching_etag_when_csp_disabled", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Use a real router for proper 304 handling
		router := gin.New()
		router.Use(server.Middleware())

		// First request to populate cache and get ETag
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w1, req1)
		etag := w1.Header().Get("ETag")
		require.NotEmpty(t, etag)

		// Second request with If-None-Match
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("If-None-Match", etag)
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusNotModified, w2.Code)
		assert.Empty(t, w2.Body.String())
	})

	t.Run("sets_cache_control_header", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		server.serveIndexHTML(c)

		assert.Equal(t, indexHTMLCacheControl, w.Header().Get("Cache-Control"))
	})

	// 带 CSP nonce 的首页正文与响应头一一绑定，任何缓存重放都会让正文里的 nonce 过期，
	// 页面上全部内联脚本（含注入 window.CAP_SCRIPT_NONCE 的那段）被浏览器拦掉，
	// 进而让 Cap 验证码的 instrumentation srcdoc iframe 报 script-src 违规。
	t.Run("nonced_html_is_never_stored_or_revalidated", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		nonce := "nonce-request-1"
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CSPNonceKey, nonce)
			c.Next()
		})
		router.Use(server.Middleware())

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, noncedIndexHTMLCacheControl, w1.Header().Get("Cache-Control"))
		// No ETag means no conditional request can ever be built for this body.
		assert.Empty(t, w1.Header().Get("ETag"))
		assert.Contains(t, w1.Body.String(), `nonce="nonce-request-1"`)

		// Even a client that kept an ETag from an earlier (CSP-disabled) response must get
		// a full body carrying the current request's nonce — never a 304.
		nonce = "nonce-request-2"
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("If-None-Match", `"stale-etag"`)
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Contains(t, w2.Body.String(), `nonce="nonce-request-2"`)
		assert.NotContains(t, w2.Body.String(), "nonce-request-1")
	})

	t.Run("fallback_on_settings_error", func(t *testing.T) {
		provider := &mockSettingsProvider{
			err: context.DeadlineExceeded,
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// Invalidate cache to force settings fetch
		server.InvalidateCache()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "nonce123")

		server.serveIndexHTML(c)

		// Should still return 200 with base HTML
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	})
}

func TestFrontendServer_InvalidateCache(t *testing.T) {
	t.Run("invalidates_cache", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		// First request to populate cache
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c1.Set(middleware.CSPNonceKey, "nonce1")

		server.serveIndexHTML(c1)
		assert.Equal(t, 1, provider.called)

		// Invalidate cache
		server.InvalidateCache()

		// Update settings
		provider.settings = map[string]string{"test": "new_value"}

		// Second request should fetch new settings
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c2.Set(middleware.CSPNonceKey, "nonce2")

		server.serveIndexHTML(c2)
		assert.Equal(t, 2, provider.called)
	})

	t.Run("handles_nil_server", func(t *testing.T) {
		var server *FrontendServer
		// Should not panic
		assert.NotPanics(t, func() {
			server.InvalidateCache()
		})
	})

	t.Run("handles_nil_cache", func(t *testing.T) {
		server := &FrontendServer{}
		// Should not panic
		assert.NotPanics(t, func() {
			server.InvalidateCache()
		})
	})
}

func TestOverrideFilesCacheHeaders(t *testing.T) {
	t.Parallel()

	overrideDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(overrideDir, "assets"), 0o755))
	immutablePath := "assets/index-AbCd1234.js"
	require.NoError(t, os.WriteFile(filepath.Join(overrideDir, immutablePath), []byte("override"), 0o644))
	mutablePath := "logo.svg"
	require.NoError(t, os.WriteFile(filepath.Join(overrideDir, mutablePath), []byte("override"), 0o644))

	t.Run("frontend_server_rejects_immutable_asset_override", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+immutablePath, nil)

		server := &FrontendServer{overrideDir: overrideDir}
		assert.False(t, server.tryServeOverride(c, immutablePath))
		assert.Empty(t, w.Body.String())
		assert.Empty(t, w.Header().Get("Cache-Control"))
	})

	t.Run("legacy_rejects_immutable_asset_override", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+immutablePath, nil)

		assert.False(t, tryServeOverrideFile(c, overrideDir, immutablePath))
		assert.Empty(t, w.Body.String())
		assert.Empty(t, w.Header().Get("Cache-Control"))
	})

	t.Run("frontend_server_mutable_override_gets_no_cache", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+mutablePath, nil)

		server := &FrontendServer{overrideDir: overrideDir}
		assert.True(t, server.tryServeOverride(c, mutablePath))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		// c.File keeps Last-Modified so conditional requests still revalidate.
		assert.NotEmpty(t, w.Header().Get("Last-Modified"))
	})

	t.Run("legacy_mutable_override_gets_no_cache", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/"+mutablePath, nil)

		assert.True(t, tryServeOverrideFile(c, overrideDir, mutablePath))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		assert.NotEmpty(t, w.Header().Get("Last-Modified"))
	})
}

func TestFrontendServer_Middleware(t *testing.T) {
	t.Run("skips_api_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		apiPaths := []string{
			"/api/v1/users",
			"/models",
			"/v1/models",
			"/v1beta/chat",
			"/backend-api/codex/responses",
			"/backend-api/codex/responses/compact",
			"/antigravity/test",
			"/setup/init",
			"/health",
			"/responses",
			"/responses/compact",
		}

		for _, path := range apiPaths {
			t.Run(path, func(t *testing.T) {
				router := gin.New()
				router.Use(server.Middleware())
				nextCalled := false
				router.GET(path, func(c *gin.Context) {
					nextCalled = true
					c.String(http.StatusOK, "ok")
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.True(t, nextCalled, "next handler should be called for API route")
			})
		}
	})

	t.Run("skips_responses_compact_post_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())
		nextCalled := false
		router.POST("/responses/compact", func(c *gin.Context) {
			nextCalled = true
			c.String(http.StatusOK, `{"ok":true}`)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/responses/compact", strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.True(t, nextCalled, "next handler should be called for compact API route")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true}`, w.Body.String())
	})

	t.Run("skips_alpha_search_post_route", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())
		nextCalled := false
		router.POST("/alpha/search", func(c *gin.Context) {
			nextCalled = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/alpha/search", strings.NewReader(`{"model":"gpt-5.6-sol"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.True(t, nextCalled, "next handler should be called for alpha search API route")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"ok":true}`, w.Body.String())
	})

	t.Run("serves_index_for_spa_routes", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CSPNonceKey, "test-nonce")
			c.Next()
		})
		router.Use(server.Middleware())

		spaPaths := []string{
			"/",
			"/dashboard",
			"/users/123",
			"/settings/profile",
		}

		for _, path := range spaPaths {
			t.Run(path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
				// SPA fallback is index HTML, never a static asset: it must not
				// receive static cache-control or ETag headers.
				assert.Equal(t, noncedIndexHTMLCacheControl, w.Header().Get("Cache-Control"))
				assert.Empty(t, w.Header().Get("ETag"))
			})
		}
	})

	t.Run("serves_static_files", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		// Mutable release-owned files are stored-but-revalidated: no-cache plus
		// a deterministic strong ETag derived from the embedded content.
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "image/svg+xml")
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		mutableETag := w.Header().Get("ETag")
		assert.NotEmpty(t, mutableETag)
		assert.Equal(t, mutableETag, server.staticCache.ETag("logo.svg"))

		// A conditional If-None-Match request for a mutable embedded file gets
		// the standard 304 through http.FileServer, keeping the ETag.
		w304 := httptest.NewRecorder()
		req304 := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
		req304.Header.Set("If-None-Match", mutableETag)
		router.ServeHTTP(w304, req304)

		assert.Equal(t, http.StatusNotModified, w304.Code)
		assert.Empty(t, w304.Body.String())
		assert.Equal(t, mutableETag, w304.Header().Get("ETag"))

		// A stale conditional value falls through to a fresh 200.
		wStale := httptest.NewRecorder()
		reqStale := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
		reqStale.Header.Set("If-None-Match", `"stale-etag"`)
		router.ServeHTTP(wStale, reqStale)
		assert.Equal(t, http.StatusOK, wStale.Code)

		entries, err := fs.ReadDir(server.distFS, "assets")
		require.NoError(t, err)
		fingerprintedPath := ""
		for _, entry := range entries {
			candidate := "assets/" + entry.Name()
			if !entry.IsDir() && isFingerprintedEmbeddedAssetPath(candidate) {
				fingerprintedPath = candidate
				break
			}
		}
		require.NotEmpty(t, fingerprintedPath)

		assetWriter := httptest.NewRecorder()
		assetRequest := httptest.NewRequest(http.MethodGet, "/"+fingerprintedPath, nil)
		router.ServeHTTP(assetWriter, assetRequest)

		assert.Equal(t, http.StatusOK, assetWriter.Code)
		assert.Equal(t, staticAssetsCacheControl, assetWriter.Header().Get("Cache-Control"))
		// Successful immutable responses carry the policy ETag too, so
		// conditional and range requests can be evaluated against it.
		assert.Equal(t, server.staticCache.ETag(fingerprintedPath), assetWriter.Header().Get("ETag"))
	})
}

// TestFrontendServer_StaticCacheHeaderRemediation exercises the deferred
// static-cache-header design end to end through the full Gin middleware. The
// regression it guards: http.FileServer evaluates If-Match/If-None-Match
// against the ETag on the response header map, so cache headers placed there
// before serving used to leak onto error responses (a 412 carrying
// "public, max-age=31536000, immutable" poisoned downstream caches).
func TestFrontendServer_StaticCacheHeaderRemediation(t *testing.T) {
	provider := &mockSettingsProvider{settings: map[string]string{"test": "value"}}
	server, err := NewFrontendServer(provider)
	require.NoError(t, err)

	// Choose a real immutable asset from the embedded dist.
	entries, err := fs.ReadDir(server.distFS, "assets")
	require.NoError(t, err)
	fingerprintedPath := ""
	for _, entry := range entries {
		candidate := "assets/" + entry.Name()
		if !entry.IsDir() && isFingerprintedEmbeddedAssetPath(candidate) {
			fingerprintedPath = candidate
			break
		}
	}
	require.NotEmpty(t, fingerprintedPath)

	t.Run("immutable_asset_precondition_failure_is_412_without_static_cache_headers", func(t *testing.T) {
		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/"+fingerprintedPath, nil)
		req.Header.Set("If-Match", `"bogus"`)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusPreconditionFailed, w.Code)
		assert.Empty(t, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})

	t.Run("mutable_file_matching_inm_returns_304_with_no_cache_and_etag", func(t *testing.T) {
		router := gin.New()
		router.Use(server.Middleware())

		etag := server.staticCache.ETag("logo.svg")
		require.NotEmpty(t, etag)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
		req.Header.Set("If-None-Match", etag)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.String())
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		assert.Equal(t, etag, w.Header().Get("ETag"))
	})

	t.Run("immutable_asset_range_returns_206_without_static_cache_headers", func(t *testing.T) {
		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/"+fingerprintedPath, nil)
		req.Header.Set("Range", "bytes=0-1")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusPartialContent, w.Code)
		assert.Equal(t, "bytes", w.Header().Get("Accept-Ranges"))
		assert.NotEmpty(t, w.Header().Get("Content-Range"))
		assert.Empty(t, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})

	t.Run("directory_path_gets_explicit_404_no_store", func(t *testing.T) {
		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/", nil)
		router.ServeHTTP(w, req)

		// A trailing-slash directory path in the /assets/ namespace does not
		// resolve to a file, so it is treated like any other unknown asset
		// path: never the SPA shell, always the explicit non-cacheable 404
		// with no static ETag and a non-HTML body.
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, missingAssetCacheControl, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
		assert.NotContains(t, w.Header().Get("Content-Type"), "text/html")
	})

	t.Run("legacy_directory_path_gets_explicit_404_no_store", func(t *testing.T) {
		legacy := ServeEmbeddedFrontend()
		router := gin.New()
		router.Use(legacy)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/", nil)
		router.ServeHTTP(w, req)

		// The legacy path mirrors the main middleware: an unknown /assets/
		// path is an explicit non-cacheable 404, not the raw SPA index.
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, missingAssetCacheControl, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
		assert.NotContains(t, w.Header().Get("Content-Type"), "text/html")
	})

	t.Run("missing_asset_gets_explicit_404_no_store_without_static_etag", func(t *testing.T) {
		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/definitely-missing-abc123.js", nil)
		router.ServeHTTP(w, req)

		// Unknown /assets/ paths must never fall through to the SPA HTML
		// shell: an explicit non-cacheable 404 with no static ETag and a
		// non-HTML body keeps asset-style URLs from being stored or
		// negative-cached by a browser, proxy, or CDN.
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, missingAssetCacheControl, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
		assert.NotContains(t, w.Header().Get("Content-Type"), "text/html")
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("legacy_missing_asset_gets_explicit_404_no_store", func(t *testing.T) {
		legacy := ServeEmbeddedFrontend()
		router := gin.New()
		router.Use(legacy)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/definitely-missing-abc123.js", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, missingAssetCacheControl, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
		assert.NotContains(t, w.Header().Get("Content-Type"), "text/html")
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("missing_non_asset_keeps_spa_fallback", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CSPNonceKey, "test-nonce")
			c.Next()
		})
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/some/unknown/spa/route", nil)
		router.ServeHTTP(w, req)

		// SPA fallback is preserved for non-asset routes.
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, noncedIndexHTMLCacheControl, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})
}

func TestEmbeddedFrontendBypassesBareVideoAPIRoutes(t *testing.T) {
	for _, path := range []string{
		"/videos/generations",
		"/videos/edits",
		"/videos/extensions",
		"/videos/request-123",
	} {
		require.True(t, shouldBypassEmbeddedFrontend(path), "path=%s", path)
	}
}

func TestNewFrontendServer(t *testing.T) {
	t.Run("creates_server_successfully", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.distFS)
		assert.NotNil(t, server.fileServer)
		assert.NotNil(t, server.baseHTML)
		assert.NotNil(t, server.cache)
		assert.Equal(t, provider, server.settings)
	})

	t.Run("reads_base_html", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		assert.NotEmpty(t, server.baseHTML)
		assert.Contains(t, string(server.baseHTML), "<!doctype html>")
	})
}

func TestHasEmbeddedFrontend(t *testing.T) {
	t.Run("returns_true_when_frontend_embedded", func(t *testing.T) {
		result := HasEmbeddedFrontend()
		assert.True(t, result)
	})
}

// Tests for legacy ServeEmbeddedFrontend function
func TestServeEmbeddedFrontend(t *testing.T) {
	t.Run("serves_static_files", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "image/svg+xml")
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		assert.NotEmpty(t, w.Header().Get("ETag"))
	})

	t.Run("serves_index_html_for_root", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("serves_index_html_for_spa_routes", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		spaPaths := []string{"/dashboard", "/users/123", "/settings"}

		for _, path := range spaPaths {
			t.Run(path, func(t *testing.T) {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
			})
		}
	})

	t.Run("skips_api_routes", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		apiPaths := []string{
			"/api/users",
			"/models",
			"/v1/models",
			"/v1beta/chat",
			"/backend-api/codex/responses",
			"/backend-api/codex/responses/compact",
			"/antigravity/test",
			"/setup/init",
			"/health",
			"/responses",
			"/responses/compact",
		}

		for _, path := range apiPaths {
			t.Run(path, func(t *testing.T) {
				nextCalled := false
				router := gin.New()
				router.Use(middleware)
				router.GET(path, func(c *gin.Context) {
					nextCalled = true
					c.String(http.StatusOK, "ok")
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				router.ServeHTTP(w, req)

				assert.True(t, nextCalled, "next handler should be called for API route")
			})
		}
	})
}

// Tests for HTMLCache
func TestHTMLCache(t *testing.T) {
	t.Run("new_cache_returns_nil", func(t *testing.T) {
		cache := NewHTMLCache()
		assert.Nil(t, cache.Get())
	})

	t.Run("set_and_get", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")
		settings := []byte(`{"key":"value"}`)
		cache.Set(html, settings)

		result := cache.Get()
		require.NotNil(t, result)
		assert.Equal(t, html, result.Content)
		assert.NotEmpty(t, result.ETag)
	})

	t.Run("invalidate_clears_cache", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")
		settings := []byte(`{"key":"value"}`)
		cache.Set(html, settings)

		require.NotNil(t, cache.Get())

		cache.Invalidate()

		assert.Nil(t, cache.Get())
	})

	t.Run("etag_changes_with_settings", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		html := []byte("<html><body>test</body></html>")

		cache.Set(html, []byte(`{"v":1}`))
		etag1 := cache.Get().ETag

		cache.Invalidate()
		cache.Set(html, []byte(`{"v":2}`))
		etag2 := cache.Get().ETag

		assert.NotEqual(t, etag1, etag2)
	})

	t.Run("etag_format", func(t *testing.T) {
		cache := NewHTMLCache()
		cache.SetBaseHTML([]byte("<html></html>"))

		cache.Set([]byte("<html></html>"), []byte(`{}`))
		result := cache.Get()

		// ETag should be quoted
		assert.True(t, strings.HasPrefix(result.ETag, `"`))
		assert.True(t, strings.HasSuffix(result.ETag, `"`))
		// Should contain dash separator
		assert.Contains(t, result.ETag[1:len(result.ETag)-1], "-")
	})
}

// TestHTMLCacheBoundedStaleness exercises the 60-second TTL fallback through
// the package-private clock seam: entries are served before expiry, treated as
// misses after expiry, the window resets on every Set, and explicit
// Invalidate still clears a fresh entry.
func TestHTMLCacheBoundedStaleness(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fakeNow := start
	cache := newHTMLCache(func() time.Time { return fakeNow })
	cache.SetBaseHTML([]byte("<html></html>"))

	html := []byte("<html><body>ttl</body></html>")
	settings := []byte(`{"key":"value"}`)

	t.Run("hit_before_expiry", func(t *testing.T) {
		cache.Set(html, settings)

		result := cache.Get()
		require.NotNil(t, result)
		assert.Equal(t, html, result.Content)
	})

	t.Run("miss_after_expiry", func(t *testing.T) {
		// Advance past the 60-second TTL; the entry set in the previous
		// subtest must now be treated as a miss.
		fakeNow = fakeNow.Add(htmlCacheTTL + time.Second)

		assert.Nil(t, cache.Get())
	})

	t.Run("set_resets_ttl", func(t *testing.T) {
		// The entry from the previous subtest is still stale...
		assert.Nil(t, cache.Get())

		// ...but a fresh Set restarts the window even though the clock has
		// not moved, so the entry is immediately readable again.
		cache.Set(html, settings)
		require.NotNil(t, cache.Get())
	})

	t.Run("invalidate_clears_even_when_fresh", func(t *testing.T) {
		require.NotNil(t, cache.Get())

		cache.Invalidate()

		assert.Nil(t, cache.Get())
	})
}

// blockingSettingsProvider blocks each settings read until release is closed
// and reports entry through entered, so tests can interleave an Invalidate
// between the cache-miss check and the render.
type blockingSettingsProvider struct {
	mu          sync.Mutex
	settings    any
	called      int
	release     chan struct{}
	entered     chan struct{}
	enteredOnce sync.Once
}

func (m *blockingSettingsProvider) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	m.mu.Lock()
	m.called++
	// Snapshot the settings at entry: the blocked read must hold the value
	// that was current when the read started, not whatever gets written after
	// the test releases it.
	settings := m.settings
	m.mu.Unlock()
	m.enteredOnce.Do(func() { close(m.entered) })
	select {
	case <-m.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return settings, nil
}

// TestHTMLCache_SetIfCurrent_GenerationGuard verifies the conditional store: a
// render captured against a stale generation is refused (and never clears a
// fresh entry), while a render on the current generation is stored with an ETag
// that matches the exact body/settings pair passed in.
func TestHTMLCache_SetIfCurrent_GenerationGuard(t *testing.T) {
	cache := NewHTMLCache()
	cache.SetBaseHTML([]byte("<html></html>"))

	oldGen := cache.generation()
	require.Zero(t, oldGen)

	cache.Invalidate()
	require.Equal(t, oldGen+1, cache.generation())

	// A render started before the invalidation must not be stored.
	require.Nil(t, cache.SetIfCurrent([]byte("old"), []byte(`{"v":"old"}`), oldGen))
	require.Nil(t, cache.Get())

	// A render on the current generation is stored, and its ETag matches the
	// exact body/settings pair passed in.
	currentGen := cache.generation()
	stored := cache.SetIfCurrent([]byte("new"), []byte(`{"v":"new"}`), currentGen)
	require.NotNil(t, stored)
	require.NotNil(t, cache.Get())
	require.Equal(t, stored.ETag, cache.Get().ETag)
	require.Equal(t, "new", string(cache.Get().Content))
}

// TestFrontendServer_StaleRenderDoesNotResurrectAfterInvalidate is the
// blocked-provider concurrency regression for the stale-render resurrection
// race: read old settings -> InvalidateCache -> release the old read. The
// in-flight request may serve the body it already read, but it must not restore
// that stale render into the shared HTMLCache (and must not hand out an ETag
// for a body that will not be served again); the next request must re-read and
// render fresh settings.
func TestFrontendServer_StaleRenderDoesNotResurrectAfterInvalidate(t *testing.T) {
	provider := &blockingSettingsProvider{
		settings: map[string]string{"v": "old"},
		release:  make(chan struct{}),
		entered:  make(chan struct{}),
	}
	server, err := NewFrontendServer(provider)
	require.NoError(t, err)

	// Request 1 misses the cache and blocks inside the settings read while
	// holding the pre-invalidate generation.
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		server.serveIndexHTML(c1)
	}()
	select {
	case <-provider.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request never entered the settings read")
	}

	// Settings change while request 1's read is in flight.
	server.InvalidateCache()
	provider.mu.Lock()
	provider.settings = map[string]string{"v": "new"}
	provider.mu.Unlock()
	close(provider.release)
	<-done1

	// The in-flight request may serve the body it already read...
	require.Equal(t, http.StatusOK, w1.Code)
	require.Contains(t, w1.Body.String(), `"v":"old"`)
	// ...but it must never restore the stale render into the shared cache, and
	// no ETag may be emitted for a body the cache will never serve again.
	require.Empty(t, w1.Header().Get("ETag"))
	require.Nil(t, server.cache.Get())

	// Request 2 re-reads and renders fresh settings, then caches them.
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	server.serveIndexHTML(c2)

	require.Equal(t, http.StatusOK, w2.Code)
	require.Contains(t, w2.Body.String(), `"v":"new"`)
	require.NotNil(t, server.cache.Get())
	require.Equal(t, 2, provider.called)
}

// Benchmark tests
func BenchmarkReplaceNoncePlaceholder(b *testing.B) {
	html := []byte(`<!DOCTYPE html><html><head><script nonce="__CSP_NONCE_VALUE__">window.__APP_CONFIG__={"test":"data"};window.CAP_SCRIPT_NONCE="__CSP_NONCE_VALUE__";window.CAP_CSS_NONCE="__CSP_NONCE_VALUE__";</script></head><body></body></html>`)
	nonce := "abcdefghijklmnop123456=="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replaceNoncePlaceholder(html, nonce)
	}
}

func BenchmarkFrontendServerServeIndexHTML(b *testing.B) {
	provider := &mockSettingsProvider{
		settings: map[string]string{"test": "value"},
	}

	server, _ := NewFrontendServer(provider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CSPNonceKey, "test-nonce")

		server.serveIndexHTML(c)
	}
}
