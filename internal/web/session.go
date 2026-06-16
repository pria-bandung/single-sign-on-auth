package web

import (
	"net/http"
	"net/url"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
	"github.com/pria-bandung/single-sign-on-auth/internal/store"
)

// currentUser returns the authenticated user for the request, if any, by
// resolving the session cookie against the store.
func (s *Server) currentUser(r *http.Request) (*store.User, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil, false
	}
	u, err := s.store.FindUserBySession(r.Context(), c.Value, s.now())
	if err != nil {
		return nil, false
	}
	return u, true
}

// requireUser wraps a handler that needs an authenticated user. Unauthenticated
// requests are redirected to the login page with a validated "next" pointing
// back at the originally requested path.
func (s *Server) requireUser(next func(http.ResponseWriter, *http.Request, *store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
			return
		}
		next(w, r, u)
	}
}

// startSession creates a server-side session for the user and sets the session
// cookie on the response. It is the shared "log this user in" step used by every
// authentication path.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	id, err := auth.NewSessionID()
	if err != nil {
		return err
	}
	now := s.now()
	if err := s.store.CreateSession(r.Context(), store.Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionTTL),
	}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
	return nil
}

// clearSessionCookie expires the session cookie in the browser.
func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
