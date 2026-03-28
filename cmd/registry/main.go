package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getskillpack/registry"
	"github.com/getskillpack/registry/internal/api"
	"github.com/getskillpack/registry/internal/metrics"
	"github.com/getskillpack/registry/internal/middleware"
	"github.com/getskillpack/registry/internal/store"
)

func metricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REGISTRY_ENABLE_METRICS")))
	return v == "1" || v == "true" || v == "yes"
}

func main() {
	dataDir := os.Getenv("REGISTRY_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	st, err := store.NewFileStore(dataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	addr := os.Getenv("REGISTRY_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	writeTok := os.Getenv("REGISTRY_WRITE_TOKEN")
	readTok := os.Getenv("REGISTRY_READ_TOKEN")
	srv := &api.Server{
		Store:               st,
		WriteToken:          writeTok,
		DefaultAddr:         addr,
		RegistryAPIMarkdown: registry.RegistryAPIMarkdown,
		OGCardSVG:           registry.OGCardSVG,
	}
	if srv.DefaultAddr != "" && srv.DefaultAddr[0] == ':' {
		srv.DefaultAddr = "localhost" + srv.DefaultAddr
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REGISTRY_LOG_FORMAT")), "json") {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	slog.SetDefault(logger)

	rps, _ := strconv.ParseFloat(strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_RPS")), 64)
	burst := 0
	if v := strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_BURST")); v != "" {
		burst, _ = strconv.Atoi(v)
	}
	trustFwd := envBool(os.Getenv("REGISTRY_TRUST_FORWARDED_FOR"))

	handler := http.Handler(mux)
	handler = middleware.RequireReadToken(readTok)(handler)
	handler = middleware.RateLimit(middleware.RateLimitConfig{
		RPS:              rps,
		Burst:            burst,
		TrustForwarded:   trustFwd,
		SkipPathPrefixes: []string{"/healthz", "/readyz", "/metrics"},
	})(handler)
	handler = middleware.RequestLog(logger)(handler)
	if metricsEnabled() {
		prom, err := metrics.NewHTTPMetrics()
		if err != nil {
			log.Fatalf("metrics: %v", err)
		}
		mux.Handle("GET /metrics", prom.Handler())
		handler = prom.Middleware(handler)
		logger.Info("prometheus metrics enabled", slog.String("path", "/metrics"))
	}
	handler = middleware.Recover(logger)(handler)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout(),
	}
	if d := optionalTimeoutFromEnv("REGISTRY_HTTP_READ_TIMEOUT_SEC"); d > 0 {
		httpSrv.ReadTimeout = d
	}
	if d := optionalTimeoutFromEnv("REGISTRY_HTTP_WRITE_TIMEOUT_SEC"); d > 0 {
		httpSrv.WriteTimeout = d
	}
	if d := optionalTimeoutFromEnv("REGISTRY_HTTP_IDLE_TIMEOUT_SEC"); d > 0 {
		httpSrv.IdleTimeout = d
	}

	logger.Info("registry listening",
		slog.String("addr", addr),
		slog.String("data", dataDir),
		slog.Float64("rate_limit_rps", rps),
		slog.Int("rate_limit_burst", burst),
		slog.Bool("trust_forwarded_for", trustFwd),
		slog.Duration("read_header_timeout", httpSrv.ReadHeaderTimeout),
	)
	log.Fatal(httpSrv.ListenAndServe())
}

// readHeaderTimeout defaults to 10s (slowloris mitigation). Set REGISTRY_HTTP_READ_HEADER_TIMEOUT_SEC=0 to disable.
func readHeaderTimeout() time.Duration {
	s := strings.TrimSpace(os.Getenv("REGISTRY_HTTP_READ_HEADER_TIMEOUT_SEC"))
	if s == "" {
		return 10 * time.Second
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 10 * time.Second
	}
	if n == 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func optionalTimeoutFromEnv(name string) time.Duration {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func envBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
