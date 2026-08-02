//go:build embed

package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	htmlpkg "html"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// NonceHTMLPlaceholder is the placeholder for nonce in HTML script tags
	NonceHTMLPlaceholder = "__CSP_NONCE_VALUE__"

	// indexHTMLCacheControl 用于不含 nonce 的首页：允许浏览器留存副本，但每次都必须回源校验。
	indexHTMLCacheControl = "no-cache"
	// noncedIndexHTMLCacheControl 用于含 per-request CSP nonce 的首页。
	// 这种正文只对"与它一起发出的那个 CSP 响应头"有效，任何一层缓存（浏览器 / 代理 / CDN）
	// 都不得留存重放，否则正文里的旧 nonce 会和新响应头里的新 nonce 对不上。
	noncedIndexHTMLCacheControl = "no-store, no-cache, must-revalidate, private"
	// missingAssetCacheControl 用于未知 /assets/ 路径的显式 404：正文绝不能被
	// 浏览器 / 代理 / CDN 留存，更不能被边缘负缓存成"这个资源不存在"的条目，
	// 否则一次普通的旧版本引用就会变成永久坏链。
	missingAssetCacheControl = "no-store"
)

//go:embed all:dist
var frontendFS embed.FS

// PublicSettingsProvider is an interface to fetch public settings
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the embedded frontend with settings injection
type FrontendServer struct {
	distFS      fs.FS
	fileServer  http.Handler
	staticCache *StaticCachePolicy
	baseHTML    []byte
	cache       *HTMLCache
	settings    PublicSettingsProvider
	overrideDir string // local file override directory
}

// NewFrontendServer creates a new frontend server with settings injection
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil, err
	}

	// Read base HTML once
	file, err := distFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	baseHTML, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	staticCache, err := NewStaticCachePolicy(distFS)
	if err != nil {
		return nil, err
	}

	return &FrontendServer{
		distFS:      distFS,
		fileServer:  http.FileServer(http.FS(distFS)),
		staticCache: staticCache,
		baseHTML:    baseHTML,
		cache:       cache,
		settings:    settingsProvider,
		overrideDir: filepath.Join("data", "public"),
	}, nil
}

// InvalidateCache invalidates the HTML cache (call when settings change)
func (s *FrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware returns the Gin middleware handler
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// For index.html or SPA routes, serve with injected settings
		if cleanPath == "index.html" || !s.fileExists(cleanPath) {
			// An unknown /assets/ path must never fall through to the SPA
			// HTML shell: an HTML body served (and possibly cached) under an
			// asset-style key, or negative-cached by an edge, would poison
			// asset URLs. Serve an explicit non-cacheable 404 instead.
			if isMissingAssetPath(cleanPath) {
				serveMissingAsset(c)
				return
			}
			s.serveIndexHTML(c)
			return
		}

		// Try local override first (immutable assets can never be shadowed)
		if s.tryServeOverride(c, cleanPath) {
			return
		}

		// Serve static files normally. The policy applies long-lived cache
		// headers to hashed assets and no-cache + ETag to mutable files, but
		// only on successful 200/304 responses: error/redirect responses such
		// as 412 or 206 never carry static cache headers.
		s.staticCache.Serve(c.Writer, c.Request, cleanPath, s.fileServer)
		c.Abort()
	}
}

