package middleware

import (
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures per-IP token-bucket limiting. Zero RPS disables the middleware.
type RateLimitConfig struct {
	RPS              float64
	Burst            int
	TrustForwarded   bool
	SkipPathPrefixes []string
}

type ipLimiter struct {
	lim *rate.Limiter
}

// RateLimit returns middleware that limits requests per client IP. If cfg.RPS <= 0, returns a no-op.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.RPS <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	burst := cfg.Burst
	if burst < 1 {
		burst = 1
	}
	var mu sync.Mutex
	byIP := make(map[string]*ipLimiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			for _, p := range cfg.SkipPathPrefixes {
				if p == "" {
					continue
				}
				if path == p || strings.HasPrefix(path, p+"/") {
					next.ServeHTTP(w, r)
					return
				}
			}
			ip := ClientIP(r, cfg.TrustForwarded)
			mu.Lock()
			entry, ok := byIP[ip]
			if !ok {
				entry = &ipLimiter{lim: rate.NewLimiter(rate.Limit(cfg.RPS), burst)}
				byIP[ip] = entry
			}
			lim := entry.lim
			mu.Unlock()
			if !lim.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
