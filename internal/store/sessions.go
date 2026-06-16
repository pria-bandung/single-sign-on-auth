package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Session is a server-side login session. The ID is the opaque value stored in
// the user's cookie.
type Session struct {
	ID        string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CreateSession inserts a session row.
func (s *Store) CreateSession(ctx context.Context, sess Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		sess.ID, sess.UserID, formatTime(sess.CreatedAt), formatTime(sess.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// FindUserBySession resolves an opaque session id to its owning user, but only
// if the session exists and has not expired as of now. Missing or expired
// sessions return ErrNotFound. This is the single lookup the auth middleware
// needs per request.
func (s *Store) FindUserBySession(ctx context.Context, id string, now time.Time) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.password_hash, u.google_sub, u.name, u.created_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = ? AND s.expires_at > ?`, id, formatTime(now))

	var (
		u                             User
		passwordHash, googleSub, name sql.NullString
		createdAt                     string
	)
	if err := row.Scan(&u.ID, &u.Email, &passwordHash, &googleSub, &name, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session user: %w", err)
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse session user created_at: %w", err)
	}
	u.CreatedAt = created
	u.PasswordHash = nullToPtr(passwordHash)
	u.GoogleSub = nullToPtr(googleSub)
	u.Name = nullToPtr(name)
	return &u, nil
}

// DeleteSession removes a session by id. Deleting a missing session is not an
// error (logout is idempotent).
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
