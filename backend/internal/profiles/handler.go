// Package profiles implements the Profile CRUD HTTP endpoints.
package profiles

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// Handler serves /profiles. It depends only on the generated query interface,
// which keeps it easy to test with a fake.
type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

// Routes returns a router to be mounted at /profiles.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

// --- request payloads -------------------------------------------------------

type profileInput struct {
	Name        string `json:"name"`
	Niche       string `json:"niche"`
	Description string `json:"description"`
	// Which Generator this profile defaults to. Empty means "text".
	Provider string `json:"provider"`
}

// normalizeAndValidate trims fields and returns a map of field -> message.
// An empty map means the input is valid. Mirrors the frontend's rules so the
// two agree; the DB is still the last line of defence.
func (in *profileInput) normalizeAndValidate() map[string]string {
	in.Name = strings.TrimSpace(in.Name)
	in.Niche = strings.TrimSpace(in.Niche)
	in.Description = strings.TrimSpace(in.Description)
	in.Provider = strings.TrimSpace(in.Provider)
	if in.Provider == "" {
		in.Provider = "text"
	}

	errs := map[string]string{}
	switch {
	case len(in.Name) < 2:
		errs["name"] = "Name must be at least 2 characters."
	case len(in.Name) > 80:
		errs["name"] = "Name must be 80 characters or fewer."
	}
	if len(in.Niche) < 2 {
		errs["niche"] = "Niche must be at least 2 characters."
	}
	if len(in.Description) > 600 {
		errs["description"] = "Description must be 600 characters or fewer."
	}
	if !generate.ValidProvider(in.Provider) {
		errs["provider"] = "Unknown provider."
	}
	return errs
}

// --- handlers -------------------------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListProfiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list profiles.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": toDTOs(rows)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile id.")
		return
	}
	row, err := h.q.GetProfile(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": toDTO(row)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	if errs := in.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	row, err := h.q.CreateProfile(r.Context(), sqlc.CreateProfileParams{
		Name:        in.Name,
		Niche:       in.Niche,
		Description: in.Description,
		Provider:    in.Provider,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create profile.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"profile": toDTO(row)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile id.")
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	if errs := in.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	row, err := h.q.UpdateProfile(r.Context(), sqlc.UpdateProfileParams{
		ID:          id,
		Name:        in.Name,
		Niche:       in.Niche,
		Description: in.Description,
		Provider:    in.Provider,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update profile.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": toDTO(row)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile id.")
		return
	}
	n, err := h.q.DeleteProfile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete profile.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers -------------------------------------------------------------

// decodeInput reads a JSON body into profileInput, writing a 400 and returning
// ok=false on malformed input.
func decodeInput(w http.ResponseWriter, r *http.Request) (profileInput, bool) {
	var in profileInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return profileInput{}, false
	}
	return in, true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
