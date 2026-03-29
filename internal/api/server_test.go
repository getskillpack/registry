package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/getskillpack/registry/internal/store"
)

// publishMultipart builds POST /api/v1/skills with multipart manifest + archive file.
func publishMultipart(t *testing.T, url, bearerToken, manifest string, archive []byte) *http.Response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("manifest", manifest)
	fw, err := w.CreateFormFile("archive", "pkg.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func newPublishTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("REGISTRY_PUBLIC_URL", "")
	srv := &Server{
		Store:       st,
		WriteToken:  "good-token",
		DefaultAddr: "127.0.0.1:8080",
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, ts.URL + "/api/v1/skills"
}

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

// TestGETSkillDetailContract locks JSON shape for GET /api/v1/skills/{name} (compiled clients).
func TestGETSkillDetailContract(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("REGISTRY_PUBLIC_URL", "")

	srv := &Server{
		Store:       st,
		WriteToken:  "contract-token",
		DefaultAddr: "127.0.0.1:8080",
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	const skillName = "detail-contract-skill"
	manifest := `{"name":"` + skillName + `","version":"1.2.3","description":"d","author":"acme"}`
	pr := publishMultipart(t, ts.URL+"/api/v1/skills", "contract-token", manifest, []byte("skill-bytes"))
	pr.Body.Close()
	if pr.StatusCode != http.StatusCreated {
		t.Fatalf("publish: %d", pr.StatusCode)
	}

	res, err := http.Get(ts.URL + "/api/v1/skills/" + skillName)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get skill: %d", res.StatusCode)
	}
	var detail store.SkillDetail
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Name != skillName || detail.Description != "d" || detail.Author != "acme" {
		t.Fatalf("detail fields: %+v", detail)
	}
	if len(detail.Versions) != 1 {
		t.Fatalf("versions: want 1, got %d", len(detail.Versions))
	}
	v, ok := detail.Versions["1.2.3"]
	if !ok {
		t.Fatalf("missing version key 1.2.3 in %#v", detail.Versions)
	}
	if v.Yanked {
		t.Fatal("unexpected yanked")
	}
	if v.Checksum == "" || !strings.HasPrefix(v.Checksum, "sha256:") {
		t.Fatalf("checksum: %q", v.Checksum)
	}
	hex := strings.TrimPrefix(v.Checksum, "sha256:")
	if len(hex) != 64 {
		t.Fatalf("want 64 hex chars, got len %d", len(hex))
	}
	wantArchive := ts.URL + "/downloads/" + hex + ".tar.gz"
	if v.ArchiveURL != wantArchive {
		t.Fatalf("archive_url: want %q got %q", wantArchive, v.ArchiveURL)
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

// TestPublishYankWriteContract locks expected HTTP status codes for the write API (contract for clients).
func TestPublishYankWriteContract(t *testing.T) {
	ts, publishURL := newPublishTestServer(t)
	goodManifest := `{"name":"c-skill","version":"1.0.0","description":"d","author":"acme"}`

	t.Run("publish_missing_authorization", func(t *testing.T) {
		resp := publishMultipart(t, publishURL, "", goodManifest, []byte("x"))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 401, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_wrong_bearer", func(t *testing.T) {
		resp := publishMultipart(t, publishURL, "nope", goodManifest, []byte("x"))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 401, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_missing_manifest_field", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("archive", "pkg.tar.gz")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write([]byte("x"))
		_ = w.Close()
		req, _ := http.NewRequest(http.MethodPost, publishURL, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer good-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 400 missing manifest, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_invalid_manifest_json", func(t *testing.T) {
		resp := publishMultipart(t, publishURL, "good-token", `{not json`, []byte("x"))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 400 invalid json, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_manifest_missing_name", func(t *testing.T) {
		resp := publishMultipart(t, publishURL, "good-token", `{"version":"1.0.0","description":"d","author":"a"}`, []byte("x"))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 400, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_missing_archive", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("manifest", goodManifest)
		_ = w.Close()
		req, _ := http.NewRequest(http.MethodPost, publishURL, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer good-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 400 missing archive, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_empty_archive", func(t *testing.T) {
		resp := publishMultipart(t, publishURL, "good-token", `{"name":"empty-arch","version":"1.0.0","description":"d","author":"a"}`, nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("want 400 empty archive, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("publish_duplicate_version_conflict", func(t *testing.T) {
		m := `{"name":"dup-skill","version":"2.0.0","description":"d","author":"a"}`
		r1 := publishMultipart(t, publishURL, "good-token", m, []byte("one"))
		r1.Body.Close()
		if r1.StatusCode != http.StatusCreated {
			t.Fatalf("first publish: %d", r1.StatusCode)
		}
		r2 := publishMultipart(t, publishURL, "good-token", m, []byte("two"))
		defer r2.Body.Close()
		if r2.StatusCode != http.StatusConflict {
			b, _ := io.ReadAll(r2.Body)
			t.Fatalf("want 409 conflict, got %d: %s", r2.StatusCode, b)
		}
	})

	t.Run("yank_unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/skills/c-skill/versions/1.0.0", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("yank want 401, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("yank_not_found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/skills/no-such-skill/versions/9.9.9", nil)
		req.Header.Set("Authorization", "Bearer good-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("yank want 404, got %d: %s", resp.StatusCode, b)
		}
	})
}

// TestPublishYankSmokeRemote exercises POST multipart + DELETE yank against a real registry when env is set.
// REGISTRY_SMOKE_BASE_URL — base URL without path, e.g. https://registry.example.com
// REGISTRY_SMOKE_WRITE_TOKEN — Bearer secret accepted by that registry for writes.
func TestPublishYankSmokeRemote(t *testing.T) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("REGISTRY_SMOKE_BASE_URL")), "/")
	token := strings.TrimSpace(os.Getenv("REGISTRY_SMOKE_WRITE_TOKEN"))
	if base == "" || token == "" {
		t.Skip("skip remote smoke: set REGISTRY_SMOKE_BASE_URL and REGISTRY_SMOKE_WRITE_TOKEN")
	}
	name := fmt.Sprintf("smoke-contract-%d", time.Now().UnixNano())
	version := "0.0.1"
	manifest := fmt.Sprintf(`{"name":%q,"version":%q,"description":"smoke","author":"ci"}`, name, version)
	publishURL := base + "/api/v1/skills"

	resp := publishMultipart(t, publishURL, token, manifest, []byte("smoke-tarball-bytes"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("remote publish: want 201, got %d: %s", resp.StatusCode, b)
	}

	getURL := fmt.Sprintf("%s/api/v1/skills/%s/versions/%s", base, name, version)
	res, err := http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("remote get version: want 200, got %d", res.StatusCode)
	}

	yankURL := fmt.Sprintf("%s/api/v1/skills/%s/versions/%s", base, name, version)
	yank, _ := http.NewRequest(http.MethodDelete, yankURL, nil)
	yank.Header.Set("Authorization", "Bearer "+token)
	yres, err := http.DefaultClient.Do(yank)
	if err != nil {
		t.Fatal(err)
	}
	defer yres.Body.Close()
	if yres.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(yres.Body)
		t.Fatalf("remote yank: want 204, got %d: %s", yres.StatusCode, b)
	}

	gone, err := http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	defer gone.Body.Close()
	if gone.StatusCode != http.StatusGone {
		t.Fatalf("after yank GET version: want 410, got %d", gone.StatusCode)
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
