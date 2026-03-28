package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_remoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	if got := ClientIP(req, false); got != "192.0.2.1" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIP_forwarded(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := ClientIP(req, true); got != "203.0.113.9" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIP_forwardedDisabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := ClientIP(req, false); got != "10.0.0.1" {
		t.Fatalf("got %q", got)
	}
}
