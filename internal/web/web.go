// Package web wires the HTTP layer: routing, handlers, middleware, and template
// rendering. It is the only package that speaks HTTP.
package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/pria-bandung/single-sign-on-auth/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

// pages lists the page templates that are composed with the shared layout. Each
// page template defines a "content" block.
var pages = []string{"home", "signup", "protected"}

const sessionCookieName = "session"

// Options configures a Server. Zero values fall back to sane defaults.
type Options struct {
	CookieSecure bool             // sets the Secure flag on the session cookie
	SessionTTL   time.Duration    // session lifetime; defaults to 24h
	Now          func() time.Time // injectable clock for tests; defaults to time.Now
}

// Server holds parsed templates, the store, and the HTTP router. It implements
// http.Handler.
type Server struct {
	mux          *http.ServeMux
	templates    map[string]*template.Template
	store        *store.Store
	cookieSecure bool
	sessionTTL   time.Duration
	now          func() time.Time
}

// pageData is the view model passed to every page render.
type pageData struct {
	Authenticated bool
	Email         string
}

// NewServer parses templates once, stores its dependencies, and registers
// routes. It returns an error if any template fails to parse.
func NewServer(st *store.Store, opts Options) (*Server, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	ttl := opts.SessionTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s := &Server{
		mux:          http.NewServeMux(),
		templates:    templates,
		store:        st,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   ttl,
		now:          now,
	}
	s.routes()
	return s, nil
}

// ServeHTTP makes Server an http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /signup", s.handleSignupForm)
	s.mux.HandleFunc("POST /signup", s.handleSignup)
	s.mux.HandleFunc("GET /protected", s.requireUser(s.handleProtected))
	s.mux.HandleFunc("POST /logout", s.handleLogout)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if u, ok := s.currentUser(r); ok {
		s.render(w, "home", pageData{Authenticated: true, Email: u.Email})
		return
	}
	s.render(w, "home", pageData{Authenticated: false})
}

func (s *Server) handleProtected(w http.ResponseWriter, r *http.Request, u *store.User) {
	s.render(w, "protected", pageData{Authenticated: true, Email: u.Email})
}

// render executes the named page (composed with the layout) into a buffer first,
// so a template error never leaves a half-written response to the client.
func (s *Server) render(w http.ResponseWriter, page string, data any) {
	t, ok := s.templates[page]
	if !ok {
		http.Error(w, "unknown page", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func parseTemplates() (map[string]*template.Template, error) {
	out := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("parse template %q: %w", page, err)
		}
		out[page] = t
	}
	return out, nil
}
