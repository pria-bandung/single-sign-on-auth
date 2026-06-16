package web_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
	"github.com/pria-bandung/single-sign-on-auth/internal/store"
	"github.com/pria-bandung/single-sign-on-auth/internal/web"
)

func newTestServer(t *testing.T) (*web.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv, err := web.NewServer(st, web.Options{SessionTTL: time.Hour})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv, st
}

func postForm(srv http.Handler, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func get(srv http.Handler, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("no session cookie was set")
	return nil
}

func TestHomePageLoggedOutShowsAuthEntryPoints(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := get(srv, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{`href="/login"`, `href="/signup"`} {
		if !strings.Contains(body, want) {
			t.Errorf("logged-out home is missing %q\n--- body ---\n%s", want, body)
		}
	}
}

func TestSignupCreatesSessionAndGrantsProtectedAccess(t *testing.T) {
	srv, st := newTestServer(t)

	rec := postForm(srv, "/signup", url.Values{"email": {"new@example.com"}, "password": {"longenough"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("signup status = %d, want %d (redirect)", rec.Code, http.StatusSeeOther)
	}

	if _, err := st.FindUserByEmail(context.Background(), "new@example.com"); err != nil {
		t.Fatalf("user was not created: %v", err)
	}

	c := sessionCookie(t, rec)
	if !c.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie SameSite = %v, want Lax", c.SameSite)
	}

	prec := get(srv, "/protected", c)
	if prec.Code != http.StatusOK {
		t.Fatalf("protected status = %d, want %d", prec.Code, http.StatusOK)
	}
	if !strings.Contains(strings.ToLower(prec.Body.String()), "protected") {
		t.Errorf("protected page does not look like the protected page:\n%s", prec.Body.String())
	}
}

func TestSignupRejectsShortPassword(t *testing.T) {
	srv, st := newTestServer(t)
	rec := postForm(srv, "/signup", url.Values{"email": {"x@example.com"}, "password": {"short"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (re-render with error)", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "at least 8") {
		t.Errorf("missing password-length error:\n%s", body)
	}
	if !strings.Contains(body, "x@example.com") {
		t.Error("typed email was not preserved on the re-rendered form")
	}
	if _, err := st.FindUserByEmail(context.Background(), "x@example.com"); !errors.Is(err, store.ErrNotFound) {
		t.Error("a user should not be created when validation fails")
	}
}

func TestSignupRejectsInvalidEmail(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := postForm(srv, "/signup", url.Values{"email": {"not-an-email"}, "password": {"longenough"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "valid email") {
		t.Errorf("missing invalid-email error:\n%s", rec.Body.String())
	}
}

func TestSignupDuplicatePasswordAccount(t *testing.T) {
	srv, st := newTestServer(t)
	h, _ := auth.HashPassword("whatever")
	if _, err := st.CreateUser(context.Background(), store.NewUser{Email: "dup@example.com", PasswordHash: &h}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	rec := postForm(srv, "/signup", url.Values{"email": {"dup@example.com"}, "password": {"longenough"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "already exists") {
		t.Errorf("missing 'account already exists' message:\n%s", rec.Body.String())
	}
}

func TestProtectedRedirectsWhenUnauthenticated(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := get(srv, "/protected")
	if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want a redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/login") || !strings.Contains(loc, "next=") {
		t.Errorf("Location = %q, want a redirect to /login with a next param", loc)
	}
}

func TestLogoutClearsSessionAndCookie(t *testing.T) {
	srv, _ := newTestServer(t)
	signup := postForm(srv, "/signup", url.Values{"email": {"bye@example.com"}, "password": {"longenough"}})
	c := sessionCookie(t, signup)

	lrec := postForm(srv, "/logout", url.Values{}, c)
	if lrec.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want %d", lrec.Code, http.StatusSeeOther)
	}
	if cleared := sessionCookie(t, lrec); cleared.Value != "" {
		t.Errorf("logout did not clear the session cookie; value = %q", cleared.Value)
	}

	// The old cookie must no longer grant access.
	prec := get(srv, "/protected", c)
	if prec.Code == http.StatusOK {
		t.Error("protected page still accessible after logout")
	}
}

func TestHomeShowsWelcomeWhenAuthenticated(t *testing.T) {
	srv, _ := newTestServer(t)
	signup := postForm(srv, "/signup", url.Values{"email": {"me@example.com"}, "password": {"longenough"}})
	c := sessionCookie(t, signup)

	rec := get(srv, "/", c)
	body := rec.Body.String()
	if !strings.Contains(body, "me@example.com") {
		t.Errorf("authenticated home does not greet the user by email:\n%s", body)
	}
	if !strings.Contains(body, "/logout") {
		t.Error("authenticated home is missing a logout control")
	}
}
