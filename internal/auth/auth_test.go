package auth_test

import (
	"testing"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == password {
		t.Fatal("hash equals plaintext; password was not hashed")
	}
	if !auth.VerifyPassword(hash, password) {
		t.Error("VerifyPassword rejected the correct password")
	}
	if auth.VerifyPassword(hash, "wrong password") {
		t.Error("VerifyPassword accepted a wrong password")
	}

	hash2, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword (second): %v", err)
	}
	if hash == hash2 {
		t.Error("two hashes of the same password are identical; expected a per-hash salt")
	}
}

func TestNewSessionIDIsUniqueAndNonEmpty(t *testing.T) {
	a, err := auth.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	b, err := auth.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("NewSessionID returned an empty id")
	}
	if a == b {
		t.Error("NewSessionID returned identical ids; expected unique values")
	}
}
