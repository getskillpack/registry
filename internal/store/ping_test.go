package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_Ping(t *testing.T) {
	dir := t.TempDir()
	st, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFileStore_Ping_missingArchives(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "skills"), 0o755)
	st := &fileStore{root: dir, skills: map[string]*skillDisk{}}
	if err := st.Ping(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
