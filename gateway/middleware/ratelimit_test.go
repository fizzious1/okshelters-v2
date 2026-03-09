package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	if !rl.allow("192.168.1.1") {
		t.Error("expected first request to be allowed")
	}
}

func TestRateLimiter_Burst(t *testing.T) {
	burst := 5.0
	rl := NewRateLimiter(1, burst) // 1 req/s sustained, burst of 5

	key := "10.0.0.1"
	allowed := 0
	for i := 0; i < int(burst)+5; i++ {
		if rl.allow(key) {
			allowed++
		}
	}

	// The first call creates a bucket with burst-1 tokens, then subsequent
	// calls consume the remaining tokens. Total allowed should equal burst.
	if allowed != int(burst) {
		t.Errorf("expected %d requests allowed (burst), got %d", int(burst), allowed)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(100, 2) // high rate so refill is fast

	key := "10.0.0.2"

	// Exhaust burst.
	for i := 0; i < 2; i++ {
		rl.allow(key)
	}

	// Should be denied now.
	if rl.allow(key) {
		t.Error("expected request to be denied after burst exhaustion")
	}

	// Manually refill by adjusting the bucket's lastRefill time.
	rl.mu.Lock()
	b := rl.buckets[key]
	// Simulate 1 second passing with rate=100, so 100 tokens refill.
	b.tokens = 0
	b.lastRefill = b.lastRefill.Add(-1e9) // subtract 1 second
	rl.mu.Unlock()

	if !rl.allow(key) {
		t.Error("expected request to be allowed after token refill")
	}
}

func TestRateLimit_Middleware_Returns429(t *testing.T) {
	rl := NewRateLimiter(1, 1) // 1 token burst only

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimit(rl, inner)

	// First request from same IP should succeed.
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d", rec1.Code)
	}

	// Second request from same IP should be rate-limited.
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.100:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status 429, got %d", rec2.Code)
	}

	retryAfter := rec2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}
