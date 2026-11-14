package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"financesite/main/api/handler"
	"financesite/main/internal/config"
	"financesite/main/internal/demo"
	"financesite/main/internal/plaid"
	"financesite/main/internal/store"
)

// spa serves the built UI, falling back to index.html for client-side routes.
func spa(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+r.URL.Path))); err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}

func main() {
	cfg := config.Load()
	api := &handler.API{Cfg: cfg}

	if cfg.PlaidConfigured() {
		st, err := store.Open(cfg.DataDir)
		if err != nil {
			log.Fatalf("opening data directory %s: %v", cfg.DataDir, err)
		}
		api.Store = st
		api.Plaid = plaid.New(cfg.PlaidEnv, cfg.PlaidClientID, cfg.PlaidSecret)
		log.Printf("Plaid configured (%s). Data in %s", cfg.PlaidEnv, cfg.DataDir)
	} else {
		api.Store = store.NewMemory()
		if err := demo.Seed(api.Store, time.Now()); err != nil {
			log.Fatalf("seeding demo data: %v", err)
		}
		log.Print("Demo mode: PLAID_CLIENT_ID / PLAID_SECRET not set, serving sample data (nothing is saved)")
	}

	mux := http.NewServeMux()
	api.Routes(mux)
	mux.Handle("/", spa(cfg.UIDir))

	srv := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("Listening on http://%s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}
