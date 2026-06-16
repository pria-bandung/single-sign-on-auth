// Package store owns all persistence for the application. It opens the SQLite
// database, applies the schema, and (in later slices) exposes typed methods for
// users and sessions backed by handwritten SQL.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

//go:embed schema.sql
var schema string

// Store is a handle to the application database.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at dsn and applies the
// schema. The schema uses "IF NOT EXISTS", so Open is safe to call against an
// existing database. It returns an error if the database cannot be reached or
// the schema cannot be applied.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying database connections.
func (s *Store) Close() error {
	return s.db.Close()
}
