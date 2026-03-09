package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCacheNearestRoundingAndTTL(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	cache := NewResponseCache(1*time.Second, 128)
	cache.nowFn = func() time.Time { return now }

	hits := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = fmt.Fprintf(w, "{\"n\":%d}", hits)
	})

	h := Cache(cache, next)

	first := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=31.77004&lon=35.21004&radius=5000&limit=10", nil)
	firstRec := httptest.NewRecorder()
	h.ServeHTTP(firstRec, first)

	if firstRec.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("expected MISS, got %q", firstRec.Header().Get("X-Cache"))
	}
	if hits != 1 {
		t.Fatalf("expected upstream hits 1, got %d", hits)
	}

	second := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=31.77003&lon=35.21003&radius=5000&limit=10", nil)
	secondRec := httptest.NewRecorder()
	h.ServeHTTP(secondRec, second)

	if secondRec.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected HIT, got %q", secondRec.Header().Get("X-Cache"))
	}
	if hits != 1 {
		t.Fatalf("expected upstream hits 1 after cached call, got %d", hits)
	}

	now = now.Add(2 * time.Second)
	third := httptest.NewRequest(http.MethodGet, "/api/v1/shelters/nearest?lat=31.77004&lon=35.21004&radius=5000&limit=10", nil)
	thirdRec := httptest.NewRecorder()
	h.ServeHTTP(thirdRec, third)

	if thirdRec.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("expected MISS after ttl expiry, got %q", thirdRec.Header().Get("X-Cache"))
	}
	if hits != 2 {
		t.Fatalf("expected upstream hits 2 after ttl expiry, got %d", hits)
	}
}
