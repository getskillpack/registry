package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/getskillpack/registry/internal/api"
	"github.com/getskillpack/registry/internal/metrics"
	"github.com/getskillpack/registry/internal/middleware"
	"github.com/getskillpack/registry/internal/store"
)

func setupLogger() {
	level := slog.LevelInfo
	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REGISTRY_LOG_FORMAT"))) {
	case "json":
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	default:
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(h))
}

func metricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REGISTRY_ENABLE_METRICS")))
	return v == "1" || v == "true" || v == "yes"
}

func main() {
	setupLogger()

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
		Store:       st,
		WriteToken:  writeTok,
		DefaultAddr: addr,
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
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if err := st.Ping(context.Background()); err != nil {
			http.Error(w, "store unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	handler := http.Handler(mux)
	handler = middleware.RequireReadToken(readTok)(handler)
	if rps, burst, ok := middleware.ParseRateLimitEnv(); ok {
		lim := middleware.NewIPRateLimiter(rps, burst, 50000)
		handler = lim.RateLimit()(handler)
		slog.Info("rate limit enabled", "rps", rps, "burst", burst)
	}
	handler = middleware.RequestLogger(handler)
	if metricsEnabled() {
		prom, err := metrics.NewHTTPMetrics()
		if err != nil {
			log.Fatalf("metrics: %v", err)
		}
		mux.Handle("GET /metrics", prom.Handler())
		handler = prom.Middleware(handler)
		slog.Info("prometheus metrics enabled", "path", "/metrics")
	}
	handler = middleware.Recover(handler)

	slog.Info("registry listening", "addr", addr, "data_dir", dataDir)
	log.Fatal(http.ListenAndServe(addr, handler))
}
