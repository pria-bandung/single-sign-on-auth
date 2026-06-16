package web

import (
	"net/http"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
)

const (
	oauthStateCookie = "oauth_state"
	oauthNextCookie  = "oauth_next"
	oauthCookieTTL   = 600 // seconds the OAuth handshake is allowed to take
)

// handleGoogleAuth starts the Google sign-in flow. It generates a random state
// value (CSRF defense), remembers it and the post-login destination in
// short-lived cookies, then redirects the user to Google's consent screen.
func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	state, err := auth.NewSessionID()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	next := validateNext(r.URL.Query().Get("next"))

	s.setOAuthCookie(w, oauthStateCookie, state)
	s.setOAuthCookie(w, oauthNextCookie, next)

	http.Redirect(w, r, s.google.AuthCodeURL(state), http.StatusFound)
}

// handleGoogleCallback completes the flow: it verifies the state against the
// cookie, exchanges the code for a token, fetches the (verified) identity, then
// looks-up-or-creates-or-links the account and starts a session.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Always clear the handshake cookies once we are back.
	defer func() {
		s.clearCookie(w, oauthStateCookie)
		s.clearCookie(w, oauthNextCookie)
	}()

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		s.failGoogle(w, r, "Google sign-in was cancelled or denied.")
		return
	}

	// Verify the CSRF state: the value Google echoed back must match our cookie.
	stateCookie, err := r.Cookie(oauthStateCookie)
	returnedState := r.URL.Query().Get("state")
	if err != nil || stateCookie.Value == "" || returnedState == "" || returnedState != stateCookie.Value {
		s.failGoogle(w, r, "Invalid sign-in state. Please try again.")
		return
	}

	token, err := s.google.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		s.failGoogle(w, r, "Could not complete Google sign-in. Please try again.")
		return
	}

	identity, err := s.google.FetchIdentity(r.Context(), token)
	if err != nil {
		s.failGoogle(w, r, "Could not read your Google profile. Please try again.")
		return
	}
	// Only trust the email for account matching if Google says it is verified.
	if !identity.EmailVerified || identity.Email == "" {
		s.failGoogle(w, r, "Your Google email is not verified.")
		return
	}

	user, err := s.store.UpsertGoogleUser(r.Context(), identity.Email, identity.Sub, identity.Name)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.startSession(w, r, user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	next := defaultNext
	if c, err := r.Cookie(oauthNextCookie); err == nil {
		next = validateNext(c.Value)
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// failGoogle sends the user back to the login page with an error message.
func (s *Server) failGoogle(w http.ResponseWriter, r *http.Request, msg string) {
	s.render(w, "login", loginData{Error: msg, Next: defaultNext})
}

func (s *Server) setOAuthCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   oauthCookieTTL,
	})
}

func (s *Server) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
