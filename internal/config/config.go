// Package config loads and validates application configuration from the
// environment, applying sensible development defaults and failing fast when a
// required value is absent.
package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the server.
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	Port               string
	DatabasePath       string
	CookieSecure       bool
}

// Load reads configuration from the environment. In development a .env file is
// loaded on a best-effort basis (its absence is not an error). Required values
// that are missing cause Load to return an error naming every missing key.
func Load() (*Config, error) {
	_ = godotenv.Load() // best-effort: dev convenience, ignored in production

	cfg := &Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Port:               envOr("PORT", "8080"),
		DatabasePath:       envOr("DATABASE_PATH", "app.db"),
		CookieSecure:       os.Getenv("COOKIE_SECURE") == "true",
	}

	required := map[string]string{
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"GOOGLE_REDIRECT_URL":  cfg.GoogleRedirectURL,
	}
	var missing []string
	for key, val := range required {
		if val == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// envOr returns the value of the environment variable named key, or def when it
// is unset or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
