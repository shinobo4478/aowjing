// Package generations runs prompt templates through the AI provider and stores
// each run. Records are immutable: create, read, list, delete.
package generations

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

type Handler struct {
	q   *sqlc.Queries
	gen generate.Generator
}

func NewHandler(q *sqlc.Queries, gen generate.Generator) *Handler {
	return &Handler{q: q, gen: gen}
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

// run executes a template through the provider and records the result. A
// provider failure is still persisted (status "failed") and returned with 201,
// so the history always reflects what was attempted.
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

	params := sqlc.CreateGenerationParams{
		ProfileID:        tmpl.ProfileID,
		PromptTemplateID: templateID,
		InputPrompt:      tmpl.Body,
		Provider:         h.gen.Name(),
	}

	if res, genErr := h.gen.Generate(r.Context(), tmpl.Body); genErr != nil {
		params.Status = "failed"
		params.Error = genErr.Error()
	} else {
		params.Status = "succeeded"
		params.Output = res.Output
		params.Provider = res.Provider
		params.Model = res.Model
	}

	row, err := h.q.CreateGeneration(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save generation.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
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
