// Package videos serves a video: an ordered list of shots, each with its own
// prompt, length and starting picture (Phase 4 item 3). A shot is a PLAN; each
// press of Generate on it makes an ordinary generation (generations.shot_id), so
// retakes keep their history and cost. Pressing Generate only enqueues; the
// worker runs it like any other clip.
package videos

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// Enqueuer hands a generation to the worker. *queue.Client satisfies it.
type Enqueuer interface {
	EnqueueGenerationRun(ctx context.Context, generationID string) error
}

type Handler struct {
	db  *pgxpool.Pool
	q   *sqlc.Queries
	enq Enqueuer
}

func NewHandler(db *pgxpool.Pool, q *sqlc.Queries, enq Enqueuer) *Handler {
	return &Handler{db: db, q: q, enq: enq}
}

// Routes returns the router to mount at /videos.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.rename)
	r.Delete("/{id}", h.delete)

	r.Post("/{id}/shots", h.addShot)
	r.Put("/{id}/shots/order", h.reorder)
	r.Patch("/{id}/shots/{shotId}", h.updateShot)
	r.Delete("/{id}/shots/{shotId}", h.deleteShot)
	r.Post("/{id}/shots/{shotId}/generate", h.generateShot)
	r.Put("/{id}/shots/{shotId}/approved", h.setApproved)
	r.Post("/{id}/generate", h.generateShots)
	return r
}

// --- videos -----------------------------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId query parameter is required.")
		return
	}
	if ok, err := h.q.ProfileExists(r.Context(), profileID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list videos.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return
	}
	rows, err := h.q.ListVideosByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list videos.")
		return
	}
	out := make([]videoDTO, len(rows))
	for i, row := range rows {
		out[i] = toVideoDTO(sqlc.Video{
			ID: row.ID, ProfileID: row.ProfileID, Title: row.Title,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}, int(row.ShotCount), int(row.TotalSeconds))
	}
	writeJSON(w, http.StatusOK, map[string]any{"videos": out})
}

// create makes a video with one empty shot, so the page it opens is not blank.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
		Title     string `json:"title"`
	}
	if !decode(w, r, &body) {
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
		writeFieldError(w, "profileId", "Videos are for fal.ai video profiles; this profile uses "+profile.Provider+".")
		return
	}
	title, err := validateTitle(body.Title)
	if err != nil {
		writeFieldError(w, "title", err.Error())
		return
	}

	v, err := h.q.CreateVideo(r.Context(), sqlc.CreateVideoParams{ProfileID: profileID, Title: title})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create the video.")
		return
	}
	if _, err := h.q.CreateShot(r.Context(), sqlc.CreateShotParams{
		VideoID: v.ID, Seconds: studio.DefaultClipSeconds,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "The video was created but its first shot was not.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"video": toVideoDTO(v, 1, studio.DefaultClipSeconds)})
}

// get returns a video with its shots in order, each with its newest attempt.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shots, err := h.q.ListShots(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}
	latest, err := h.latestByShot(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}

	out := make([]shotDTO, len(shots))
	total := 0
	for i, s := range shots {
		out[i] = toShotDTO(s, latest[pgconv.UUIDString(s.ID)])
		total += int(s.Seconds)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"video": toVideoDTO(v, len(shots), total),
		"shots": out,
	})
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if !decode(w, r, &body) {
		return
	}
	title, err := validateTitle(body.Title)
	if err != nil {
		writeFieldError(w, "title", err.Error())
		return
	}
	updated, err := h.q.UpdateVideoTitle(r.Context(), sqlc.UpdateVideoTitleParams{ID: v.ID, Title: title})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to rename the video.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"video": toVideoDTO(updated, 0, 0)})
}

