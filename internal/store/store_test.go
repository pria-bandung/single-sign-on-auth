package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

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

func TestUpsertGoogleUserCreatesWhenAbsent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	u, err := s.UpsertGoogleUser(ctx, "gina@example.com", "sub-123", "Gina")
	if err != nil {
		t.Fatalf("UpsertGoogleUser: %v", err)
	}
	if u.GoogleSub == nil || *u.GoogleSub != "sub-123" {
		t.Errorf("GoogleSub = %v, want %q", u.GoogleSub, "sub-123")
	}
	if u.PasswordHash != nil {
		t.Errorf("PasswordHash = %v, want nil for a Google-only account", *u.PasswordHash)
	}
	if u.Name == nil || *u.Name != "Gina" {
		t.Errorf("Name = %v, want %q", u.Name, "Gina")
	}

	got, err := s.FindUserByEmail(ctx, "gina@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("created user not findable by email (id %d vs %d)", got.ID, u.ID)
	}
}

func TestUpsertGoogleUserLinksExistingPasswordAccount(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	h := "bcrypt-hash"
	created, err := s.CreateUser(ctx, store.NewUser{Email: "link@example.com", PasswordHash: &h})
	if err != nil {
		t.Fatalf("seed password user: %v", err)
	}

	u, err := s.UpsertGoogleUser(ctx, "link@example.com", "sub-xyz", "Linda")
	if err != nil {
		t.Fatalf("UpsertGoogleUser: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("linked user id = %d, want the existing account %d (no duplicate)", u.ID, created.ID)
	}
	if u.PasswordHash == nil || *u.PasswordHash != h {
		t.Error("existing password was lost while linking the Google account")
	}
	if u.GoogleSub == nil || *u.GoogleSub != "sub-xyz" {
		t.Errorf("GoogleSub = %v, want %q after linking", u.GoogleSub, "sub-xyz")
	}
}

func TestUpsertGoogleUserIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a, err := s.UpsertGoogleUser(ctx, "x@example.com", "sub-1", "X")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	b, err := s.UpsertGoogleUser(ctx, "x@example.com", "sub-1", "X")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if a.ID != b.ID {
		t.Errorf("second upsert created a new account (%d vs %d)", b.ID, a.ID)
	}
}

// Timestamps must be stored as monotonic-clock-free, sortable UTC text so that
// the "expires_at > now" text comparison stays correct across server restarts.
// There is no interface that exposes a session's stored timestamps, so this
// storage-format property is asserted by reading the stored text directly.
func TestSessionTimestampsStoredAsSortableUTC(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx := context.Background()
	h := "h"
	u, err := s.CreateUser(ctx, store.NewUser{Email: "ts@example.com", PasswordHash: &h})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	// A local time carrying a monotonic clock reading, exactly like the web layer
	// produces from time.Now().
	now := time.Now()
	if err := s.CreateSession(ctx, store.Session{
		ID: "sid", UserID: u.ID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	_ = s.Close()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open for inspection: %v", err)
	}
	defer db.Close()

	var createdAt, expiresAt string
	if err := db.QueryRow(`SELECT created_at, expires_at FROM sessions WHERE id = 'sid'`).
		Scan(&createdAt, &expiresAt); err != nil {
		t.Fatalf("read raw timestamps: %v", err)
	}

	for _, ts := range []string{createdAt, expiresAt} {
		if strings.Contains(ts, "m=") {
			t.Errorf("stored timestamp %q still contains a monotonic clock reading", ts)
		}
		if !strings.HasSuffix(ts, "Z") {
			t.Errorf("stored timestamp %q is not UTC (expected a Z suffix)", ts)
		}
		if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
			t.Errorf("stored timestamp %q is not RFC3339: %v", ts, err)
		}
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
