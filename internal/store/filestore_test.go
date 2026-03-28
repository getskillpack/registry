package store

import (
	"context"
	"testing"
)

func TestFileStorePing(t *testing.T) {
	dir := t.TempDir()
	st, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
