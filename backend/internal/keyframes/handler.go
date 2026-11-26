// Package keyframes makes the starting picture of a clip. The owner describes a
// scene, picks an image model and how many candidates, and the worker makes the
// pictures (using the profile's reference pictures so its character stays the
// same); the owner then picks one as the clip's starting image. The HTTP
// handler only enqueues work and reads results; Runner (in cmd/worker) does the
// work.
package keyframes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

const (
	// minPromptRunes: the image models reject a prompt under 3 characters.
	minPromptRunes = 3
	maxPromptRunes = 1500
)

// Enqueuer hands a keyframe request to the worker. *queue.Client satisfies it.
type Enqueuer interface {
	EnqueueKeyframeRun(ctx context.Context, keyframeID string) error
}

type Handler struct {
	q   *sqlc.Queries
	enq Enqueuer
}

func NewHandler(q *sqlc.Queries, enq Enqueuer) *Handler {
	return &Handler{q: q, enq: enq}
}

// Routes returns the router to mount at /keyframes.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/models", h.models)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Delete("/{id}", h.delete)
	return r
}

// models lists the image models and the limits, so the screen never repeats
// them (and shows the price before anything is spent).
func (h *Handler) models(w http.ResponseWriter, _ *http.Request) {
	type modelDTO struct {
		Key      string  `json:"key"`
		Label    string  `json:"label"`
		PriceUSD float64 `json:"priceUsd"` // per picture
	}
	ms := studio.KeyframeModels()
	out := make([]modelDTO, len(ms))
	for i, m := range ms {
		out[i] = modelDTO{Key: m.Key, Label: m.Label, PriceUSD: m.PriceUSD}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models":        out,
		"maxImages":     studio.MaxKeyframeImages,
		"maxReferences": studio.MaxReferences,
		"aspect":        studio.KeyframeAspect,
	})
}

// create records a pending request and hands it to the worker. Returns 202
// with the pending record; the client polls for the outcome.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
		Count     int    `json:"count"`
		// UseReferences: send the profile's reference pictures with the request.
		// Left out means yes; false makes the picture from the text alone.
		UseReferences *bool `json:"useReferences"`
		// SceneImageIDs are stored pictures of the SAME scene to keep (the other
		// shots of a video), so a new shot shows it from another angle. At most
		// 2; they are sent before the profile's reference pictures.
		SceneImageIDs []string `json:"sceneImageIds"`
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
	profile, err := h.q.GetProfile(r.Context(), profileID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}
	if profile.Provider != "fal" {
		writeFieldError(w, "profileId", "Keyframe pictures are for fal.ai video profiles; this profile uses "+profile.Provider+".")
		return
	}

	prompt := strings.TrimSpace(body.Prompt)
	if n := utf8.RuneCountInString(prompt); n < minPromptRunes {
		writeFieldError(w, "prompt", "Describe the picture (at least 3 characters).")
		return
	} else if n > maxPromptRunes {
		writeFieldError(w, "prompt", "The description must be 1500 characters or fewer.")
		return
	}
	model, ok := studio.KeyframeModelFor(body.Model)
	if !ok {
		writeFieldError(w, "model", "Unknown picture model.")
		return
	}
	if body.Count < 1 || body.Count > studio.MaxKeyframeImages {
		writeFieldError(w, "count", "Ask for 1 to 4 pictures.")
		return
	}

	lookRefs := 0
	if body.UseReferences == nil || *body.UseReferences {
		refs, err := h.q.ListProfileReferences(r.Context(), profileID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load the reference pictures.")
			return
		}
		lookRefs = len(refs)
	}
	sceneIDs, ok := h.parseSceneImages(w, r, body.SceneImageIDs)
	if !ok {
		return
	}

	row, err := h.q.CreateKeyframe(r.Context(), sqlc.CreateKeyframeParams{
		ProfileID: profileID,
		// The style's wording is added here, not in the worker, so the stored
		// prompt is exactly what the image model receives.
		Prompt:         studio.For(profile.VideoStyle).KeyframePrompt(prompt, lookRefs, len(sceneIDs)),
		Model:          model.Key,
		Count:          int16(body.Count),
		ReferenceCount: int16(lookRefs + len(sceneIDs)),
		ExtraImageIds:  sceneIDs,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save the request.")
		return
	}

	if err := h.enq.EnqueueKeyframeRun(r.Context(), pgconv.UUIDString(row.ID)); err != nil {
		// Nothing will process the row: mark it failed so the screen says why
		// instead of waiting forever.
		_ = h.q.FinishKeyframe(r.Context(), sqlc.FinishKeyframeParams{
			ID: row.ID, Status: "failed", Error: "could not queue the job",
		})
		writeError(w, http.StatusInternalServerError, "Could not queue the request.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"keyframe": toDTO(row, nil)})
}

// parseSceneImages checks the same-scene pictures: at most MaxSceneReferences,
// each a valid id that exists, no repeats. Order is kept (the first is the main
// one). It writes the error response itself.
func (h *Handler) parseSceneImages(w http.ResponseWriter, r *http.Request, raw []string) ([]pgtype.UUID, bool) {
	if len(raw) > studio.MaxSceneReferences {
		writeFieldError(w, "sceneImageIds", "Keep the scene of at most 2 pictures.")
		return nil, false
	}
	ids := make([]pgtype.UUID, 0, len(raw))
	seen := map[string]bool{}
	for _, s := range raw {
		id, err := pgconv.ParseUUID(s)
		if err != nil {
			writeFieldError(w, "sceneImageIds", "Invalid picture id.")
			return nil, false
		}
		key := pgconv.UUIDString(id)
		if seen[key] {
			writeFieldError(w, "sceneImageIds", "The same picture is listed twice.")
			return nil, false
		}
		seen[key] = true
		ok, err := h.q.ImageExists(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to check the pictures.")
			return nil, false
		}
		if !ok {
			writeFieldError(w, "sceneImageIds", "One of those pictures no longer exists.")
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid keyframe id.")
		return
	}
	row, err := h.q.GetKeyframe(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Keyframe request not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the request.")
		return
	}
	imgs, err := h.q.ListKeyframeImages(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the pictures.")
		return
	}
	ids := make([]pgtype.UUID, len(imgs))
	for i, im := range imgs {
		ids[i] = im.ImageID
	}
	writeJSON(w, http.StatusOK, map[string]any{"keyframe": toDTO(row, ids)})
}

// delete discards a request and every picture of it that nothing else uses. A
// picture already chosen for a clip is used by that clip, so it stays.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid keyframe id.")
		return
	}
	// Look first: the picture ids are gone once the request is deleted.
	imgs, err := h.q.ListKeyframeImages(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the request.")
		return
	}
	n, err := h.q.DeleteKeyframe(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the request.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Keyframe request not found.")
		return
	}
	// A failure here only leaves a stray picture behind, so it does not fail
	// the delete.
	for _, im := range imgs {
		_ = h.q.DeleteImageIfUnused(r.Context(), im.ImageID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeFieldError sends a 422 with a message for one input field, the shape
// the other handlers use for validation problems.
func writeFieldError(w http.ResponseWriter, field, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"errors": map[string]string{field: msg},
	})
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
