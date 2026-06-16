// Package web wires the HTTP layer: routing, handlers, middleware, and template
// rendering. It is the only package that speaks HTTP.
package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templateFS embed.FS

// pages lists the page templates that are composed with the shared layout. Each
// page template defines a "content" block.
var pages = []string{"home"}

// Server holds parsed templates and the HTTP router. It implements http.Handler.
type Server struct {
	mux       *http.ServeMux
	templates map[string]*template.Template
}

// pageData is the view model passed to every page render.
type pageData struct {
	Authenticated bool
	Email         string
}

// NewServer parses templates once and registers routes. It returns an error if
// any template fails to parse.
func NewServer() (*Server, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	s := &Server{
		mux:       http.NewServeMux(),
		templates: templates,
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
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Slice 1: no sessions yet, so the home page always renders logged-out.
	s.render(w, "home", pageData{Authenticated: false})
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
