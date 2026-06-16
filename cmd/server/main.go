// Command server is the composition root for the SSO auth demo: it loads
// configuration, opens the database (auto-applying the schema), builds the HTTP
// server, and starts listening.
package main

import (
	"log"
	"net/http"

	"github.com/pria-bandung/single-sign-on-auth/internal/config"
	"github.com/pria-bandung/single-sign-on-auth/internal/store"
	"github.com/pria-bandung/single-sign-on-auth/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer db.Close()

	srv, err := web.NewServer()
	if err != nil {
		log.Fatalf("web server error: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
