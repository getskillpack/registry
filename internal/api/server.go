package api

import (
	"encoding/json"
	"errors"
	"html/template"
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
	// RegistryAPIMarkdown is optional; when set, served at GET /docs/registry-api (text/markdown).
	RegistryAPIMarkdown []byte
	// OGCardSVG is optional; when set, served at GET /og.svg for Open Graph previews.
	OGCardSVG []byte
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
	mux.HandleFunc("GET /", s.handleRoot)
	if len(s.RegistryAPIMarkdown) > 0 {
		mux.HandleFunc("GET /docs/registry-api", s.handleRegistryAPIDoc)
	}
	if len(s.OGCardSVG) > 0 {
		mux.HandleFunc("GET /og.svg", s.handleOGCard)
	}
	mux.HandleFunc("GET /api/v1/skills", s.handleListSkills)
	mux.HandleFunc("GET /api/v1/skills/{name}", s.handleGetSkill)
	mux.HandleFunc("GET /api/v1/skills/{name}/versions/{version}", s.handleGetVersion)
	mux.HandleFunc("POST /api/v1/skills", s.handlePublish)
	mux.HandleFunc("DELETE /api/v1/skills/{name}/versions/{version}", s.handleYank)
	mux.HandleFunc("GET /downloads/{file}", s.handleDownload)
}

var rootPageTmpl = template.Must(template.New("root").Parse(`<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>getskillpack registry</title>
<meta name="description" content="HTTP API и хранилище skill-пакетов для экосистемы getskillpack.">
<meta property="og:title" content="getskillpack registry">
<meta property="og:description" content="HTTP API и хранилище skill-пакетов для экосистемы getskillpack.">
<meta property="og:type" content="website">
<meta property="og:url" content="{{.Base}}">
{{if .OGImageURL}}<meta property="og:image" content="{{.OGImageURL}}">
<meta name="twitter:card" content="summary_large_image">{{end}}
{{if .DocsURL}}<link rel="alternate" type="text/markdown" href="{{.DocsURL}}">{{end}}
</head>
<body style="font-family:system-ui,sans-serif;max-width:40rem;margin:2rem auto;padding:0 1rem">
<h1>getskillpack registry</h1>
<p>Центральный реестр skill-пакетов для <a href="https://github.com/getskillpack">getskillpack</a>.</p>
<ul>
{{if .DocsURL}}<li><a href="{{.DocsURL}}">Документация API (Markdown)</a></li>{{end}}
<li><a href="https://github.com/getskillpack/registry">Исходники на GitHub</a></li>
<li><a href="/healthz"><code>/healthz</code></a> · <a href="/readyz"><code>/readyz</code></a> · <a href="/api/v1/skills"><code>GET /api/v1/skills</code></a></li>
</ul>
</body>
</html>`))

type rootPageData struct {
	Base        string
	OGImageURL  string
	DocsURL     string
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	base := s.publicBase(r)
	data := rootPageData{Base: base + "/"}
	if len(s.OGCardSVG) > 0 {
		data.OGImageURL = base + "/og.svg"
	}
	if len(s.RegistryAPIMarkdown) > 0 {
		data.DocsURL = base + "/docs/registry-api"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = rootPageTmpl.Execute(w, data)
}

func (s *Server) handleRegistryAPIDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write(s.RegistryAPIMarkdown)
}

func (s *Server) handleOGCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	_, _ = w.Write(s.OGCardSVG)
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
