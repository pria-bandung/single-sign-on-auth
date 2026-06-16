package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Sentinel errors returned by the store so callers can branch on them with
// errors.Is rather than inspecting driver-specific error strings.
var (
	// ErrNotFound is returned when a requested row does not exist.
	ErrNotFound = errors.New("store: not found")
	// ErrEmailTaken is returned when creating a user whose email already exists.
	ErrEmailTaken = errors.New("store: email already registered")
)

// User is a persisted account. PasswordHash, GoogleSub, and Name are nil when
// the corresponding column is NULL (e.g. a Google-only user has no PasswordHash).
type User struct {
	ID           int64
	Email        string
	PasswordHash *string
	GoogleSub    *string
	Name         *string
	CreatedAt    time.Time
}

// NewUser describes a user to create. Email is required; the pointer fields are
// optional and stored as NULL when nil.
type NewUser struct {
	Email        string
	PasswordHash *string
	GoogleSub    *string
	Name         *string
}

// CreateUser inserts a user and returns it with its generated ID. It returns
// ErrEmailTaken if the email already exists.
func (s *Store) CreateUser(ctx context.Context, u NewUser) (*User, error) {
	createdAt := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, google_sub, name, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		u.Email, ptrToNull(u.PasswordHash), ptrToNull(u.GoogleSub), ptrToNull(u.Name), formatTime(createdAt),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("user id: %w", err)
	}
	return &User{
		ID:           id,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		GoogleSub:    u.GoogleSub,
		Name:         u.Name,
		CreatedAt:    createdAt,
	}, nil
}

// FindUserByEmail returns the user with the given email, or ErrNotFound.
func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, google_sub, name, created_at
		 FROM users WHERE email = ?`, email)

	var (
		u                             User
		passwordHash, googleSub, name sql.NullString
		createdAt                     string
	)
	if err := row.Scan(&u.ID, &u.Email, &passwordHash, &googleSub, &name, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse user created_at: %w", err)
	}
	u.CreatedAt = created
	u.PasswordHash = nullToPtr(passwordHash)
	u.GoogleSub = nullToPtr(googleSub)
	u.Name = nullToPtr(name)
	return &u, nil
}

// UpsertGoogleUser resolves a Google sign-in to a single account by email,
// implementing the "lookup-or-create-or-link" rule:
//
//   - no user with that email      -> create a Google-only user (no password)
//   - user exists without a sub     -> link the Google account onto it
//   - user exists (already linked)  -> return it unchanged
//
// A name is filled in only when the existing account has none. The caller is
// responsible for verifying the email before calling this (see the auth layer).
func (s *Store) UpsertGoogleUser(ctx context.Context, email, sub, name string) (*User, error) {
	existing, err := s.FindUserByEmail(ctx, email)
	switch {
	case err == nil:
		if existing.GoogleSub == nil {
			if _, err := s.db.ExecContext(ctx, `UPDATE users SET google_sub = ? WHERE id = ?`, sub, existing.ID); err != nil {
				return nil, fmt.Errorf("link google account: %w", err)
			}
			existing.GoogleSub = &sub
		}
		if existing.Name == nil && name != "" {
			if _, err := s.db.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, name, existing.ID); err != nil {
				return nil, fmt.Errorf("set name: %w", err)
			}
			existing.Name = &name
		}
		return existing, nil
	case errors.Is(err, ErrNotFound):
		var namePtr *string
		if name != "" {
			namePtr = &name
		}
		return s.CreateUser(ctx, NewUser{Email: email, GoogleSub: &sub, Name: namePtr})
	default:
		return nil, err
	}
}

func ptrToNull(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func nullToPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}