func (s *FrontendServer) fileExists(path string) bool {
	file, err := s.distFS.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

// tryServeOverride checks if a local override file exists and serves it.
// Files in overrideDir take precedence over embedded mutable files, but never
// over immutable Vite-hashed assets: their names are unique per release, so a
// local copy would only ever go stale.
func (s *FrontendServer) tryServeOverride(c *gin.Context, cleanPath string) bool {
	if s.overrideDir == "" {
		return false
	}
	// Immutable Vite-hashed assets can never be shadowed by data/public: their
	// names are unique per release, so a local copy would only ever go stale.
	if isFingerprintedEmbeddedAssetPath(cleanPath) {
		return false
	}
	filePath := filepath.Join(s.overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	// Mutable overrides are stored-but-revalidated; c.File keeps Last-Modified
	// so If-Modified-Since/If-None-Match conditionals still get standard 304s.
	c.Header("Cache-Control", staticMutableCacheControl)
	c.File(filePath)
	c.Abort()
	return true
}

func (s *FrontendServer) serveIndexHTML(c *gin.Context) {
	// Get nonce from context (generated by SecurityHeaders middleware)
	nonce := middleware.GetNonceFromContext(c)

	// 首页正文里的 nonce 与本次响应的 Content-Security-Policy 头是一对一绑定的，必须同进同出。
	// 若正文可被缓存，浏览器下次重新校验时只会拿到新的 CSP 头（新 nonce），正文却仍是缓存里的
	// 旧 nonce（304 更极端：整个正文都来自缓存），页面上每一段内联脚本都会被拦掉——其中就包括
	// 注入 window.CAP_SCRIPT_NONCE / CAP_CSS_NONCE 的那段。Cap 验证码随后在做 instrumentation
	// 挑战时拿不到 nonce，其 srcdoc iframe 里的内联脚本便触发
	// "Executing inline script violates the following Content Security Policy directive 'script-src ...'"。
	// 所以只要带 nonce：不下发 ETag、不回 304，并用 no-store 阻止浏览器/代理/CDN 留存这份正文。
	nonced := nonce != ""
	cacheControl := indexHTMLCacheControl
	if nonced {
		cacheControl = noncedIndexHTMLCacheControl
	}

	// Check cache first
	cached := s.cache.Get()
	if cached != nil {
		// Check If-None-Match for 304 response
		if !nonced {
			if match := c.GetHeader("If-None-Match"); match == cached.ETag {
				c.Status(http.StatusNotModified)
				c.Abort()
				return
			}
			c.Header("ETag", cached.ETag)
		}

		// Replace nonce placeholder with actual nonce before serving
		content := replaceNoncePlaceholder(cached.Content, nonce)

		c.Header("Cache-Control", cacheControl)
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		c.Abort()
		return
	}

	// Cache miss - fetch settings and render. Capture the settings generation
	// BEFORE the provider read: if an Invalidate lands while the read is in
	// flight, the render below must not restore its (now stale) HTML into the
	// shared cache.
	gen := s.cache.generation()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		// Fallback: serve without injection
		c.Header("Cache-Control", cacheControl)
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		// Fallback: serve without injection
		c.Header("Cache-Control", cacheControl)
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	rendered := s.injectSettings(settingsJSON)
	// The generation guard keeps this in-flight render from resurrecting stale
	// HTML after a settings update: if the generation moved, nothing is stored
	// and the request simply serves the body it already read.
	stored := s.cache.SetIfCurrent(rendered, settingsJSON, gen)

	// Replace nonce placeholder with actual nonce before serving
	content := replaceNoncePlaceholder(rendered, nonce)

	// The ETag must always describe the exact body being served. Only the
	// generation-guarded store's own entry qualifies: when the entry was not
	// stored (generation changed mid-render) no ETag is emitted, so a client
	// can never build a conditional request against a body that will not be
	// served again.
	if !nonced && stored != nil {
		c.Header("ETag", stored.ETag)
	}
	c.Header("Cache-Control", cacheControl)
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func (s *FrontendServer) injectSettings(settingsJSON []byte) []byte {
	// Create the script tag to inject with nonce placeholder
	// The placeholder will be replaced with actual nonce at request time
	// Also inject CAP_SCRIPT_NONCE / CAP_CSS_NONCE so the Cap captcha widget can stamp
	// nonces onto its dynamically created <script> and <style> elements.
	// The NonceHTMLPlaceholder inside the JS string literal will be replaced with the
	// real per-request nonce by replaceNoncePlaceholder, just like the nonce= attribute.
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">` +
		`window.__APP_CONFIG__=` + string(settingsJSON) + `;` +
		`window.CAP_SCRIPT_NONCE="` + NonceHTMLPlaceholder + `";` +
		`window.CAP_CSS_NONCE="` + NonceHTMLPlaceholder + `";` +
		`</script>`)

	// Inject before </head>
	headClose := []byte("</head>")
	result := bytes.Replace(s.baseHTML, headClose, append(script, headClose...), 1)

	// Apply custom branding before the browser paints the static defaults.
	result = injectSiteTitle(result, settingsJSON)
	result = injectSiteFavicon(result, settingsJSON)

	return result
}

// injectSiteFavicon replaces the static favicon with a configured, browser-safe image URL.
func injectSiteFavicon(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteLogo string `json:"site_logo"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil {
		return html
	}

	logoURL := safeImageURL(cfg.SiteLogo)
	if logoURL == "" {
		return html
	}

	linkStart := bytes.Index(html, []byte(`<link rel="icon"`))
	if linkStart == -1 {
		return html
	}
	linkEndOffset := bytes.IndexByte(html[linkStart:], '>')
	if linkEndOffset == -1 {
		return html
	}
	linkEnd := linkStart + linkEndOffset + 1
	replacement := []byte(`<link rel="icon" href="` + htmlpkg.EscapeString(logoURL) + `" />`)

	var buf bytes.Buffer
	buf.Write(html[:linkStart])
	buf.Write(replacement)
	buf.Write(html[linkEnd:])
	return buf.Bytes()
}

func safeImageURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return trimmed
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "data:image/") {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return trimmed
}

// injectSiteTitle replaces the static <title> in HTML with the configured site name.
// This ensures the browser tab shows the correct title before JS executes.
func injectSiteTitle(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil || cfg.SiteName == "" {
		return html
	}

	// Find and replace the existing <title>...</title>
	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	newTitle := []byte("<title>" + htmlpkg.EscapeString(cfg.SiteName) + " - AI API Gateway</title>")
	var buf bytes.Buffer
	buf.Write(html[:titleStart])
	buf.Write(newTitle)
	buf.Write(html[titleEnd+len("</title>"):])
	return buf.Bytes()
}

// replaceNoncePlaceholder replaces the nonce placeholder with actual nonce value
func replaceNoncePlaceholder(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(NonceHTMLPlaceholder), []byte(nonce))
}

// ServeEmbeddedFrontend returns a middleware for serving embedded frontend
// This is the legacy function for backward compatibility when no settings provider is available
func ServeEmbeddedFrontend() gin.HandlerFunc {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))
	overrideDir := filepath.Join("data", "public")
	staticCache, err := NewStaticCachePolicy(distFS)
	if err != nil {
		panic("failed to build static cache policy: " + err.Error())
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			// Try local override first (immutable assets can never be shadowed)
			if tryServeOverrideFile(c, overrideDir, cleanPath) {
				return
			}
			staticCache.Serve(c.Writer, c.Request, cleanPath, fileServer)
			c.Abort()
			return
		}

		// An unknown /assets/ path must never fall through to the SPA HTML
		// shell; mirror the main FrontendServer middleware's explicit
		// non-cacheable 404.
		if isMissingAssetPath(cleanPath) {
			serveMissingAsset(c)
			return
		}

		serveIndexHTML(c, distFS)
	}
}

// tryServeOverrideFile is a standalone version of tryServeOverride for legacy usage.
func tryServeOverrideFile(c *gin.Context, overrideDir, cleanPath string) bool {
	if overrideDir == "" {
		return false
	}
	// Immutable Vite-hashed assets can never be shadowed by data/public.
	if isFingerprintedEmbeddedAssetPath(cleanPath) {
		return false
	}
	filePath := filepath.Join(overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.Header("Cache-Control", staticMutableCacheControl)
	c.File(filePath)
	c.Abort()
	return true
}

func shouldBypassEmbeddedFrontend(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/api/") ||
		strings.HasPrefix(trimmed, "/v1/") ||
		strings.HasPrefix(trimmed, "/v1beta/") ||
		strings.HasPrefix(trimmed, "/backend-api/") ||
		strings.HasPrefix(trimmed, "/antigravity/") ||
		strings.HasPrefix(trimmed, "/setup/") ||
		trimmed == "/health" ||
		trimmed == "/models" ||
		trimmed == "/responses" ||
		strings.HasPrefix(trimmed, "/responses/") ||
		trimmed == "/alpha/search" ||
		strings.HasPrefix(trimmed, "/images/") ||
		strings.HasPrefix(trimmed, "/videos/")
}

// isMissingAssetPath reports whether a cleaned URL path lives in the Vite
// /assets/ namespace. Combined with a failed existence check it identifies
// unknown asset paths that must be served as explicit non-cacheable 404s
// instead of falling through to the SPA HTML shell.
func isMissingAssetPath(cleanPath string) bool {
	return strings.HasPrefix(strings.TrimPrefix(cleanPath, "/"), "assets/")
}

// serveMissingAsset writes the explicit 404 for unknown /assets/ paths:
// Cache-Control: no-store so browsers, proxies, and CDNs can never store (or
// negative-cache) the response under an asset-style key, no static ETag, and a
// non-HTML body so the response can never be mistaken for the SPA shell or an
// immutable asset.
func serveMissingAsset(c *gin.Context) {
	c.Header("Cache-Control", missingAssetCacheControl)
	c.String(http.StatusNotFound, "asset not found")
	c.Abort()
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html")
		c.Abort()
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func HasEmbeddedFrontend() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}
