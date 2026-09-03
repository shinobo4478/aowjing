// Package server wires the HTTP router and its dependencies.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shinobo4478/aowjing/backend/internal/auth"
	"github.com/shinobo4478/aowjing/backend/internal/channels"
	"github.com/shinobo4478/aowjing/backend/internal/config"
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/generations"
	"github.com/shinobo4478/aowjing/backend/internal/profiles"
	"github.com/shinobo4478/aowjing/backend/internal/prompttemplates"
)

// Server holds the shared dependencies handlers need.
type Server struct {
	db *pgxpool.Pool
}

// New builds the HTTP handler for the API, with middleware and routes mounted.
func New(db *pgxpool.Pool, cfg config.Config) http.Handler {
	s := &Server{db: db}
	queries := sqlc.New(db)
	authr := auth.New(queries, cfg.AdminUsername, cfg.AdminPassword, cfg.CookieSecure)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors(cfg))

	// Public.
	r.Get("/healthz", s.handleHealth)
	r.Mount("/auth", authr.Routes())

	// Everything below requires a valid session.
	r.Group(func(r chi.Router) {
		r.Use(authr.Middleware)
		r.Mount("/profiles", profiles.NewHandler(queries).Routes())
		r.Mount("/channels", channels.NewHandler(queries).Routes())
		r.Mount("/prompt-templates", prompttemplates.NewHandler(queries).Routes())

		// TODO(phase-1): swap MockGenerator for a real provider once one is
		// chosen (Anthropic / OpenAI); pick it from config.
		r.Mount("/generations", generations.NewHandler(queries, generate.MockGenerator{}).Routes())
	})

	return r
}

// cors adds CORS headers for the single-page frontend and short-circuits
// preflight requests. The allowed origin comes from CORS_ORIGIN (default
// http://localhost:3000). Credentials are allowed so the session cookie flows;
// that rules out the "*" wildcard, so a concrete origin is always echoed.
func cors(cfg config.Config) func(http.Handler) http.Handler {
	origin := cfg.CORSOrigin
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
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
}
