//go:build unit

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFingerprintedEmbeddedAssetPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "fingerprinted_js", path: "assets/index-AbCd1234.js", want: true},
		{name: "fingerprinted_css", path: "assets/app-a1B2c3D4.css", want: true},
		{name: "fingerprinted_url_safe_hash", path: "assets/app-aB1-2_Cd.css", want: true},
		{name: "nested_fingerprinted_asset", path: "assets/vendor/chunk-AbCd1234.js", want: true},
		{name: "leading_slash_fingerprinted_asset", path: "/assets/index-AbCd1234.js", want: true},
		{name: "unhashed_asset", path: "assets/index.js", want: false},
		{name: "short_suffix", path: "assets/index-abc123.js", want: false},
		{name: "logo", path: "logo.png", want: false},
		{name: "favicon", path: "favicon.ico", want: false},
		{name: "fingerprint_outside_assets", path: "downloads/index-AbCd1234.js", want: false},
		{name: "index_html", path: "index.html", want: false},
		{name: "spa_route", path: "dashboard", want: false},
		{name: "assets_prefix_only", path: "assets", want: false},
		{name: "similar_name", path: "assets-backup/x.js", want: false},
		{name: "empty", path: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isFingerprintedEmbeddedAssetPath(tc.path))
		})
	}
}

// quotedContentETag mirrors the policy's deterministic ETag derivation so tests
// can assert the exact value the policy must produce.
func quotedContentETag(data []byte) string {
	hash := sha256.Sum256(data)
	return `"` + hex.EncodeToString(hash[:16]) + `"`
}

func TestNewStaticCachePolicy(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"index.html":             {Data: []byte("<!doctype html>")},
		"logo.png":               {Data: []byte("png-bytes")},
		"assets/app-AbCd1234.js": {Data: []byte("console.log(1);")},
		"assets/vendor/deep.css": {Data: []byte("body{}")},
	}

	policy, err := NewStaticCachePolicy(fsys)
	require.NoError(t, err)
	require.NotNil(t, policy)

	t.Run("produces_deterministic_quoted_etags", func(t *testing.T) {
		t.Parallel()
		logoETag := policy.ETag("logo.png")
		assert.Equal(t, quotedContentETag([]byte("png-bytes")), logoETag)
		assert.True(t, strings.HasPrefix(logoETag, `"`))
		assert.True(t, strings.HasSuffix(logoETag, `"`))
		// Same content hash regardless of how many times it is read back.
		assert.Equal(t, logoETag, policy.ETag("/logo.png"))
	})

	t.Run("tracks_every_regular_file", func(t *testing.T) {
		t.Parallel()
		assert.NotEmpty(t, policy.ETag("index.html"))
		assert.NotEmpty(t, policy.ETag("assets/app-AbCd1234.js"))
		assert.NotEmpty(t, policy.ETag("assets/vendor/deep.css"))
	})

	t.Run("different_content_yields_different_etag", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, policy.ETag("logo.png"), policy.ETag("assets/app-AbCd1234.js"))
	})

	t.Run("untracked_paths_have_no_etag", func(t *testing.T) {
		t.Parallel()
		// SPA routes, missing files, and directories are never tracked.
		assert.Empty(t, policy.ETag("dashboard"))
		assert.Empty(t, policy.ETag("missing.js"))
		assert.Empty(t, policy.ETag("assets"))
		assert.Empty(t, policy.ETag("assets/vendor"))
	})
}

func TestStaticCachePolicy_CacheControl(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"index.html":             {Data: []byte("<!doctype html>")},
		"logo.png":               {Data: []byte("png-bytes")},
		"assets/app-AbCd1234.js": {Data: []byte("console.log(1);")},
	}
	policy, err := NewStaticCachePolicy(fsys)
	require.NoError(t, err)

	t.Run("immutable_asset_gets_long_lived_cache_control", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, staticAssetsCacheControl, policy.cacheControlFor("assets/app-AbCd1234.js"))
	})

	t.Run("mutable_file_gets_no_cache", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, staticMutableCacheControl, policy.cacheControlFor("logo.png"))
	})
}

