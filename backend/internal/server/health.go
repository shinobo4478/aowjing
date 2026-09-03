package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// handleHealth reports liveness plus a quick database round-trip. Returns 200
// when the DB responds, 503 otherwise, so a load balancer can route away from
// an instance that has lost its database connection.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	resp := healthResponse{Status: "ok", DB: "ok"}
	code := http.StatusOK

	if err := s.db.Ping(ctx); err != nil {
		resp.Status = "degraded"
		resp.DB = "unreachable"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, resp)
}

// writeJSON serialises v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
