package config_test

import (
	"strings"
	"testing"

	"github.com/pria-bandung/single-sign-on-auth/internal/config"
)

// setRequired sets the minimum env vars that Load requires, so a test can focus
// on the behavior under test without Load failing for unrelated missing vars.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback")
}

func TestLoadAppliesDefaultsWhenRequiredVarsPresent(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
	if cfg.DatabasePath == "" {
		t.Error("DatabasePath = empty, want a non-empty default")
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want default false in dev")
	}
	if cfg.GoogleClientID != "client-id" {
		t.Errorf("GoogleClientID = %q, want %q", cfg.GoogleClientID, "client-id")
	}
}

func TestLoadFailsWhenRequiredVarMissing(t *testing.T) {
	setRequired(t)
	t.Setenv("GOOGLE_CLIENT_SECRET", "") // unset a required value

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() returned nil error, want an error naming the missing var")
	}
	if !strings.Contains(err.Error(), "GOOGLE_CLIENT_SECRET") {
		t.Errorf("error %q does not name the missing var GOOGLE_CLIENT_SECRET", err.Error())
	}
}
