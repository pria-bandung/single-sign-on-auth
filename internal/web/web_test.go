package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pria-bandung/single-sign-on-auth/internal/web"
)

func TestHomePageLoggedOutShowsAuthEntryPoints(t *testing.T) {
	srv, err := web.NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	for _, want := range []string{`href="/login"`, `href="/signup"`} {
		if !strings.Contains(body, want) {
			t.Errorf("logged-out home page is missing %q\n--- body ---\n%s", want, body)
		}
	}
}
