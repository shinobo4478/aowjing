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
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/spendcap"
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
	r.Get("/latest", h.latest) // before /{id}, which would take "latest" as an id
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
		"maxPictures":   studio.MaxRequestPictures,
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
		// CastIDs are the profile's cast members that appear in this picture (the
		// ticks in the dialog). Their pictures go with the request, named in the
		// prompt. Left out or empty makes the picture from the text alone.
		CastIDs []string `json:"castIds"`
		// ProductPictureIDs are stored pictures of catalog products (the ticked
		// pictures in the dialog; each belongs to a product's own pictures). They go
		// after the cast's, named with the product's name and the picture's label.
		// At most 2 different products.
		ProductPictureIDs []string `json:"productPictureIds"`
		// SceneImageIDs are stored pictures of the SAME scene to keep (the other
		// shots of a video), so a new shot shows it from another angle. At most
		// 2; they are sent before the cast's pictures.
		SceneImageIDs []string `json:"sceneImageIds"`
		// SameCamera, with scene pictures, asks for the strict wording that keeps the
		// camera, room and everything but the pose fixed: for an END picture, since a
		// clip morphs whatever differs between its first and last picture.
		SameCamera bool `json:"sameCamera"`
		// Context says what the pictures are for ("shot:<id>:start" or ":end"). The
		// request is then kept after the dialog closes, replaces the earlier request
		// for the same place, and is shown again when the dialog is reopened. Empty
		// = thrown away when the dialog closes (the Generate dialog).
		Context string `json:"context"`
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

	if body.Context != "" && !contextRE.MatchString(body.Context) {
		writeFieldError(w, "context", "Invalid context.")
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

	// The daily spending limit (Settings), before anything is saved or queued.
	if over, err := spendcap.Check(r.Context(), h.q, model.CostUSD(body.Count), time.Now()); err != nil {
		log.Printf("spendcap: %v", err)
	} else if over != nil {
		writeError(w, http.StatusPaymentRequired, over.Error())
		return
	}

	sceneIDs, ok := h.parseSceneImages(w, r, body.SceneImageIDs)
	if !ok {
		return
	}
	cast, castImageIDs, ok := h.resolveCast(w, r, profileID, body.CastIDs, len(sceneIDs))
	if !ok {
		return
	}
	productRefs, productImageIDs, ok := h.resolveProducts(w, r, body.ProductPictureIDs, len(sceneIDs)+len(castImageIDs))
	if !ok {
		return
	}
	// The product pictures are sent after the cast's, and the prompt numbers them
	// in the same order.
	cast = append(cast, productRefs...)
	castImageIDs = append(castImageIDs, productImageIDs...)

	row, err := h.q.CreateKeyframe(r.Context(), sqlc.CreateKeyframeParams{
		ProfileID: profileID,
		// The style's wording is added here, not in the worker, so the stored
		// prompt is exactly what the image model receives.
		Prompt:         studio.For(profile.VideoStyle).WithLook(profile.LookNotes).KeyframePrompt(prompt, len(sceneIDs), cast, body.SameCamera),
		Model:          model.Key,
		Count:          int16(body.Count),
		ReferenceCount: int16(len(sceneIDs) + len(castImageIDs)),
		// Never nil: both columns are NOT NULL and pgx sends a nil slice as NULL, so a
		// request with no scene or cast picture would be refused by the database.
		ExtraImageIds:  nonNilUUIDs(sceneIDs),
		CastImageIds:   nonNilUUIDs(castImageIDs),
		Context:        body.Context,
	})
	if err != nil {
		log.Printf("keyframes: saving the request: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save the request.")
		return
	}
	// The new request replaces the earlier ones kept for the same place, and requests
	// of shots that were deleted are swept away. Failures only leave stray pictures
	// behind, so they are logged and do not fail the request.
	if body.Context != "" {
		h.discardOlder(r.Context(), profileID, body.Context, row.ID)
	}
	h.sweepOrphans(r.Context())

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

// resolveCast turns the ticked cast ids into what the prompt and the worker
// need: each member described (role, name, note, how many pictures) and the
// pictures themselves in the request's order (the cast's own order, each
// member's pictures together). It checks the members belong to this profile,
// have a picture, and that scene + cast pictures stay within the cap. It writes
// the error response itself.
func (h *Handler) resolveCast(w http.ResponseWriter, r *http.Request, profileID pgtype.UUID, raw []string, sceneRefs int) ([]studio.CastRef, []pgtype.UUID, bool) {
	if len(raw) == 0 {
		return nil, nil, true
	}
	if len(raw) > studio.MaxCastMembers {
		writeFieldError(w, "castIds", "Too many cast members ticked.")
		return nil, nil, false
	}
	ids := make([]pgtype.UUID, 0, len(raw))
	seen := map[string]bool{}
	for _, s := range raw {
		id, err := pgconv.ParseUUID(s)
		if err != nil {
			writeFieldError(w, "castIds", "Invalid cast member id.")
			return nil, nil, false
		}
		if key := pgconv.UUIDString(id); !seen[key] {
			seen[key] = true
			ids = append(ids, id)
		}
	}
	members, err := h.q.ListCastMembersByIDs(r.Context(), sqlc.ListCastMembersByIDsParams{ProfileID: profileID, Column2: ids})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the cast.")
		return nil, nil, false
	}
	if len(members) != len(ids) {
		// Not found, or belongs to another profile: the same answer for both.
		writeFieldError(w, "castIds", "One of those cast members no longer exists.")
		return nil, nil, false
	}
	pics, err := h.q.ListCastPicturesByCast(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the cast pictures.")
		return nil, nil, false
	}
	byCast := map[string][]pgtype.UUID{}
	for _, p := range pics { // already ordered by cast, then position
		k := pgconv.UUIDString(p.CastID)
		byCast[k] = append(byCast[k], p.ImageID)
	}
	var (
		cast   []studio.CastRef
		images []pgtype.UUID
	)
	for _, m := range members {
		mp := byCast[pgconv.UUIDString(m.ID)]
		if len(mp) == 0 {
			writeFieldError(w, "castIds", m.Name+" has no pictures yet. Add one in the Cast tab or untick it.")
			return nil, nil, false
		}
		cast = append(cast, studio.CastRef{Role: studio.CastRole(m.Role), Name: m.Name, Note: m.Note, Pictures: len(mp)})
		images = append(images, mp...)
	}
	if sceneRefs+len(images) > studio.MaxRequestPictures {
		writeFieldError(w, "castIds", "Too many reference pictures ("+strconv.Itoa(sceneRefs+len(images))+", at most "+strconv.Itoa(studio.MaxRequestPictures)+"). Untick someone or remove a scene picture.")
		return nil, nil, false
	}
	return cast, images, true
}

