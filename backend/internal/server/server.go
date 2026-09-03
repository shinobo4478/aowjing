// Package server wires the HTTP router and its dependencies.
package server

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/profiles"
)

// Server holds the shared dependencies handlers need.
type Server struct {
	db *pgxpool.Pool
}

// New builds the HTTP handler for the API, with middleware and routes mounted.
func New(db *pgxpool.Pool) http.Handler {
	s := &Server{db: db}
	queries := sqlc.New(db)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/healthz", s.handleHealth)
	r.Mount("/profiles", profiles.NewHandler(queries).Routes())

	return r
}

// cors adds permissive CORS headers for the single-page frontend and
// short-circuits preflight requests. The allowed origin comes from
// CORS_ORIGIN (default http://localhost:3000); "*" allows any.
func cors(next http.Handler) http.Handler {
	origin := os.Getenv("CORS_ORIGIN")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
