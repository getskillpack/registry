package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/getskillpack/registry/internal/store"
)

// Server serves registry HTTP API under /api/v1.
type Server struct {
	Store       store.Store
	WriteToken  string // if empty, POST/DELETE return 503
	DefaultAddr string // host:port for public URL fallback (e.g. localhost:8080)
}

func (s *Server) publicBase(r *http.Request) string {
	if u := strings.TrimSpace(os.Getenv("REGISTRY_PUBLIC_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := r.Host
	if xf := r.Header.Get("X-Forwarded-Host"); xf != "" {
		host = xf
	}
	if host == "" && s.DefaultAddr != "" {
		host = s.DefaultAddr
	}
	return proto + "://" + host
}

func (s *Server) requireWrite(w http.ResponseWriter, r *http.Request) bool {
	if s.WriteToken == "" {
		http.Error(w, "write operations disabled (set REGISTRY_WRITE_TOKEN)", http.StatusServiceUnavailable)
		return false
	}
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != s.WriteToken {
		w.Header().Set("WWW-Authenticate", `Bearer realm="registry"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// Register mounts routes on mux (Go 1.22+ patterns).
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/skills", s.handleListSkills)
	mux.HandleFunc("GET /api/v1/skills/{name}", s.handleGetSkill)
	mux.HandleFunc("GET /api/v1/skills/{name}/versions/{version}", s.handleGetVersion)
	mux.HandleFunc("POST /api/v1/skills", s.handlePublish)
	mux.HandleFunc("DELETE /api/v1/skills/{name}/versions/{version}", s.handleYank)
	mux.HandleFunc("GET /downloads/{file}", s.handleDownload)
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	author := r.URL.Query().Get("author")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	res, err := s.Store.List(r.Context(), store.ListParams{
		Query: q, Author: author, Limit: limit, Offset: offset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lim := limit
	if lim <= 0 {
		lim = 20
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": res.Items,
		"meta": map[string]int{"total": res.Total, "limit": lim, "offset": offset},
	})
}

func (s *Server) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	base := s.publicBase(r)
	detail, err := s.Store.GetSkill(r.Context(), name, base)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	version := r.PathValue("version")
	base := s.publicBase(r)
	rec, err := s.Store.GetVersion(r.Context(), name, version, base)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrYanked) {
			http.Error(w, "gone", http.StatusGone)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if !s.requireWrite(w, r) {
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "bad multipart", http.StatusBadRequest)
		return
	}
	manifestStr := r.FormValue("manifest")
	if manifestStr == "" {
		http.Error(w, "missing manifest", http.StatusBadRequest)
		return
	}
	var manifest struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Author      string `json:"author"`
	}
	if err := json.Unmarshal([]byte(manifestStr), &manifest); err != nil {
		http.Error(w, "invalid manifest json", http.StatusBadRequest)
		return
	}
	if manifest.Name == "" || manifest.Version == "" {
		http.Error(w, "manifest must include name and version", http.StatusBadRequest)
		return
	}
	f, _, err := r.FormFile("archive")
	if err != nil {
		http.Error(w, "missing archive", http.StatusBadRequest)
		return
	}
	defer f.Close()
	archive, err := io.ReadAll(f)
	if err != nil || len(archive) == 0 {
		http.Error(w, "empty archive", http.StatusBadRequest)
		return
	}
	raw := json.RawMessage(manifestStr)
	err = s.Store.Publish(r.Context(), manifest.Name, manifest.Version, manifest.Description, manifest.Author, raw, archive)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			http.Error(w, "version already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleYank(w http.ResponseWriter, r *http.Request) {
	if !s.requireWrite(w, r) {
		return
	}
	name := r.PathValue("name")
	version := r.PathValue("version")
	err := s.Store.Yank(r.Context(), name, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !strings.HasSuffix(file, ".tar.gz") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	hexSum := strings.TrimSuffix(file, ".tar.gz")
	if hexSum == "" || strings.Contains(hexSum, "/") || strings.Contains(hexSum, "..") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	rc, size, err := s.Store.OpenArchive(r.Context(), hexSum)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file+"\"")
	_, _ = io.Copy(w, rc)
}
