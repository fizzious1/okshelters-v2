package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// cacheEntry stores a captured HTTP response with an expiration time.
type cacheEntry struct {
	body       []byte
	header     http.Header
	statusCode int
	expiresAt  time.Time

	// Doubly-linked list pointers for LRU eviction.
	prev, next *cacheEntry
	key        string
}

// ResponseCache is a simple in-memory LRU response cache for GET requests.
// Keyed on the full request URL. Thread-safe via sync.RWMutex.
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int

	// Sentinel nodes for the doubly-linked list (most-recent at head).
	head, tail *cacheEntry
}

// NewResponseCache creates a cache with the given TTL and maximum number of entries.
func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	head := &cacheEntry{}
	tail := &cacheEntry{}
	head.next = tail
	tail.prev = head

	return &ResponseCache{
		entries: make(map[string]*cacheEntry, maxSize),
		ttl:     ttl,
		maxSize: maxSize,
		head:    head,
		tail:    tail,
	}
}

// moveToFront moves an entry to the front of the LRU list.
// Caller must hold the write lock.
func (rc *ResponseCache) moveToFront(e *cacheEntry) {
	// Remove from current position.
	e.prev.next = e.next
	e.next.prev = e.prev

	// Insert after head sentinel.
	e.next = rc.head.next
	e.prev = rc.head
	rc.head.next.prev = e
	rc.head.next = e
}

// addToFront inserts a new entry at the front of the LRU list.
// Caller must hold the write lock.
func (rc *ResponseCache) addToFront(e *cacheEntry) {
	e.next = rc.head.next
	e.prev = rc.head
	rc.head.next.prev = e
	rc.head.next = e
}

// evictOldest removes the least recently used entry.
// Caller must hold the write lock.
func (rc *ResponseCache) evictOldest() {
	oldest := rc.tail.prev
	if oldest == rc.head {
		return // list is empty
	}
	oldest.prev.next = rc.tail
	rc.tail.prev = oldest.prev
	delete(rc.entries, oldest.key)
}

// get retrieves a cached response if it exists and hasn't expired.
func (rc *ResponseCache) get(key string) (*cacheEntry, bool) {
	rc.mu.RLock()
	e, exists := rc.entries[key]
	rc.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(e.expiresAt) {
		// Expired entry; remove it.
		rc.mu.Lock()
		// Double-check after acquiring write lock.
		if e2, ok := rc.entries[key]; ok && time.Now().After(e2.expiresAt) {
			e2.prev.next = e2.next
			e2.next.prev = e2.prev
			delete(rc.entries, key)
		}
		rc.mu.Unlock()
		return nil, false
	}

	// Promote to front.
	rc.mu.Lock()
	rc.moveToFront(e)
	rc.mu.Unlock()

	return e, true
}

// put stores a response in the cache.
func (rc *ResponseCache) put(key string, statusCode int, header http.Header, body []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if existing, ok := rc.entries[key]; ok {
		// Update existing entry.
		existing.body = body
		existing.header = header.Clone()
		existing.statusCode = statusCode
		existing.expiresAt = time.Now().Add(rc.ttl)
		rc.moveToFront(existing)
		return
	}

	// Evict if at capacity.
	if len(rc.entries) >= rc.maxSize {
		rc.evictOldest()
	}

	e := &cacheEntry{
		key:        key,
		body:       body,
		header:     header.Clone(),
		statusCode: statusCode,
		expiresAt:  time.Now().Add(rc.ttl),
	}
	rc.entries[key] = e
	rc.addToFront(e)
}

// responseBuffer captures a response written by the downstream handler.
type responseBuffer struct {
	http.ResponseWriter
	buf        bytes.Buffer
	statusCode int
	written    bool
}

func (rb *responseBuffer) WriteHeader(code int) {
	rb.statusCode = code
	rb.written = true
	rb.ResponseWriter.WriteHeader(code)
}

func (rb *responseBuffer) Write(b []byte) (int, error) {
	if !rb.written {
		rb.statusCode = http.StatusOK
		rb.written = true
	}
	rb.buf.Write(b)
	return rb.ResponseWriter.Write(b)
}

// Cache returns middleware that caches GET responses in-memory with LRU eviction.
// Only successful responses (2xx) are cached. Non-GET requests pass through.
func Cache(rc *ResponseCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests.
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		key := r.URL.String()

		// Check cache.
		if entry, ok := rc.get(key); ok {
			for k, vals := range entry.header {
				for _, v := range vals {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.statusCode)
			_, _ = w.Write(entry.body)
			return
		}

		// Cache miss: capture the response.
		buf := &responseBuffer{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		w.Header().Set("X-Cache", "MISS")
		next.ServeHTTP(buf, r)

		// Only cache successful responses.
		if buf.statusCode >= 200 && buf.statusCode < 300 {
			rc.put(key, buf.statusCode, w.Header(), buf.buf.Bytes())
		}
	})
}
