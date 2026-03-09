package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseCache_Miss(t *testing.T) {
	rc := NewResponseCache(1*time.Second, 100)

	_, ok := rc.get("/test")
	if ok {
		t.Error("expected cache miss on empty cache")
	}
}

func TestResponseCache_Hit(t *testing.T) {
	rc := NewResponseCache(5*time.Second, 100)

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	rc.put("/test", http.StatusOK, header, []byte(`{"status":"ok"}`))

	entry, ok := rc.get("/test")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.statusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", entry.statusCode)
	}
	if string(entry.body) != `{"status":"ok"}` {
		t.Errorf("unexpected cached body: %q", string(entry.body))
	}
}

func TestResponseCache_Expiration(t *testing.T) {
	rc := NewResponseCache(50*time.Millisecond, 100)

	header := http.Header{}
	rc.put("/expire-test", http.StatusOK, header, []byte("data"))

	// Should hit immediately.
	if _, ok := rc.get("/expire-test"); !ok {
		t.Fatal("expected cache hit before expiration")
	}

	// Wait for TTL to expire.
	time.Sleep(80 * time.Millisecond)

	if _, ok := rc.get("/expire-test"); ok {
		t.Error("expected cache miss after TTL expiration")
	}
}

func TestResponseCache_NonGetBypass(t *testing.T) {
	rc := NewResponseCache(5*time.Second, 100)

	callCount := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := Cache(rc, inner)

	// POST request should bypass cache.
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Second POST should also bypass cache (handler called again).
	req2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if callCount != 2 {
		t.Errorf("expected handler to be called twice for POST requests, called %d times", callCount)
	}
}

func TestResponseCache_LRUEviction(t *testing.T) {
	rc := NewResponseCache(5*time.Second, 3) // max 3 entries

	header := http.Header{}
	// Fill cache to capacity.
	rc.put("/a", http.StatusOK, header, []byte("a"))
	rc.put("/b", http.StatusOK, header, []byte("b"))
	rc.put("/c", http.StatusOK, header, []byte("c"))

	// All should be present.
	if _, ok := rc.get("/a"); !ok {
		t.Error("expected /a to be in cache")
	}
	if _, ok := rc.get("/b"); !ok {
		t.Error("expected /b to be in cache")
	}
	if _, ok := rc.get("/c"); !ok {
		t.Error("expected /c to be in cache")
	}

	// Access /a to make it most recently used. Order is now: /a, /c, /b
	// (since /c was accessed via get above, then /b, then /a).
	// Actually after the 3 gets above, the LRU order (front to back) is: /c, /b, /a
	// No -- the gets above promote each to front: /a then /b then /c.
	// After gets: /c is at front, then /b, then /a is at tail.
	// Now access /a to bring it to front.
	rc.get("/a")
	// LRU order (front to back): /a, /c, /b. So /b is the LRU.

	// Add a new entry; this should evict /b (least recently used).
	rc.put("/d", http.StatusOK, header, []byte("d"))

	if _, ok := rc.get("/b"); ok {
		t.Error("expected /b to be evicted (LRU)")
	}

	// /a, /c, /d should still be present.
	if _, ok := rc.get("/a"); !ok {
		t.Error("expected /a to remain in cache after eviction")
	}
	if _, ok := rc.get("/c"); !ok {
		t.Error("expected /c to remain in cache after eviction")
	}
	if _, ok := rc.get("/d"); !ok {
		t.Error("expected /d to be in cache")
	}
}

func TestCache_XCacheHeader(t *testing.T) {
	rc := NewResponseCache(5*time.Second, 100)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := Cache(rc, inner)

	// First request: cache miss.
	req1 := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35&lon=-97", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("expected X-Cache: MISS on first request, got %q", rec1.Header().Get("X-Cache"))
	}

	// Second request: cache hit.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/shelters/nearest?lat=35&lon=-97", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected X-Cache: HIT on second request, got %q", rec2.Header().Get("X-Cache"))
	}
}
