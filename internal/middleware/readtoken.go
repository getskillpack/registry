package middleware

import (
	"net/http"
	"strings"
)

// RequireReadToken enforces Bearer auth on GET/HEAD for /api/v1/* and /downloads/*
// when token is non-empty. POST/DELETE are unchanged (write handlers use write token).
func RequireReadToken(token string) func(http.Handler) http.Handler {
	if strings.TrimSpace(token) == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	tok := strings.TrimSpace(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresReadAuth(r) {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != tok {
				w.Header().Set("WWW-Authenticate", `Bearer realm="registry"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requiresReadAuth(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	if path == "/metrics" {
		return false
	}
	return strings.HasPrefix(path, "/api/v1/") || strings.HasPrefix(path, "/downloads/")
}
