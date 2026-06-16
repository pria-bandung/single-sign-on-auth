package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pria-bandung/single-sign-on-auth/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

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

func TestCreateUserIsFindableByEmail(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	hash := "bcrypt-hash"

	created, err := s.CreateUser(ctx, store.NewUser{Email: "alice@example.com", PasswordHash: &hash})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID = 0, want a generated id")
	}

	got, err := s.FindUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
	if got.PasswordHash == nil || *got.PasswordHash != hash {
		t.Errorf("PasswordHash = %v, want %q", got.PasswordHash, hash)
	}
	if got.GoogleSub != nil {
		t.Errorf("GoogleSub = %v, want nil for a password-only user", *got.GoogleSub)
	}
}

func TestFindUserByEmailNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.FindUserByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateUserDuplicateEmailRejected(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	h := "h"
	if _, err := s.CreateUser(ctx, store.NewUser{Email: "dup@example.com", PasswordHash: &h}); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err := s.CreateUser(ctx, store.NewUser{Email: "dup@example.com", PasswordHash: &h})
	if !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("duplicate CreateUser err = %v, want ErrEmailTaken", err)
	}
}

func TestSessionLifecycleResolvesUser(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	h := "h"
	u, err := s.CreateUser(ctx, store.NewUser{Email: "s@example.com", PasswordHash: &h})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := store.Session{ID: "sid-1", UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// A valid (unexpired) session resolves to its owning user.
	got, err := s.FindUserBySession(ctx, "sid-1", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("FindUserBySession before expiry: %v", err)
	}
	if got.ID != u.ID || got.Email != u.Email {
		t.Errorf("resolved user = %+v, want id %d / %q", got, u.ID, u.Email)
	}

	// Once expired, the session no longer resolves.
	if _, err := s.FindUserBySession(ctx, "sid-1", now.Add(2*time.Hour)); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired session err = %v, want ErrNotFound", err)
	}

	// After deletion (logout), the session no longer resolves.
	if err := s.DeleteSession(ctx, "sid-1"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := s.FindUserBySession(ctx, "sid-1", now.Add(time.Minute)); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleted session err = %v, want ErrNotFound", err)
	}
}
