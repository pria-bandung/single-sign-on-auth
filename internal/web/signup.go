package web

import (
	"errors"
	"net/http"
	"net/mail"

	"github.com/pria-bandung/single-sign-on-auth/internal/auth"
	"github.com/pria-bandung/single-sign-on-auth/internal/store"
)

const minPasswordLen = 8

// signupData is the view model for the signup form, carrying any error and the
// previously typed email so the user need not retype it.
type signupData struct {
	Error string
	Email string
}

func (s *Server) handleSignupForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "signup", signupData{})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if _, err := mail.ParseAddress(email); err != nil {
		s.renderSignupError(w, email, "Please enter a valid email address.")
		return
	}
	if len(password) < minPasswordLen {
		s.renderSignupError(w, email, "Password must be at least 8 characters.")
		return
	}

	// Decide how to handle an email that already exists, distinguishing a
	// password account (already registered) from a Google-only account.
	if existing, err := s.store.FindUserByEmail(r.Context(), email); err == nil {
		if existing.PasswordHash != nil {
			s.renderSignupError(w, email, "An account with that email already exists. Please log in.")
		} else {
			s.renderSignupError(w, email, "An account with that email already exists. Please sign in with Google.")
		}
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	user, err := s.store.CreateUser(r.Context(), store.NewUser{Email: email, PasswordHash: &hash})
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			s.renderSignupError(w, email, "An account with that email already exists. Please log in.")
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.startSession(w, r, user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/protected", http.StatusSeeOther)
}

func (s *Server) renderSignupError(w http.ResponseWriter, email, msg string) {
	// A re-rendered form on a validation error is a 200 (the default status), so
	// render writes the body directly without overriding the status code.
	s.render(w, "signup", signupData{Error: msg, Email: email})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = s.store.DeleteSession(r.Context(), c.Value)
	}
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
