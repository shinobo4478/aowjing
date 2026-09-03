// Package server wires the HTTP router and its dependencies.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds the shared dependencies handlers need.
type Server struct {
	db *pgxpool.Pool
}

// New builds the HTTP handler for the API, with middleware and routes mounted.
func New(db *pgxpool.Pool) http.Handler {
	s := &Server{db: db}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealth)

	return r
}
