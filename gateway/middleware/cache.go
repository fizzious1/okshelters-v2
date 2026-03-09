package middleware

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

type cacheEntry struct {
	statusCode int
	header     http.Header
	body       []byte
	expiresAt  time.Time
}

// ResponseCache stores successful GET responses with TTL.
type ResponseCache struct {
	mu    sync.Mutex
	lru   *lru.Cache
	ttl   time.Duration
	nowFn func() time.Time
}

// NewResponseCache creates a response cache using groupcache LRU.
func NewResponseCache(ttl time.Duration, maxEntries int) *ResponseCache {
	if ttl <= 0 {
		ttl = time.Second
	}
	if maxEntries < 1 {
		maxEntries = 256
	}
	return &ResponseCache{
		lru:   lru.New(maxEntries),
		ttl:   ttl,
		nowFn: time.Now,
	}
}

func (rc *ResponseCache) get(key string, now time.Time) (cacheEntry, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	v, ok := rc.lru.Get(key)
	if !ok {
		return cacheEntry{}, false
	}

	entry, ok := v.(*cacheEntry)
	if !ok {
		rc.lru.Remove(key)
		return cacheEntry{}, false
	}

	if now.After(entry.expiresAt) {
		rc.lru.Remove(key)
		return cacheEntry{}, false
	}

	return cacheEntry{
		statusCode: entry.statusCode,
		header:     entry.header.Clone(),
		body:       append([]byte(nil), entry.body...),
		expiresAt:  entry.expiresAt,
	}, true
}

func (rc *ResponseCache) put(key string, statusCode int, header http.Header, body []byte, now time.Time) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.lru.Add(key, &cacheEntry{
		statusCode: statusCode,
		header:     header.Clone(),
		body:       append([]byte(nil), body...),
		expiresAt:  now.Add(rc.ttl),
	})
}

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

// Cache caches successful GET responses for supported API endpoints.
func Cache(rc *ResponseCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacheKey, cacheable := requestCacheKey(r)
		if !cacheable {
			next.ServeHTTP(w, r)
			return
		}

		now := rc.nowFn()
		if entry, ok := rc.get(cacheKey, now); ok {
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

		w.Header().Set("X-Cache", "MISS")
		buf := &responseBuffer{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(buf, r)

		if buf.statusCode >= 200 && buf.statusCode < 300 {
			rc.put(cacheKey, buf.statusCode, w.Header(), buf.buf.Bytes(), now)
		}
	})
}

func requestCacheKey(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet {
		return "", false
	}

	switch r.URL.Path {
	case "/api/v1/shelters/nearest":
		return nearestCacheKey(r)
	case "/api/v1/route":
		return "route:" + r.URL.RawQuery, true
	default:
		return "", false
	}
}

func nearestCacheKey(r *http.Request) (string, bool) {
	q := r.URL.Query()
	lat, err := strconv.ParseFloat(q.Get("lat"), 64)
	if err != nil {
		return "", false
	}
	lon, err := strconv.ParseFloat(q.Get("lon"), 64)
	if err != nil {
		return "", false
	}

	radius := q.Get("radius")
	if radius == "" {
		radius = "5000"
	}
	limit := q.Get("limit")
	if limit == "" {
		limit = "10"
	}

	return fmt.Sprintf("nearest:%.4f:%.4f:%s:%s", roundCoord(lat), roundCoord(lon), radius, limit), true
}

func roundCoord(v float64) float64 {
	return math.Round(v*10000) / 10000
}
