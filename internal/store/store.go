package store

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("version already exists")
	ErrYanked = errors.New("version yanked")
)

// SkillSummary matches GET /skills list items.
type SkillSummary struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Author         string    `json:"author"`
	LatestVersion  string    `json:"latest_version"`
	CreatedAt      time.Time `json:"created_at"`
}

// SkillDetail is returned by GET /skills/:name.
type SkillDetail struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Author      string                       `json:"author"`
	CreatedAt   time.Time                    `json:"created_at"`
	Versions    map[string]VersionPublicInfo `json:"versions"`
}

// VersionPublicInfo is exposed for each version in detail.
type VersionPublicInfo struct {
	Manifest    json.RawMessage `json:"manifest"`
	Checksum    string          `json:"checksum"`
	ArchiveURL  string          `json:"archive_url"`
	PublishedAt time.Time       `json:"published_at"`
	Yanked      bool            `json:"yanked"`
}

// VersionRecord is returned by GET /skills/:name/versions/:version.
type VersionRecord struct {
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Manifest   json.RawMessage `json:"manifest"`
	ArchiveURL string          `json:"archive_url"`
	Checksum   string          `json:"checksum"`
}

type ListParams struct {
	Query  string
	Author string
	Limit  int
	Offset int
}

type ListResult struct {
	Items []SkillSummary
	Total int
}

// Store is the registry persistence layer.
type Store interface {
	// Ping checks that persistence is reachable (readiness).
	Ping(ctx context.Context) error
	List(ctx context.Context, p ListParams) (ListResult, error)
	GetSkill(ctx context.Context, name, publicBaseURL string) (*SkillDetail, error)
	GetVersion(ctx context.Context, name, version, publicBaseURL string) (*VersionRecord, error)
	OpenArchive(ctx context.Context, sha256Hex string) (io.ReadCloser, int64, error)
	Publish(ctx context.Context, name, version string, description, author string, manifest json.RawMessage, archive []byte) error
	Yank(ctx context.Context, name, version string) error
}