// resolveProducts turns the ticked product picture ids into prompt entries (one
// per picture: the product's name and the picture's label) and the ids in the
// request's order. It checks every id is a picture of some product, no repeats,
// at most MaxPictureProducts different products, and that everything stays within
// the request cap (alreadyUsed = scene + cast pictures). It writes the error
// response itself.
func (h *Handler) resolveProducts(w http.ResponseWriter, r *http.Request, raw []string, alreadyUsed int) ([]studio.CastRef, []pgtype.UUID, bool) {
	if len(raw) == 0 {
		return nil, nil, true
	}
	ids := make([]pgtype.UUID, 0, len(raw))
	seen := map[string]bool{}
	for _, s := range raw {
		id, err := pgconv.ParseUUID(s)
		if err != nil {
			writeFieldError(w, "productPictureIds", "Invalid picture id.")
			return nil, nil, false
		}
		key := pgconv.UUIDString(id)
		if seen[key] {
			writeFieldError(w, "productPictureIds", "The same picture is listed twice.")
			return nil, nil, false
		}
		seen[key] = true
		ids = append(ids, id)
	}
	rows, err := h.q.ListProductPicturesByImageIDs(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the product pictures.")
		return nil, nil, false
	}
	byImage := make(map[string]sqlc.ListProductPicturesByImageIDsRow, len(rows))
	for _, row := range rows {
		byImage[pgconv.UUIDString(row.ImageID)] = row
	}
	products := map[string]bool{}
	refs := make([]studio.CastRef, 0, len(ids))
	for _, id := range ids { // keep the order that was asked for
		row, ok := byImage[pgconv.UUIDString(id)]
		if !ok {
			writeFieldError(w, "productPictureIds", "One of those product pictures no longer exists.")
			return nil, nil, false
		}
		products[pgconv.UUIDString(row.ProductID)] = true
		refs = append(refs, studio.CastRef{Role: studio.RoleProduct, Name: row.Name, Note: row.Label, Pictures: 1})
	}
	if len(products) > studio.MaxPictureProducts {
		writeFieldError(w, "productPictureIds", "A picture can show at most "+strconv.Itoa(studio.MaxPictureProducts)+" products.")
		return nil, nil, false
	}
	if total := alreadyUsed + len(ids); total > studio.MaxRequestPictures {
		writeFieldError(w, "productPictureIds", "Too many reference pictures ("+strconv.Itoa(total)+", at most "+strconv.Itoa(studio.MaxRequestPictures)+"). Untick someone or something.")
		return nil, nil, false
	}
	return refs, ids, true
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
	n, err := h.discard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the request.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Keyframe request not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// contextRE is what a context may look like: "shot:<uuid>:start" or ":end".
var contextRE = regexp.MustCompile(`^shot:[0-9a-f-]{36}:(start|end)$`)

// discard deletes a request and then every picture of it that nothing else
// uses; it returns how many requests were deleted (0 = there was none).
func (h *Handler) discard(ctx context.Context, id pgtype.UUID) (int64, error) {
	// Look first: the picture ids are gone once the request is deleted.
	imgs, err := h.q.ListKeyframeImages(ctx, id)
	if err != nil {
		return 0, err
	}
	n, err := h.q.DeleteKeyframe(ctx, id)
	if err != nil || n == 0 {
		return n, err
	}
	// A failure here only leaves a stray picture behind, so it is not returned.
	for _, im := range imgs {
		_ = h.q.DeleteImageIfUnused(ctx, im.ImageID)
	}
	return n, nil
}

// discardOlder drops the earlier requests kept for the same place as keep.
func (h *Handler) discardOlder(ctx context.Context, profileID pgtype.UUID, kctx string, keep pgtype.UUID) {
	ids, err := h.q.ListOlderKeyframesByContext(ctx, sqlc.ListOlderKeyframesByContextParams{
		ProfileID: profileID, Context: kctx, ID: keep,
	})
	if err != nil {
		log.Printf("keyframes: listing the older requests: %v", err)
		return
	}
	for _, id := range ids {
		if _, err := h.discard(ctx, id); err != nil {
			log.Printf("keyframes: discarding an older request: %v", err)
		}
	}
}

// sweepOrphans drops kept requests whose shot has been deleted.
func (h *Handler) sweepOrphans(ctx context.Context) {
	ids, err := h.q.ListOrphanKeyframes(ctx)
	if err != nil {
		log.Printf("keyframes: listing the orphan requests: %v", err)
		return
	}
	for _, id := range ids {
		if _, err := h.discard(ctx, id); err != nil {
			log.Printf("keyframes: sweeping an orphan request: %v", err)
		}
	}
}

// latest returns the newest request kept for a place (?profileId=…&context=…), or
// {"keyframe": null} when there is none, so reopening the dialog shows the
// pictures made last time.
func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId is required.")
		return
	}
	kctx := r.URL.Query().Get("context")
	if !contextRE.MatchString(kctx) {
		writeError(w, http.StatusBadRequest, "A valid context is required.")
		return
	}
	row, err := h.q.GetLatestKeyframeByContext(r.Context(), sqlc.GetLatestKeyframeByContextParams{
		ProfileID: profileID, Context: kctx,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"keyframe": nil})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the request.")
		return
	}
	imgs, err := h.q.ListKeyframeImages(r.Context(), row.ID)
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

// nonNilUUIDs returns ids, or an empty (non-nil) slice when there are none. pgx
// writes a nil slice as SQL NULL, which a NOT NULL uuid[] column refuses.
func nonNilUUIDs(ids []pgtype.UUID) []pgtype.UUID {
	if ids == nil {
		return []pgtype.UUID{}
	}
	return ids
}
