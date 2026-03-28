package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getskillpack/registry/internal/store"
)

func TestRegistryFlow(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("REGISTRY_PUBLIC_URL", "") // use request host

	srv := &Server{
		Store:       st,
		WriteToken:  "test-token",
		DefaultAddr: "127.0.0.1:8080",
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// Publish
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("manifest", `{"name":"demo-skill","version":"1.0.0","description":"d","author":"acme"}`)
	fw, err := w.CreateFormFile("archive", "demo.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("fake-tarball"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/skills", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("publish: %d %s", resp.StatusCode, b)
	}

	// List
	res, err := http.Get(ts.URL + "/api/v1/skills")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", res.StatusCode)
	}
	var list struct {
		Data []store.SkillSummary `json:"data"`
		Meta map[string]int       `json:"meta"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Meta["total"] != 1 || len(list.Data) != 1 || list.Data[0].Name != "demo-skill" {
		t.Fatalf("unexpected list: %+v", list)
	}

	// Get version
	res, err = http.Get(ts.URL + "/api/v1/skills/demo-skill/versions/1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get version: %d", res.StatusCode)
	}
	var ver store.VersionRecord
	if err := json.NewDecoder(res.Body).Decode(&ver); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(ver.ArchiveURL, ".tar.gz") || ver.Checksum == "" {
		t.Fatalf("version: %+v", ver)
	}

	// Download
	res, err = http.Get(ver.ArchiveURL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("download: %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if string(b) != "fake-tarball" {
		t.Fatalf("archive bytes mismatch")
	}

	// Yank
	yank, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/skills/demo-skill/versions/1.0.0", nil)
	yank.Header.Set("Authorization", "Bearer test-token")
	res, err = http.DefaultClient.Do(yank)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("yank: %d", res.StatusCode)
	}

	res, err = http.Get(ts.URL + "/api/v1/skills/demo-skill/versions/1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusGone {
		t.Fatalf("expected 410, got %d", res.StatusCode)
	}
}

func TestWriteDisabledWithoutToken(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Store: st, WriteToken: "", DefaultAddr: "127.0.0.1:8080"}
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/skills", strings.NewReader(""))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}

func TestPublicLandingAndDocs(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{
		Store:               st,
		DefaultAddr:         "127.0.0.1:8080",
		RegistryAPIMarkdown: []byte("# API\n"),
		OGCardSVG:           []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`),
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("root: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("root content-type: %q", ct)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "og:title") {
		t.Fatalf("root html missing og:title")
	}

	res, err = http.Get(ts.URL + "/docs/registry-api")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("docs: %d", res.StatusCode)
	}
	b, _ = io.ReadAll(res.Body)
	if string(b) != "# API\n" {
		t.Fatalf("docs body: %q", b)
	}

	res, err = http.Get(ts.URL + "/og.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("og: %d", res.StatusCode)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "image/svg+xml") {
		t.Fatalf("og content-type: %q", res.Header.Get("Content-Type"))
	}
}
