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
		u.Email, ptrToNull(u.PasswordHash), ptrToNull(u.GoogleSub), ptrToNull(u.Name), createdAt,
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
	)
	if err := row.Scan(&u.ID, &u.Email, &passwordHash, &googleSub, &name, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.PasswordHash = nullToPtr(passwordHash)
	u.GoogleSub = nullToPtr(googleSub)
	u.Name = nullToPtr(name)
	return &u, nil
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
