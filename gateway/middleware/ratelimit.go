package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter implements per-IP token bucket limiting.
type RateLimiter struct {
	mu            sync.Mutex
	clients       map[string]*clientLimiter
	limit         rate.Limit
	burst         int
	staleAfter    time.Duration
	pruneInterval time.Duration
	lastPruneAt   time.Time
}

// NewRateLimiter creates a limiter with requests-per-second rate and burst.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if rps <= 0 {
		rps = 100
	}
	if burst < 1 {
		burst = 1
	}

	return &RateLimiter{
		clients:       make(map[string]*clientLimiter, 1024),
		limit:         rate.Limit(rps),
		burst:         burst,
		staleAfter:    5 * time.Minute,
		pruneInterval: 1 * time.Minute,
	}
}

func (rl *RateLimiter) allow(key string, now time.Time) bool {
	limiter := rl.getLimiter(key, now)
	return limiter.AllowN(now, 1)
}

func (rl *RateLimiter) getLimiter(key string, now time.Time) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.Sub(rl.lastPruneAt) >= rl.pruneInterval {
		for clientKey, client := range rl.clients {
			if now.Sub(client.lastSeen) >= rl.staleAfter {
				delete(rl.clients, clientKey)
			}
		}
		rl.lastPruneAt = now
	}

	entry, ok := rl.clients[key]
	if !ok {
		entry = &clientLimiter{
			limiter:  rate.NewLimiter(rl.limit, rl.burst),
			lastSeen: now,
		}
		rl.clients[key] = entry
		return entry.limiter
	}

	entry.lastSeen = now
	return entry.limiter
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
	}

	ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		if r.RemoteAddr == "" {
			return "unknown"
		}
		return strings.TrimSpace(r.RemoteAddr)
	}
	return ip
}

// RateLimit enforces per-IP rate limits and returns HTTP 429 when exceeded.
func RateLimit(rl *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r), time.Now()) {
			w.Header().Set("Retry-After", "1")
			writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
