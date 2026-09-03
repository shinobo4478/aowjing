// Package prompttemplates implements Prompt Template CRUD. A template always
// belongs to a profile; the collection is /prompt-templates?profileId=<uuid>
// and items are /prompt-templates/{id}.
package prompttemplates

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

// Routes returns the router to mount at /prompt-templates.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

type templateInput struct {
	Name        string `json:"name"`
	Body        string `json:"body"`
	Description string `json:"description"`
}

func (in *templateInput) normalizeAndValidate() map[string]string {
	in.Name = strings.TrimSpace(in.Name)
	in.Body = strings.TrimSpace(in.Body)
	in.Description = strings.TrimSpace(in.Description)

	errs := map[string]string{}
	switch {
	case len(in.Name) < 2:
		errs["name"] = "Name must be at least 2 characters."
	case len(in.Name) > 120:
		errs["name"] = "Name must be 120 characters or fewer."
	}
	switch {
	case len(in.Body) < 1:
		errs["body"] = "Template body is required."
	case len(in.Body) > 5000:
		errs["body"] = "Template body must be 5000 characters or fewer."
	}
	if len(in.Description) > 600 {
		errs["description"] = "Description must be 600 characters or fewer."
	}
	return errs
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId query parameter is required.")
		return
	}
	if ok, err := h.q.ProfileExists(r.Context(), profileID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list prompt templates.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}

	rows, err := h.q.ListPromptTemplatesByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list prompt templates.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promptTemplates": toDTOs(rows)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
		templateInput
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}

	profileID, err := pgconv.ParseUUID(body.ProfileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId is required.")
		return
	}
	if ok, err := h.q.ProfileExists(r.Context(), profileID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create prompt template.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	if errs := body.templateInput.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	row, err := h.q.CreatePromptTemplate(r.Context(), sqlc.CreatePromptTemplateParams{
		ProfileID:   profileID,
		Name:        body.Name,
		Body:        body.Body,
		Description: body.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create prompt template.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"promptTemplate": toDTO(row)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid prompt template id.")
		return
	}
	row, err := h.q.GetPromptTemplate(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Prompt template not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load prompt template.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promptTemplate": toDTO(row)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid prompt template id.")
		return
	}

	var in templateInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	if errs := in.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	row, err := h.q.UpdatePromptTemplate(r.Context(), sqlc.UpdatePromptTemplateParams{
		ID:          id,
		Name:        in.Name,
		Body:        in.Body,
		Description: in.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Prompt template not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update prompt template.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promptTemplate": toDTO(row)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid prompt template id.")
		return
	}
	n, err := h.q.DeletePromptTemplate(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete prompt template.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Prompt template not found.")
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
