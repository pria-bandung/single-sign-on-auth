package web

import (
	"net/http"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
)

// loginData is the view model for the login form. Next is carried as a hidden
// field so the user returns to the page they originally requested.
type loginData struct {
	Error string
	Email string
	Next  string
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login", loginData{Next: validateNext(r.URL.Query().Get("next"))})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	next := validateNext(r.FormValue("next"))

	user, err := s.store.FindUserByEmail(r.Context(), email)
	// A single generic error for both "no such user" and "wrong password" avoids
	// leaking which emails are registered (account enumeration).
	if err != nil || user.PasswordHash == nil || !auth.VerifyPassword(*user.PasswordHash, password) {
		s.render(w, "login", loginData{Error: "Invalid email or password.", Email: email, Next: next})
		return
	}

	if err := s.startSession(w, r, user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}
