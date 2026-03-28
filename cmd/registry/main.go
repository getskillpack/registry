package main

import (
	"log"
	"net/http"
	"os"

	"github.com/getskillpack/registry/internal/api"
	"github.com/getskillpack/registry/internal/store"
)

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
	log.Printf("registry listening on %s (data=%s)", addr, dataDir)
	log.Fatal(http.ListenAndServe(addr, mux))
}
