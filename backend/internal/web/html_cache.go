//go:build embed

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// htmlCacheTTL is how long a rendered HTML entry may be served before Get
// treats it as a miss and the next request re-renders from current settings.
// This is a bounded-staleness fallback: explicit Invalidate still forces an
// immediate miss, and the window restarts on every Set.
const htmlCacheTTL = 60 * time.Second

// HTMLCache manages the cached index.html with injected settings
type HTMLCache struct {
	mu              sync.RWMutex
	cachedHTML      []byte
	etag            string
	baseHTMLHash    string           // Hash of the original index.html (immutable after build)
	settingsVersion uint64           // Incremented when settings change
	createdAt       time.Time        // When the current entry was Set
	now             func() time.Time // Clock seam for deterministic tests; nil means time.Now
}

// CachedHTML represents the cache state
type CachedHTML struct {
	Content []byte
	ETag    string
}

// NewHTMLCache creates a new HTML cache instance with a fixed 60-second TTL
func NewHTMLCache() *HTMLCache {
	return newHTMLCache(time.Now)
}

// newHTMLCache is the package-private construction/clock seam. Tests inject a
// fake clock to advance time deterministically without sleeps.
func newHTMLCache(now func() time.Time) *HTMLCache {
	if now == nil {
		now = time.Now
	}
	return &HTMLCache{now: now}
}

func (c *HTMLCache) nowFn() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// SetBaseHTML initializes the cache with the base HTML template
func (c *HTMLCache) SetBaseHTML(baseHTML []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := sha256.Sum256(baseHTML)
	c.baseHTMLHash = hex.EncodeToString(hash[:8]) // First 8 bytes for brevity
}

// Invalidate marks the cache as stale
func (c *HTMLCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.settingsVersion++
	c.cachedHTML = nil
	c.etag = ""
}

// Get returns the cached HTML or nil if the cache is stale. Entries older than
// htmlCacheTTL are treated as misses, so settings are re-read at most once per
// TTL window while bursts of traffic within the window hit the cache.
func (c *HTMLCache) Get() *CachedHTML {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cachedHTML == nil {
		return nil
	}
	if c.nowFn().Sub(c.createdAt) > htmlCacheTTL {
		return nil
	}
	return &CachedHTML{
		Content: c.cachedHTML,
		ETag:    c.etag,
	}
}

// generation returns the current settings generation. Every Invalidate
// increments it, so a render that captured the generation before reading
// settings can detect, at store time, that the settings changed while it was in
// flight.
func (c *HTMLCache) generation() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.settingsVersion
}

// Set updates the cache with new rendered HTML and resets the TTL window
func (c *HTMLCache) Set(html []byte, settingsJSON []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cachedHTML = html
	c.etag = c.generateETag(settingsJSON)
	c.createdAt = c.nowFn()
}

// SetIfCurrent is the generation-guarded store used by the index-render path.
// It only writes when settingsVersion still equals gen (the generation captured
// before the settings-provider read). On a match it stores the entry, resets
// the TTL window, and returns the stored entry whose ETag corresponds exactly
// to the html/settingsJSON pair; on a mismatch it leaves the cache untouched
// and returns nil, so a render started against stale settings can never
// resurrect old HTML into the shared cache. Set remains the unconditional store
// for direct callers (e.g. tests) that want to bypass the guard.
func (c *HTMLCache) SetIfCurrent(html []byte, settingsJSON []byte, gen uint64) *CachedHTML {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.settingsVersion != gen {
		return nil
	}
	c.cachedHTML = html
	c.etag = c.generateETag(settingsJSON)
	c.createdAt = c.nowFn()
	return &CachedHTML{
		Content: c.cachedHTML,
		ETag:    c.etag,
	}
}

// generateETag creates an ETag from base HTML hash + settings hash
func (c *HTMLCache) generateETag(settingsJSON []byte) string {
	settingsHash := sha256.Sum256(settingsJSON)
	return `"` + c.baseHTMLHash + "-" + hex.EncodeToString(settingsHash[:8]) + `"`
}
