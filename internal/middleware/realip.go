package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns a best-effort client identifier for rate limiting.
// When trustForwarded is true, the first address in X-Forwarded-For is used when present.
func ClientIP(r *http.Request, trustForwarded bool) string {
	if trustForwarded {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
