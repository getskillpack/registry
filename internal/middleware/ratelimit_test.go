package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimit_skipsHealth(t *testing.T) {
	var hits int
	h := RateLimit(RateLimitConfig{RPS: 0.5, Burst: 1, SkipPathPrefixes: []string{"/healthz"}})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }),
	)
	for range 5 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz: %d", rec.Code)
		}
	}
	if hits != 5 {
		t.Fatalf("hits=%d want 5", hits)
	}
}

func TestRateLimit_enforces(t *testing.T) {
	h := RateLimit(RateLimitConfig{RPS: 1, Burst: 1})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	req.RemoteAddr = "192.0.2.1:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first: %d", rec.Code)
	}
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second: %d want 429", rec2.Code)
	}
}

func TestRateLimit_disabled(t *testing.T) {
	var hits int
	h := RateLimit(RateLimitConfig{RPS: 0})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }),
	)
	for range 10 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if hits != 10 {
		t.Fatalf("hits=%d", hits)
	}
}
