package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireReadToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RequireReadToken("secret-read")(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("without token: %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	req2.Header.Set("Authorization", "Bearer secret-read")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("with token: %d", rec2.Code)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/skills", nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("POST should skip read token: %d", rec3.Code)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec4 := httptest.NewRecorder()
	h.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("GET /metrics should skip read token: %d", rec4.Code)
	}

	req5 := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec5 := httptest.NewRecorder()
	h.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusOK {
		t.Fatalf("GET /version should skip read token: %d", rec5.Code)
	}
}
