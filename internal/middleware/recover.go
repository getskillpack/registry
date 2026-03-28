package middleware

import (
	"log/slog"
	"net/http"
)

// Recover catches panics in downstream handlers and responds with 500.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("handler panic", "err", v, "path", r.URL.Path, "method", r.Method)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
