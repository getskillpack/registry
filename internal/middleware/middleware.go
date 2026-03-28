package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Chain applies middlewares outer-to-inner: first middleware is the outermost.
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger logs one line per request (method, path, status, duration, client IP).
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rr.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_ip", clientIP(r),
		)
	})
}

func clientIP(r *http.Request) string {
	if trust := strings.TrimSpace(os.Getenv("REGISTRY_TRUST_FORWARDED_FOR")); trust == "1" || strings.EqualFold(trust, "true") {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// IPRateLimiter is a per-client-IP token bucket (by RemoteAddr or X-Forwarded-For when trusted).
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	lim      rate.Limit
	burst    int
	maxKeys  int
}

// NewIPRateLimiter creates a limiter. maxKeys bounds distinct IPs kept in memory (oldest dropped when full).
func NewIPRateLimiter(rps float64, burst, maxKeys int) *IPRateLimiter {
	if maxKeys <= 0 {
		maxKeys = 50000
	}
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		lim:      rate.Limit(rps),
		burst:    burst,
		maxKeys:  maxKeys,
	}
}

func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limiters[ip]
	if !ok {
		if len(l.limiters) >= l.maxKeys {
			for k := range l.limiters {
				delete(l.limiters, k)
				break
			}
		}
		lim = rate.NewLimiter(l.lim, l.burst)
		l.limiters[ip] = lim
	}
	return lim.Allow()
}

// RateLimit returns middleware that skips paths in skipPrefixes and returns 429 when exceeded.
func (l *IPRateLimiter) RateLimit(skipPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			for _, p := range skipPrefixes {
				if p != "" && strings.HasPrefix(path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			if path == "/healthz" || path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}
			if !l.allow(clientIP(r)) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ParseRateLimitEnv reads REGISTRY_RATE_LIMIT_RPS and REGISTRY_RATE_LIMIT_BURST.
// If RPS <= 0 or unset, returns ok=false (caller should disable limiting).
func ParseRateLimitEnv() (rps float64, burst int, ok bool) {
	s := strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_RPS"))
	if s == "" {
		return 0, 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0, 0, false
	}
	burst = 0
	if bs := strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_BURST")); bs != "" {
		if n, err := strconv.Atoi(bs); err == nil && n > 0 {
			burst = n
		}
	}
	if burst <= 0 {
		burst = int(v * 2)
		if burst < 10 {
			burst = 10
		}
	}
	return v, burst, true
}
