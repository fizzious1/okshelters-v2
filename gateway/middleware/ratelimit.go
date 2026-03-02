package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// bucket tracks token-bucket state for a single client.
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter implements a per-IP token bucket rate limiter.
// All state is in-memory; no external dependencies.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64 // maximum tokens (burst capacity)
}

// NewRateLimiter creates a rate limiter with the given sustained rate
// (requests per second) and burst size.
func NewRateLimiter(rate float64, burst float64) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
}

// allow checks whether the given key (IP) is allowed to proceed.
// Returns true if a token was consumed, false if rate-limited.
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{
			tokens:     rl.burst - 1, // consume one token immediately
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// clientIP extracts the client IP from the request, preferring
// X-Forwarded-For when behind a trusted reverse proxy.
func clientIP(r *http.Request) string {
	// In production, only trust X-Forwarded-For from known proxy IPs.
	// For now, fall back to RemoteAddr.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimit returns middleware that enforces per-IP rate limiting.
// Returns HTTP 429 when the client exceeds the configured rate.
func RateLimit(rl *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
