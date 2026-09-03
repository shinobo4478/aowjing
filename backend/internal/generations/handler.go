// Package generations records each run of a prompt template through a provider.
// The HTTP handler only enqueues work and reads history; cmd/worker executes
// the run via Runner. Records are immutable except for the worker filling in
// the outcome.
package generations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// Enqueuer hands a generation off to the worker. *queue.Client satisfies it.
type Enqueuer interface {
	EnqueueGenerationRun(ctx context.Context, generationID string) error
}

type Handler struct {
	q   *sqlc.Queries
	enq Enqueuer
}

func NewHandler(q *sqlc.Queries, enq Enqueuer) *Handler {
	return &Handler{q: q, enq: enq}
}

// Routes returns the router to mount at /generations.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.run)
	r.Get("/{id}", h.get)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId query parameter is required.")
		return
	}
	if ok, err := h.q.ProfileExists(r.Context(), profileID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list generations.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}

	rows, err := h.q.ListGenerationsByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list generations.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"generations": listRowDTOs(rows)})
}

// run creates a pending generation and hands it to the worker. Returns 202
// with the pending record; the client polls for the outcome.
func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PromptTemplateID string `json:"promptTemplateId"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}

	templateID, err := pgconv.ParseUUID(body.PromptTemplateID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid promptTemplateId is required.")
		return
	}

	tmpl, err := h.q.GetPromptTemplate(r.Context(), templateID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Prompt template not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load prompt template.")
		return
	}

	// The profile's provider is the intended generator; the worker reads it.
	profile, err := h.q.GetProfile(r.Context(), tmpl.ProfileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}

	row, err := h.q.CreateGeneration(r.Context(), sqlc.CreateGenerationParams{
		ProfileID:        tmpl.ProfileID,
		PromptTemplateID: templateID,
		InputPrompt:      tmpl.Body,
		Status:           "pending",
		Provider:         profile.Provider,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save generation.")
		return
	}

	genID := pgconv.UUIDString(row.ID)
	if err := h.enq.EnqueueGenerationRun(r.Context(), genID); err != nil {
		// The row exists but nothing will process it — mark it failed so the
		// user sees why rather than a stuck "pending".
		_ = h.q.FinishGeneration(r.Context(), sqlc.FinishGenerationParams{
			ID:       row.ID,
			Status:   "failed",
			Error:    "could not queue the job",
			Provider: row.Provider,
		})
		writeError(w, http.StatusInternalServerError, "Could not queue the generation.")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"generation": modelDTO(row, tmpl.Name),
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid generation id.")
		return
	}
	row, err := h.q.GetGeneration(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Generation not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load generation.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"generation": getRowDTO(row)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid generation id.")
		return
	}
	n, err := h.q.DeleteGeneration(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete generation.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Generation not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
