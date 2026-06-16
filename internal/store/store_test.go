package store_test

import (
	"path/filepath"
	"testing"

	"github.com/pria-bandung/single-sign-on-auth/internal/store"
)

func TestOpenAppliesSchemaAndIsReRunnable(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")

	// First open on a fresh path must create the database and apply the schema.
	// A malformed schema would surface as an error here.
	s, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("first Open(%q): %v", dsn, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-opening the same database must succeed: the schema is auto-applied only
	// when absent, so startup is safe to repeat across restarts.
	s2, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("second Open(%q): %v", dsn, err)
	}
	t.Cleanup(func() { _ = s2.Close() })
}