// TestStaticCachePolicy_Serve exercises the deferred-header response wrapper
// directly with a stub handler, covering every status the static file server
// can commit.
func TestStaticCachePolicy_Serve(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"index.html":                       {Data: []byte("<!doctype html>")},
		"logo.png":                         {Data: []byte("png-bytes")},
		"favicon.ico":                      {Data: []byte("ico")},
		"assets/app-AbCd1234.js":           {Data: []byte("console.log(1);")},
		"assets/vendor/chunk-aB1-2_Cd.css": {Data: []byte("body{}")},
	}
	policy, err := NewStaticCachePolicy(fsys)
	require.NoError(t, err)

	// statusHandler commits the given status before writing any body, the way
	// http.FileServer does (0 means implicit 200 via Write alone).
	statusHandler := func(status int, body []byte) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if status == 0 {
				_, _ = w.Write(body)
				return
			}
			w.WriteHeader(status)
			if status >= 200 && status < 300 {
				_, _ = w.Write(body)
			}
		})
	}

	cases := []struct {
		name     string
		path     string
		status   int
		wantCC   string // expected Cache-Control on the wire ("" = absent)
		wantETag bool   // whether the policy ETag must reach the wire
	}{
		{
			name:     "mutable_200_gets_no_cache_and_etag",
			path:     "logo.png",
			status:   http.StatusOK,
			wantCC:   staticMutableCacheControl,
			wantETag: true,
		},
		{
			name:     "mutable_implicit_200_via_write",
			path:     "favicon.ico",
			status:   0,
			wantCC:   staticMutableCacheControl,
			wantETag: true,
		},
		{
			name:     "immutable_200_gets_immutable_cache_and_etag",
			path:     "assets/app-AbCd1234.js",
			status:   http.StatusOK,
			wantCC:   staticAssetsCacheControl,
			wantETag: true,
		},
		{
			name:     "mutable_304_keeps_no_cache_and_etag",
			path:     "logo.png",
			status:   http.StatusNotModified,
			wantCC:   staticMutableCacheControl,
			wantETag: true,
		},
		{
			name:     "immutable_206_drops_static_cache_headers",
			path:     "assets/app-AbCd1234.js",
			status:   http.StatusPartialContent,
			wantCC:   "",
			wantETag: false,
		},
		{
			name:     "mutable_301_drops_static_cache_headers",
			path:     "logo.png",
			status:   http.StatusMovedPermanently,
			wantCC:   "",
			wantETag: false,
		},
		{
			name:     "mutable_404_drops_static_cache_headers",
			path:     "logo.png",
			status:   http.StatusNotFound,
			wantCC:   "",
			wantETag: false,
		},
		{
			name:     "mutable_405_drops_static_cache_headers",
			path:     "logo.png",
			status:   http.StatusMethodNotAllowed,
			wantCC:   "",
			wantETag: false,
		},
		{
			name:     "mutable_412_drops_static_cache_headers",
			path:     "logo.png",
			status:   http.StatusPreconditionFailed,
			wantCC:   "",
			wantETag: false,
		},
		{
			name:     "mutable_416_drops_static_cache_headers",
			path:     "logo.png",
			status:   http.StatusRequestedRangeNotSatisfiable,
			wantCC:   "",
			wantETag: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/"+tc.path, nil)
			policy.Serve(w, req, tc.path, statusHandler(tc.status, []byte("hello")))

			wantStatus := tc.status
			if wantStatus == 0 {
				wantStatus = http.StatusOK
			}
			assert.Equal(t, wantStatus, w.Code)
			assert.Equal(t, tc.wantCC, w.Header().Get("Cache-Control"))
			if tc.wantETag {
				assert.Equal(t, policy.ETag(tc.path), w.Header().Get("ETag"))
			} else {
				assert.Empty(t, w.Header().Get("ETag"))
			}
		})
	}

	t.Run("etag_is_visible_to_handler_for_precondition_eval", func(t *testing.T) {
		t.Parallel()
		// http.FileServer evaluates If-Match/If-None-Match/If-Range against
		// the header map handed to it, so the policy ETag must already be
		// present when the handler runs.
		seen := ""
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = w.Header().Get("ETag")
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		policy.Serve(w, req, "logo.png", handler)
		assert.Equal(t, policy.ETag("logo.png"), seen)
	})

	t.Run("untracked_path_is_served_without_wrapper", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=60")
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets", nil)
		policy.Serve(w, req, "assets", handler)
		// The handler's headers pass through untouched: no policy headers, no
		// stripping.
		assert.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})

	t.Run("index_html_is_served_without_wrapper", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
		policy.Serve(w, req, "index.html", handler)
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})

	t.Run("preserves_unrelated_outer_headers", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		w.Header().Set("X-Outer", "kept")
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		policy.Serve(w, req, "logo.png", handler)
		assert.Equal(t, "kept", w.Header().Get("X-Outer"))
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
		assert.Equal(t, policy.ETag("logo.png"), w.Header().Get("ETag"))
	})

	t.Run("non_200_status_strips_pre_existing_cache_headers", func(t *testing.T) {
		t.Parallel()
		// Outer middleware may have placed cache headers on the underlying
		// writer before the wrapper ran; an error response must strip them.
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPreconditionFailed)
		})
		w := httptest.NewRecorder()
		w.Header().Set("Cache-Control", "public, max-age=999")
		w.Header().Set("ETag", `"inner"`)
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		policy.Serve(w, req, "logo.png", handler)
		assert.Equal(t, http.StatusPreconditionFailed, w.Code)
		assert.Empty(t, w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("ETag"))
	})

	t.Run("repeated_writeheader_is_safe", func(t *testing.T) {
		t.Parallel()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.WriteHeader(http.StatusInternalServerError)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		policy.Serve(w, req, "logo.png", handler)
		// The first WriteHeader wins.
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, staticMutableCacheControl, w.Header().Get("Cache-Control"))
	})

	t.Run("nil_policy_is_noop", func(t *testing.T) {
		t.Parallel()
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		var policy *StaticCachePolicy
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		policy.Serve(w, req, "logo.png", handler)
		assert.True(t, called)
		assert.Empty(t, w.Header().Get("Cache-Control"))
	})

	t.Run("nil_handler_is_noop", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/logo.png", nil)
		assert.NotPanics(t, func() {
			policy.Serve(w, req, "logo.png", nil)
		})
	})
}
