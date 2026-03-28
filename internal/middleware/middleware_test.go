package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPRateLimit429(t *testing.T) {
	lim := NewIPRateLimiter(1, 1, 100)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/skills", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Chain(mux, lim.RateLimit())
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/v1/skills")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first request: %d", resp.StatusCode)
	}

	resp2, err := http.Get(ts.URL + "/api/v1/skills")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second request: want 429, got %d", resp2.StatusCode)
	}
}

func TestRateLimitSkipsHealthEndpoints(t *testing.T) {
	lim := NewIPRateLimiter(1, 1, 100)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Chain(mux, lim.RateLimit())
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	for i := 0; i < 5; i++ {
		resp, err := http.Get(ts.URL + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: %d", i, resp.StatusCode)
		}
	}
}

func TestParseRateLimitEnv(t *testing.T) {
	t.Setenv("REGISTRY_RATE_LIMIT_RPS", "")
	t.Setenv("REGISTRY_RATE_LIMIT_BURST", "")
	if _, _, ok := ParseRateLimitEnv(); ok {
		t.Fatal("expected disabled when unset")
	}

	t.Setenv("REGISTRY_RATE_LIMIT_RPS", "10")
	t.Setenv("REGISTRY_RATE_LIMIT_BURST", "5")
	rps, burst, ok := ParseRateLimitEnv()
	if !ok || rps != 10 || burst != 5 {
		t.Fatalf("got rps=%v burst=%v ok=%v", rps, burst, ok)
	}
}

func TestClientIPForwardedFor(t *testing.T) {
	t.Setenv("REGISTRY_TRUST_FORWARDED_FOR", "1")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
	if got := clientIP(req); got != "203.0.113.7" {
		t.Fatalf("clientIP: %q", got)
	}
}
