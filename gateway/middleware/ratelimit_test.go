package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitBlocksBurstExceeded(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1, 1)
	h := RateLimit(rl, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest", nil)
	req1.RemoteAddr = "203.0.113.10:12345"
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusNoContent {
		t.Fatalf("expected first request 204, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest", nil)
	req2.RemoteAddr = "203.0.113.10:9876"
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", rr2.Code)
	}
}

func TestRateLimitUsesXForwardedFor(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1, 1)
	h := RateLimit(rl, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest", nil)
	req1.RemoteAddr = "127.0.0.1:4444"
	req1.Header.Set("X-Forwarded-For", "198.51.100.40, 127.0.0.1")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest", nil)
	req2.RemoteAddr = "127.0.0.1:5555"
	req2.Header.Set("X-Forwarded-For", "198.51.100.40")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	if rr1.Code != http.StatusNoContent {
		t.Fatalf("expected first request 204, got %d", rr1.Code)
	}
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", rr2.Code)
	}
}
