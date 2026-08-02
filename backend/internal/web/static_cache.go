//go:build embed || unit

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Vite emits content-hashed filenames under assets/, so the backend can apply
// immutable caching without relying on a reverse proxy to classify paths.
const staticAssetsCacheControl = "public, max-age=31536000, immutable"

// staticMutableCacheControl lets browsers/CDNs store release-owned mutable
// static files but forces a revalidation round-trip before every reuse.
const staticMutableCacheControl = "no-cache"

// isFingerprintedEmbeddedAssetPath reports whether a cleaned URL path refers to
// a Vite asset whose filename contains the default eight-character build hash.
func isFingerprintedEmbeddedAssetPath(cleanPath string) bool {
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	if !strings.HasPrefix(cleanPath, "assets/") {
		return false
	}

	filename := path.Base(cleanPath)
	extension := path.Ext(filename)
	stem := strings.TrimSuffix(filename, extension)
	const fingerprintLength = 8
	delimiterIndex := len(stem) - fingerprintLength - 1
	if extension == "" || delimiterIndex < 1 || stem[delimiterIndex] != '-' {
		return false
	}

	// Vite hashes use URL-safe characters and are stable for immutable caching.
	fingerprint := stem[delimiterIndex+1:]
	for _, char := range fingerprint {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// StaticCachePolicy precomputes deterministic cache metadata for every regular
// file in the embedded dist/ subtree at construction time, so request handling
// never has to hash, open, or allocate for static assets.
type StaticCachePolicy struct {
	// etags maps a clean release-owned path to its quoted strong ETag, derived
	// from the file content's SHA-256 at startup.
	etags map[string]string
}

// NewStaticCachePolicy walks fsys once and records a quoted strong ETag for
// every regular file. Directories and non-regular entries are skipped.
func NewStaticCachePolicy(fsys fs.FS) (*StaticCachePolicy, error) {
	policy := &StaticCachePolicy{etags: make(map[string]string)}
	err := fs.WalkDir(fsys, ".", func(relPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		data, err := fs.ReadFile(fsys, relPath)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		policy.etags[relPath] = `"` + hex.EncodeToString(hash[:16]) + `"`
		return nil
	})
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// ETag returns the precomputed quoted ETag for a release-owned regular file,
// or "" when the path is not tracked.
func (p *StaticCachePolicy) ETag(cleanPath string) string {
	if p == nil {
		return ""
	}
	return p.etags[strings.TrimPrefix(cleanPath, "/")]
}

// IsImmutableAsset reports whether the release-owned path is a Vite
// content-hashed asset whose filename is unique per build.
func (p *StaticCachePolicy) IsImmutableAsset(cleanPath string) bool {
	return isFingerprintedEmbeddedAssetPath(cleanPath)
}

// cacheControlFor returns the policy's Cache-Control value for a tracked path:
//   - Vite content-hashed assets get the immutable, one-year policy;
//   - every other release-owned file gets "no-cache" plus its deterministic
//     ETag so caches may store it but must revalidate.
func (p *StaticCachePolicy) cacheControlFor(cleanPath string) string {
	if p.IsImmutableAsset(cleanPath) {
		return staticAssetsCacheControl
	}
	return staticMutableCacheControl
}

// Serve serves a release-owned static file through handler while enforcing the
// policy's cache headers only on successful responses.
//
// http.FileServer evaluates If-Match / If-None-Match / If-Range against the
// ETag found in the response header map when the precondition check runs. If
// that ETag (and Cache-Control) were placed on the real writer before serving,
// the failure responses would inherit them: a 412 "Precondition Failed" would
// go out with "public, max-age=31536000, immutable", and a cache would happily
// store that error in place of the asset — cache poisoning.
//
// Serve therefore hands the handler a local response writer whose Header() is
// a private map. The policy ETag lives there only long enough for ServeContent
// to evaluate the request's conditionals, and only a 200/304 response copies
// the policy Cache-Control and ETag to the underlying writer; every other
// status (206, 301, 404, 405, 412, 416, ...) has them stripped explicitly.
// index.html, SPA fallbacks, directories, and missing paths are not tracked by
// the policy and are served without the wrapper.
func (p *StaticCachePolicy) Serve(w http.ResponseWriter, r *http.Request, cleanPath string, handler http.Handler) {
	if p == nil || handler == nil {
		if handler != nil && w != nil {
			handler.ServeHTTP(w, r)
		}
		return
	}
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	etag, tracked := p.etags[cleanPath]
	if !tracked || cleanPath == "index.html" {
		handler.ServeHTTP(w, r)
		return
	}
	wrapped := &staticCacheResponseWriter{
		inner:     w,
		policy:    p,
		cleanPath: cleanPath,
		etag:      etag,
		private:   make(http.Header),
	}
	// The private ETag is what ServeContent's precondition checks see; it is
	// copied to the wire only on 200/304.
	wrapped.private.Set("ETag", etag)
	handler.ServeHTTP(wrapped, r)
}

// staticCacheResponseWriter is a local response writer that defers the
// policy's cache headers until the handler commits to a final status.
type staticCacheResponseWriter struct {
	inner     http.ResponseWriter
	private   http.Header
	policy    *StaticCachePolicy
	cleanPath string
	etag      string
	written   bool
}

// Header returns the private header map accumulated while the handler runs.
func (w *staticCacheResponseWriter) Header() http.Header {
	return w.private
}

// WriteHeader commits the response. Only the first call has any effect, so
// repeated WriteHeader calls from the handler are safe.
func (w *staticCacheResponseWriter) WriteHeader(status int) {
	if w.written {
		return
	}
	w.written = true

	switch status {
	case http.StatusOK, http.StatusNotModified:
		w.private.Set("Cache-Control", w.policy.cacheControlFor(w.cleanPath))
		w.private.Set("ETag", w.etag)
	default:
		// Error, redirect, and partial responses must never carry static
		// cache headers: caching a 412/206/301/404/405/416 as if it were the
		// asset would poison every downstream cache. Strip them from both the
		// accumulated headers and anything already on the underlying writer.
		w.private.Del("Cache-Control")
		w.private.Del("ETag")
		w.inner.Header().Del("Cache-Control")
		w.inner.Header().Del("ETag")
	}

	// Merge the accumulated headers into the underlying writer, preserving
	// unrelated headers already set by outer middleware.
	for key, values := range w.private {
		w.inner.Header()[key] = values
	}
	w.inner.WriteHeader(status)
}

// Write flushes the accumulated headers as an implicit 200, then forwards the
// body.
func (w *staticCacheResponseWriter) Write(p []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.inner.Write(p)
}
