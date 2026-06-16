-- Schema for the SSO auth demo. Applied on startup; statements use
-- "IF NOT EXISTS" so applying it to an existing database is a safe no-op.

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT,            -- nullable: NULL for Google-only accounts
    google_sub    TEXT,            -- nullable: NULL for password-only accounts
    name          TEXT,
    created_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,   -- opaque random id stored in the cookie
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