// delete removes the video and its shots. The clips already made from its shots
// stay (they were paid for) as ordinary clips of the profile.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	// Look first: the shots' pictures are unreachable once the video is gone.
	imageIDs, err := h.q.ListVideoImageIDs(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the video.")
		return
	}
	if _, err := h.q.DeleteVideo(r.Context(), v.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the video.")
		return
	}
	// A failure here only leaves a stray picture behind.
	for _, id := range imageIDs {
		_ = h.q.DeleteImageIfUnused(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- shots ------------------------------------------------------------------

func (h *Handler) addShot(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	var in shotInput
	if !decodeOptional(w, r, &in) {
		return
	}
	if errs := in.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	n, err := h.q.CountShots(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add the shot.")
		return
	}
	if int(n) >= MaxShots {
		writeFieldError(w, "shots", "A video can have at most 12 shots.")
		return
	}
	imageID, ok := h.resolveImage(w, r, in)
	if !ok {
		return
	}

	shot, err := h.q.CreateShot(r.Context(), sqlc.CreateShotParams{
		VideoID: v.ID, Prompt: in.Prompt, Seconds: int16(in.Seconds),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add the shot.")
		return
	}
	// A picture given with the new shot goes on with the update (CreateShot has
	// no picture columns, to keep its query simple).
	if in.ImageURL != "" || imageID.Valid {
		shot, err = h.q.UpdateShot(r.Context(), sqlc.UpdateShotParams{
			ID: shot.ID, Prompt: shot.Prompt, Seconds: shot.Seconds,
			ImageUrl: in.ImageURL, ImageID: imageID,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "The shot was added but its picture was not.")
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"shot": toShotDTO(shot, nil)})
}

func (h *Handler) updateShot(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	var in shotInput
	if !decode(w, r, &in) {
		return
	}
	if errs := in.normalizeAndValidate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	imageID, ok := h.resolveImage(w, r, in)
	if !ok {
		return
	}

	updated, err := h.q.UpdateShot(r.Context(), sqlc.UpdateShotParams{
		ID: shot.ID, Prompt: in.Prompt, Seconds: int16(in.Seconds),
		ImageUrl: in.ImageURL, ImageID: imageID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update the shot.")
		return
	}
	// If the shot swapped to another picture, drop the old one when nothing
	// else uses it.
	if shot.ImageID.Valid && shot.ImageID != imageID {
		_ = h.q.DeleteImageIfUnused(r.Context(), shot.ImageID)
	}
	latest, _ := h.latestByShot(r.Context(), v.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"shot": toShotDTO(updated, latest[pgconv.UUIDString(updated.ID)]),
	})
}

func (h *Handler) deleteShot(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	if _, err := h.q.DeleteShot(r.Context(), shot.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete the shot.")
		return
	}
	if err := h.q.CompactShotPositions(r.Context(), v.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "The shot was deleted but the order was not tidied.")
		return
	}
	if shot.ImageID.Valid {
		_ = h.q.DeleteImageIfUnused(r.Context(), shot.ImageID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// reorder sets the shots' order from a list naming every shot once. It runs in
// one transaction: positions are unique per video, and the constraint is
// deferred, so shots can swap places inside it.
func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	var body struct {
		ShotIDs []string `json:"shotIds"`
	}
	if !decode(w, r, &body) {
		return
	}
	shots, err := h.q.ListShots(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reorder.")
		return
	}
	have := make([]string, len(shots))
	for i, s := range shots {
		have[i] = pgconv.UUIDString(s.ID)
	}
	if !sameSet(have, body.ShotIDs) {
		writeFieldError(w, "shotIds", "List every shot of the video exactly once.")
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reorder.")
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck // a no-op after Commit
	qtx := h.q.WithTx(tx)
	for i, idStr := range body.ShotIDs {
		id, err := pgconv.ParseUUID(idStr)
		if err != nil {
			writeFieldError(w, "shotIds", "Invalid shot id.")
			return
		}
		if err := qtx.SetShotPosition(r.Context(), sqlc.SetShotPositionParams{
			ID: id, VideoID: v.ID, Position: int16(i + 1),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to reorder.")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reorder.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// generateShot makes one attempt at a shot: a pending generation from the
// shot's current plan, queued for the worker. This is what spends money, so the
// screen asks first; here the guards are that the shot has what it needs and is
// not already being made (a double click must not pay twice).
func (h *Handler) generateShot(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}

	profile, err := h.q.GetProfile(r.Context(), v.ProfileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}
	latest, err := h.latestByShot(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check the shot.")
		return
	}
	attempt, qe := h.queueShot(r.Context(), v, profile, shot, latest[pgconv.UUIDString(shot.ID)])
	if qe != nil {
		qe.write(w)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"attempt": attempt})
}

// queueError says why a shot could not be queued, in the shape the handlers
// answer with: a message on one field (422), or a plain error with its status.
type queueError struct {
	code  int
	field string
	msg   string
}

func (e *queueError) write(w http.ResponseWriter) {
	if e.field != "" {
		writeFieldError(w, e.field, e.msg)
		return
	}
	writeError(w, e.code, e.msg)
}

// queueShot is the one place a shot becomes a paid attempt: it checks the shot
// has what it needs and is not already being made (a double click must not pay
// twice), saves a pending generation from the shot's current plan, and hands it
// to the worker. `latest` is the shot's newest attempt, or nil.
func (h *Handler) queueShot(ctx context.Context, v sqlc.Video, profile sqlc.Profile, shot sqlc.VideoShot, latest *sqlc.ListShotLatestGenerationsRow) (attemptDTO, *queueError) {
	if profile.Provider != "fal" {
		return attemptDTO{}, &queueError{field: "shot", msg: "This profile does not make video; switch its generator to fal.ai video."}
	}
	if shot.Prompt == "" {
		return attemptDTO{}, &queueError{field: "prompt", msg: "Write the shot's prompt first."}
	}
	// A shot starts from a picture. That is the path whose length and price are
	// known; a text-only clip has neither a length setting nor a checked price.
	if shot.ImageUrl == "" && !shot.ImageID.Valid {
		return attemptDTO{}, &queueError{field: "imageId", msg: "Give the shot a starting picture first: make one, reuse one or upload one."}
	}
	if latest != nil && latest.Status == "pending" {
		return attemptDTO{}, &queueError{code: http.StatusConflict, msg: "This shot is already being made."}
	}

	row, err := h.q.CreateGeneration(ctx, sqlc.CreateGenerationParams{
		ProfileID: v.ProfileID,
		// The style's wording is added here, so the stored prompt is exactly what
		// fal receives (as for a clip made from a template).
		InputPrompt: studio.For(profile.VideoStyle).Compose(shot.Prompt, true),
		ImageUrl:    shot.ImageUrl,
		ImageID:     shot.ImageID,
		Status:      "pending",
		Provider:    profile.Provider,
		Seconds:     shot.Seconds,
		ShotID:      shot.ID,
	})
	if err != nil {
		return attemptDTO{}, &queueError{code: http.StatusInternalServerError, msg: "Failed to save the attempt."}
	}
	if err := h.enq.EnqueueGenerationRun(ctx, pgconv.UUIDString(row.ID)); err != nil {
		// Nothing will process the row: mark it failed so the screen says why.
		_ = h.q.FinishGeneration(ctx, sqlc.FinishGenerationParams{
			ID: row.ID, Status: "failed", Error: "could not queue the job", Provider: row.Provider,
		})
		return attemptDTO{}, &queueError{code: http.StatusInternalServerError, msg: "Could not queue the shot."}
	}
	return attemptDTO{
		ID:        pgconv.UUIDString(row.ID),
		Status:    row.Status,
		Seconds:   int(row.Seconds),
		CreatedAt: pgconv.TimeString(row.CreatedAt),
	}, nil
}

// setApproved marks a shot's plan as looked at and OK. Only a shot that could be
// generated can be approved (it has a prompt and a picture).
func (h *Handler) setApproved(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	var body struct {
		Approved bool `json:"approved"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Approved && (shot.Prompt == "" || (shot.ImageUrl == "" && !shot.ImageID.Valid)) {
		writeFieldError(w, "approved", "Give the shot a prompt and a starting picture before approving it.")
		return
	}
	updated, err := h.q.SetShotApproved(r.Context(), sqlc.SetShotApprovedParams{ID: shot.ID, Approved: body.Approved})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to approve the shot.")
		return
	}
	latest, _ := h.latestByShot(r.Context(), v.ID)
	writeJSON(w, http.StatusOK, map[string]any{"shot": toShotDTO(updated, latest[pgconv.UUIDString(updated.ID)])})
}

// generateShots makes an attempt at each of the listed shots in one go (the
// screen showed the total price and asked once). Only approved shots of this
// video are made; any other is reported as skipped with the reason, so a stale
// list on screen can never make something the owner did not look at.
func (h *Handler) generateShots(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	var body struct {
		ShotIDs []string `json:"shotIds"`
	}
	if !decode(w, r, &body) {
		return
	}
	if len(body.ShotIDs) == 0 || len(body.ShotIDs) > MaxShots {
		writeFieldError(w, "shotIds", "Choose between 1 and 12 shots.")
		return
	}

	profile, err := h.q.GetProfile(r.Context(), v.ProfileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}
	shots, err := h.q.ListShots(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}
	byID := make(map[string]sqlc.VideoShot, len(shots))
	for _, s := range shots {
		byID[pgconv.UUIDString(s.ID)] = s
	}
	latest, err := h.latestByShot(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check the shots.")
		return
	}

	type queued struct {
		ShotID string `json:"shotId"`
		attemptDTO
	}
	type skipped struct {
		ShotID string `json:"shotId"`
		Reason string `json:"reason"`
	}
	queuedOut, skippedOut := []queued{}, []skipped{}
	seen := map[string]bool{}
	for _, id := range body.ShotIDs {
		shot, found := byID[id]
		switch {
		case seen[id]:
			continue
		case !found:
			skippedOut = append(skippedOut, skipped{id, "Not a shot of this video."})
			continue
		case !shot.Approved:
			skippedOut = append(skippedOut, skipped{id, "Not approved."})
			continue
		}
		seen[id] = true
		attempt, qe := h.queueShot(r.Context(), v, profile, shot, latest[id])
		if qe != nil {
			skippedOut = append(skippedOut, skipped{id, qe.msg})
			continue
		}
		queuedOut = append(queuedOut, queued{id, attempt})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"queued": queuedOut, "skipped": skippedOut})
}

// --- helpers ----------------------------------------------------------------

// loadVideo reads {id} and loads the video, writing the error response itself.
func (h *Handler) loadVideo(w http.ResponseWriter, r *http.Request) (sqlc.Video, bool) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid video id.")
		return sqlc.Video{}, false
	}
	v, err := h.q.GetVideo(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Video not found.")
		return sqlc.Video{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the video.")
		return sqlc.Video{}, false
	}
	return v, true
}

// loadShot reads {shotId} and loads it, checking it belongs to the video in the
// path (so another video's shot id cannot be changed through this one).
func (h *Handler) loadShot(w http.ResponseWriter, r *http.Request, v sqlc.Video) (sqlc.VideoShot, bool) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "shotId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid shot id.")
		return sqlc.VideoShot{}, false
	}
	s, err := h.q.GetShot(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && s.VideoID != v.ID) {
		writeError(w, http.StatusNotFound, "Shot not found.")
		return sqlc.VideoShot{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shot.")
		return sqlc.VideoShot{}, false
	}
	return s, true
}

// latestByShot maps a shot id to its newest attempt.
func (h *Handler) latestByShot(ctx context.Context, videoID pgtype.UUID) (map[string]*sqlc.ListShotLatestGenerationsRow, error) {
	rows, err := h.q.ListShotLatestGenerations(ctx, videoID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*sqlc.ListShotLatestGenerationsRow, len(rows))
	for i := range rows {
		out[pgconv.UUIDString(rows[i].ShotID)] = &rows[i]
	}
	return out, nil
}

// resolveImage turns the input's imageId into a UUID that is known to exist.
// An empty imageId is a NULL (the shot has no stored picture). It writes the
// error response itself.
func (h *Handler) resolveImage(w http.ResponseWriter, r *http.Request, in shotInput) (pgtype.UUID, bool) {
	if in.ImageID == "" {
		return pgtype.UUID{}, true
	}
	id, err := pgconv.ParseUUID(in.ImageID)
	if err != nil {
		writeFieldError(w, "imageId", "Invalid image id.")
		return pgtype.UUID{}, false
	}
	ok, err := h.q.ImageExists(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check the picture.")
		return pgtype.UUID{}, false
	}
	if !ok {
		writeFieldError(w, "imageId", "That picture no longer exists. Choose it again.")
		return pgtype.UUID{}, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return false
	}
	return true
}

// decodeOptional is decode for a body that may be empty (adding a shot with all
// defaults).
func decodeOptional(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.ContentLength == 0 {
		return true
	}
	return decode(w, r, v)
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
