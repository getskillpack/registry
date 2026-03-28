package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

var _ Store = (*fileStore)(nil)

type fileStore struct {
	root   string
	mu     sync.Mutex
	skills map[string]*skillDisk // name -> skill
}

type skillDisk struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Author      string                    `json:"author"`
	CreatedAt   time.Time                 `json:"created_at"`
	Versions    map[string]*versionDisk   `json:"versions"`
	path        string                    `json:"-"`
}

type versionDisk struct {
	Manifest    json.RawMessage `json:"manifest"`
	Checksum    string          `json:"checksum"` // "sha256:hex"
	ArchiveFile string          `json:"archive_file"`
	PublishedAt time.Time       `json:"published_at"`
	Yanked      bool            `json:"yanked"`
}

// NewFileStore loads registry state from root (default ./data).
func NewFileStore(root string) (Store, error) {
	return newFileStore(root)
}

func newFileStore(root string) (*fileStore, error) {
	if root == "" {
		root = "data"
	}
	s := &fileStore{
		root:   root,
		skills: make(map[string]*skillDisk),
	}
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "archives"), 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(root, "skills", e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var sk skillDisk
		if err := json.Unmarshal(b, &sk); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		if sk.Versions == nil {
			sk.Versions = make(map[string]*versionDisk)
		}
		sk.path = p
		if sk.Name == "" {
			sk.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		s.skills[sk.Name] = &sk
	}
	return s, nil
}

func (s *fileStore) persistSkillLocked(sk *skillDisk) error {
	if sk.path == "" {
		sk.path = filepath.Join(s.root, "skills", safeFilename(sk.Name)+".json")
	}
	b, err := json.MarshalIndent(sk, "", "  ")
	if err != nil {
		return err
	}
	tmp := sk.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, sk.path)
}

func safeFilename(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "/", "_"), "\\", "_")
}

func (s *fileStore) List(ctx context.Context, p ListParams) (ListResult, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []SkillSummary
	q := strings.ToLower(strings.TrimSpace(p.Query))
	author := strings.ToLower(strings.TrimSpace(p.Author))
	for _, sk := range s.skills {
		if author != "" && strings.ToLower(sk.Author) != author {
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(sk.Name), q) &&
				!strings.Contains(strings.ToLower(sk.Description), q) {
				continue
			}
		}
		lv := latestNonYanked(sk)
		if lv == "" {
			continue
		}
		out = append(out, SkillSummary{
			Name:          sk.Name,
			Description: sk.Description,
			Author:        sk.Author,
			LatestVersion: lv,
			CreatedAt:     sk.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	total := len(out)
	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	off := p.Offset
	if off < 0 {
		off = 0
	}
	if off > len(out) {
		off = len(out)
	}
	end := off + limit
	if end > len(out) {
		end = len(out)
	}
	return ListResult{Items: out[off:end], Total: total}, nil
}

func latestNonYanked(sk *skillDisk) string {
	var versions []string
	for v, rec := range sk.Versions {
		if rec == nil || rec.Yanked {
			continue
		}
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return ""
	}
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(canonicalV(versions[i]), canonicalV(versions[j])) < 0
	})
	return versions[len(versions)-1]
}

func canonicalV(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func (s *fileStore) GetSkill(ctx context.Context, name, publicBaseURL string) (*SkillDetail, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	sk, ok := s.skills[name]
	if !ok || sk == nil {
		return nil, ErrNotFound
	}
	pub := strings.TrimRight(publicBaseURL, "/")
	vers := make(map[string]VersionPublicInfo)
	for v, rec := range sk.Versions {
		if rec == nil {
			continue
		}
		hexSum := strings.TrimPrefix(rec.Checksum, "sha256:")
		vers[v] = VersionPublicInfo{
			Manifest:    rec.Manifest,
			Checksum:    rec.Checksum,
			ArchiveURL:  fmt.Sprintf("%s/downloads/%s.tar.gz", pub, hexSum),
			PublishedAt: rec.PublishedAt,
			Yanked:      rec.Yanked,
		}
	}
	return &SkillDetail{
		Name:        sk.Name,
		Description: sk.Description,
		Author:      sk.Author,
		CreatedAt:   sk.CreatedAt,
		Versions:    vers,
	}, nil
}

func (s *fileStore) GetVersion(ctx context.Context, name, version, publicBaseURL string) (*VersionRecord, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	sk, ok := s.skills[name]
	if !ok || sk == nil {
		return nil, ErrNotFound
	}
	rec, ok := sk.Versions[version]
	if !ok || rec == nil {
		return nil, ErrNotFound
	}
	if rec.Yanked {
		return nil, ErrYanked
	}
	hexSum := strings.TrimPrefix(rec.Checksum, "sha256:")
	pub := strings.TrimRight(publicBaseURL, "/")
	return &VersionRecord{
		Name:       name,
		Version:    version,
		Manifest:   rec.Manifest,
		Checksum:   rec.Checksum,
		ArchiveURL: fmt.Sprintf("%s/downloads/%s.tar.gz", pub, hexSum),
	}, nil
}

func (s *fileStore) OpenArchive(ctx context.Context, sha256Hex string) (io.ReadCloser, int64, error) {
	_ = ctx
	path := filepath.Join(s.root, "archives", sha256Hex+".tar.gz")
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	return f, st.Size(), nil
}

func (s *fileStore) Publish(ctx context.Context, name, version, description, author string, manifest json.RawMessage, archive []byte) error {
	_ = ctx
	if name == "" || version == "" {
		return fmt.Errorf("name and version required")
	}
	if !semver.IsValid(canonicalV(version)) {
		return fmt.Errorf("invalid semver version")
	}
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	checksum := "sha256:" + hexSum

	s.mu.Lock()
	defer s.mu.Unlock()
	sk, ok := s.skills[name]
	if !ok {
		sk = &skillDisk{
			Name:        name,
			Description: description,
			Author:      author,
			CreatedAt:   time.Now().UTC().Truncate(time.Second),
			Versions:    make(map[string]*versionDisk),
		}
		s.skills[name] = sk
	} else {
		if description != "" {
			sk.Description = description
		}
		if author != "" {
			sk.Author = author
		}
	}
	if _, exists := sk.Versions[version]; exists {
		return ErrConflict
	}
	archDir := filepath.Join(s.root, "archives")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		return err
	}
	archPath := filepath.Join(archDir, hexSum+".tar.gz")
	if err := os.WriteFile(archPath, archive, 0o644); err != nil {
		return err
	}
	sk.Versions[version] = &versionDisk{
		Manifest:    json.RawMessage(manifest),
		Checksum:    checksum,
		ArchiveFile: hexSum + ".tar.gz",
		PublishedAt: time.Now().UTC().Truncate(time.Second),
		Yanked:      false,
	}
	return s.persistSkillLocked(sk)
}

func (s *fileStore) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fi, err := os.Stat(s.root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("registry data root is not a directory")
	}
	for _, sub := range []string{"skills", "archives"} {
		p := filepath.Join(s.root, sub)
		st, err := os.Stat(p)
		if err != nil {
			return err
		}
		if !st.IsDir() {
			return fmt.Errorf("registry %s is not a directory", sub)
		}
	}
	return nil
}

func (s *fileStore) Yank(ctx context.Context, name, version string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	sk, ok := s.skills[name]
	if !ok || sk == nil {
		return ErrNotFound
	}
	rec, ok := sk.Versions[version]
	if !ok || rec == nil {
		return ErrNotFound
	}
	rec.Yanked = true
	return s.persistSkillLocked(sk)
}
