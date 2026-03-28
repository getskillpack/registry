package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics holds Prometheus collectors for HTTP (isolated registry).
type HTTPMetrics struct {
	reg             *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewHTTPMetrics builds a dedicated registry with HTTP metrics.
func NewHTTPMetrics() (*HTTPMetrics, error) {
	reg := prometheus.NewRegistry()
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_http_requests_total",
			Help: "HTTP requests processed by the registry.",
		},
		[]string{"method", "route", "code"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "registry_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "code"},
	)
	if err := reg.Register(requestsTotal); err != nil {
		return nil, err
	}
	if err := reg.Register(requestDuration); err != nil {
		return nil, err
	}
	return &HTTPMetrics{
		reg:             reg,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
	}, nil
}

// Handler serves scrape output for this metrics registry.
func (m *HTTPMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{Registry: m.reg})
}

// RouteTemplate maps a request path to a low-cardinality route label.
func RouteTemplate(path string) string {
	switch {
	case path == "/healthz", path == "/readyz", path == "/metrics":
		return path
	case path == "/api/v1/skills":
		return "/api/v1/skills"
	case strings.HasPrefix(path, "/downloads/"):
		return "/downloads/{file}"
	}
	if strings.HasPrefix(path, "/api/v1/skills/") {
		rest := strings.TrimPrefix(path, "/api/v1/skills/")
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) >= 3 && parts[1] == "versions" {
			return "/api/v1/skills/{name}/versions/{version}"
		}
		if len(parts) >= 1 && parts[0] != "" {
			return "/api/v1/skills/{name}"
		}
	}
	if strings.HasPrefix(path, "/api/") {
		return "/api/unknown"
	}
	return path
}

// Middleware records request counts and latency.
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		route := RouteTemplate(r.URL.Path)
		code := strconv.Itoa(rr.status)
		m.requestsTotal.WithLabelValues(r.Method, route, code).Inc()
		m.requestDuration.WithLabelValues(r.Method, route, code).Observe(time.Since(start).Seconds())
	})
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (rw *recorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
