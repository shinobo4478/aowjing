// Package channels implements Channel CRUD. A channel always belongs to a
// profile; the collection is addressed as /channels?profileId=<uuid> and items
// as /channels/{id}.
package channels

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

// Routes returns the router to mount at /channels.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

type channelInput struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Handle      string `json:"handle"`
	Description string `json:"description"`
}

func (in *channelInput) normalizeAndValidate() map[string]string {
	in.Name = strings.TrimSpace(in.Name)
	in.Platform = strings.TrimSpace(in.Platform)
	in.Handle = strings.TrimSpace(in.Handle)
	in.Description = strings.TrimSpace(in.Description)

	errs := map[string]string{}
	switch {
	case len(in.Name) < 2:
		errs["name"] = "Name must be at least 2 characters."
	case len(in.Name) > 80:
		errs["name"] = "Name must be 80 characters or fewer."
	}
	switch {
	case len(in.Platform) < 2:
		errs["platform"] = "Platform must be at least 2 characters."
	case len(in.Platform) > 40:
		errs["platform"] = "Platform must be 40 characters or fewer."
	}
	if len(in.Handle) > 80 {
		errs["handle"] = "Handle must be 80 characters or fewer."
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
		writeError(w, http.StatusInternalServerError, "Failed to list channels.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}

	rows, err := h.q.ListChannelsByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list channels.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": toDTOs(rows)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
		channelInput
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
		writeError(w, http.StatusInternalServerError, "Failed to create channel.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	if errs := body.channelInput.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	row, err := h.q.CreateChannel(r.Context(), sqlc.CreateChannelParams{
		ProfileID:   profileID,
		Name:        body.Name,
		Platform:    body.Platform,
		Handle:      body.Handle,
		Description: body.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create channel.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"channel": toDTO(row)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid channel id.")
		return
	}
	row, err := h.q.GetChannel(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Channel not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load channel.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": toDTO(row)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid channel id.")
		return
	}

	var in channelInput
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

	row, err := h.q.UpdateChannel(r.Context(), sqlc.UpdateChannelParams{
		ID:          id,
		Name:        in.Name,
		Platform:    in.Platform,
		Handle:      in.Handle,
		Description: in.Description,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Channel not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update channel.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": toDTO(row)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid channel id.")
		return
	}
	n, err := h.q.DeleteChannel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete channel.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Channel not found.")
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
