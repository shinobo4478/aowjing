// Package settings serves the global key/value settings store — provider
// credentials and similar, one row per key, never per profile.
package settings

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
)

// knownKeys is the whitelist the API accepts. Add a key here when a new
// provider needs a credential.
var knownKeys = []string{"falApiKey"}

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

// Routes returns the router to mount at /settings.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.get)
	r.Put("/", h.put)
	return r
}

// current reads every known key, defaulting missing ones to "".
func (h *Handler) current(r *http.Request) (map[string]string, error) {
	out := make(map[string]string, len(knownKeys))
	for _, k := range knownKeys {
		out[k] = ""
	}
	rows, err := h.q.ListSettings(r.Context())
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, ok := out[row.Key]; ok {
			out[row.Key] = row.Value
		}
	}
	return out, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.current(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}

	for key := range body {
		if !slices.Contains(knownKeys, key) {
			writeError(w, http.StatusBadRequest, "Unknown setting: "+key)
			return
		}
	}

	for key, value := range body {
		if err := h.q.UpsertSetting(r.Context(), sqlc.UpsertSettingParams{
			Key:   key,
			Value: value,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save settings.")
			return
		}
	}

	settings, err := h.current(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Saved, but failed to reload settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
