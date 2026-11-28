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
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/scriptwriter"
	"github.com/shinobo4478/aowjing/backend/internal/spendcap"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// Enqueuer hands a generation to the worker. *queue.Client satisfies it.
type Enqueuer interface {
	EnqueueGenerationRun(ctx context.Context, generationID string) error
}

type Handler struct {
	db      *pgxpool.Pool
	q       *sqlc.Queries
	enq     Enqueuer
	drafter Drafter
	// fakeFal makes the voice-over with a silent stand-in (AI_FAKE_FAL=1).
	fakeFal bool
}

func NewHandler(db *pgxpool.Pool, q *sqlc.Queries, enq Enqueuer, drafter Drafter, fakeFal bool) *Handler {
	return &Handler{db: db, q: q, enq: enq, drafter: drafter, fakeFal: fakeFal}
}

// Routes returns the router to mount at /videos.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Post("/draft", h.draft)
	r.Post("/translate", h.translate)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.rename)
	r.Delete("/{id}", h.delete)

	r.Post("/{id}/shots", h.addShot)
	r.Put("/{id}/shots/order", h.reorder)
	r.Patch("/{id}/shots/{shotId}", h.updateShot)
	r.Delete("/{id}/shots/{shotId}", h.deleteShot)
	r.Post("/{id}/shots/{shotId}/generate", h.generateShot)
	r.Post("/{id}/shots/{shotId}/revise", h.reviseShot)
	// The voice-over of a shot: speak it, play it, remove it.
	r.Post("/{id}/shots/{shotId}/voice", h.makeVoice)
	r.Get("/{id}/shots/{shotId}/voice/audio", h.voiceAudio)
	r.Delete("/{id}/shots/{shotId}/voice", h.deleteVoice)
	r.Put("/{id}/shots/{shotId}/approved", h.setApproved)
	r.Get("/{id}/shots/{shotId}/attempts", h.listAttempts)
	r.Put("/{id}/shots/{shotId}/take", h.setTake)
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
	firstSeconds := studio.ModelFor(profile.VideoModel).DefaultSeconds
	if _, err := h.q.CreateShot(r.Context(), sqlc.CreateShotParams{
		VideoID: v.ID, Seconds: int16(firstSeconds),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "The video was created but its first shot was not.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"video": toVideoDTO(v, 1, firstSeconds)})
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
	takes, err := h.takeByShot(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}

	audio, err := h.audioByShot(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}

	out := make([]shotDTO, len(shots))
	total := 0
	for i, s := range shots {
		id := pgconv.UUIDString(s.ID)
		out[i] = toShotDTO(s, latest[id], takes[id], audio[id])
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
	if in.Seconds == 0 {
		// No length given: the model's own default (Veo's is not 5).
		if p, err := h.q.GetProfile(r.Context(), v.ProfileID); err == nil {
			in.Seconds = studio.ModelFor(p.VideoModel).DefaultSeconds
		}
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
	imageID, endID, ok := h.resolveImages(w, r, in)
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
	if in.ImageURL != "" || imageID.Valid || endID.Valid || in.PicturePrompt != "" || in.EndPicturePrompt != "" ||
		in.PromptTh != "" || in.PicturePromptTh != "" || in.EndPicturePromptTh != "" || in.NarrationTh != "" {
		shot, err = h.q.UpdateShot(r.Context(), sqlc.UpdateShotParams{
			ID: shot.ID, Prompt: shot.Prompt, Seconds: shot.Seconds,
			ImageUrl: in.ImageURL, ImageID: imageID, EndImageID: endID,
			PicturePrompt: in.PicturePrompt, EndPicturePrompt: in.EndPicturePrompt,
			PromptTh: in.PromptTh, PicturePromptTh: in.PicturePromptTh, EndPicturePromptTh: in.EndPicturePromptTh,
			NarrationTh: in.NarrationTh,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "The shot was added but its picture was not.")
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"shot": toShotDTO(shot, nil, nil, nil)})
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
	imageID, endID, ok := h.resolveImages(w, r, in)
	if !ok {
		return
	}

	updated, err := h.q.UpdateShot(r.Context(), sqlc.UpdateShotParams{
		ID: shot.ID, Prompt: in.Prompt, Seconds: int16(in.Seconds),
		ImageUrl: in.ImageURL, ImageID: imageID, EndImageID: endID,
		PicturePrompt: in.PicturePrompt, EndPicturePrompt: in.EndPicturePrompt,
			PromptTh: in.PromptTh, PicturePromptTh: in.PicturePromptTh, EndPicturePromptTh: in.EndPicturePromptTh,
			NarrationTh: in.NarrationTh,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update the shot.")
		return
	}
	// If the shot swapped to another picture (start or end), drop the old one
	// when nothing else uses it.
	if shot.ImageID.Valid && shot.ImageID != imageID {
		_ = h.q.DeleteImageIfUnused(r.Context(), shot.ImageID)
	}
	if shot.EndImageID.Valid && shot.EndImageID != endID {
		_ = h.q.DeleteImageIfUnused(r.Context(), shot.EndImageID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"shot": h.shotDTOFor(r.Context(), v.ID, updated)})
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
	// The video model decides which lengths exist (Veo: 4, 6 or 8) and whether an
	// end picture can be used. Checked here, before anything is paid for, so a
	// shot kept from before the model was switched says what to change.
	if m := studio.ModelFor(profile.VideoModel); !m.ValidSeconds(int(shot.Seconds)) {
		return attemptDTO{}, &queueError{field: "seconds", msg: m.Label + " makes shots of " + m.SecondsList() + " seconds; this shot is " + strconv.Itoa(int(shot.Seconds)) + " s. Change its length."}
	} else if shot.EndImageID.Valid && !m.EndPicture {
		return attemptDTO{}, &queueError{field: "endImageId", msg: m.Label + " cannot use an end picture. Remove the shot's end picture, or switch the profile's video model."}
	}
	if latest != nil && latest.Status == "pending" {
		return attemptDTO{}, &queueError{code: http.StatusConflict, msg: "This shot is already being made."}
	}
	// The daily spending limit (Settings). In "generate all" the shots queued
	// before this one already count (they are pending rows), so the total is held.
	m := studio.ModelFor(profile.VideoModel)
	if over, err := spendcap.Check(ctx, h.q, spendcap.ClipUSD(m, int(shot.Seconds), profile.VideoAudio), time.Now()); err != nil {
		log.Printf("spendcap: %v", err)
	} else if over != nil {
		return attemptDTO{}, &queueError{code: http.StatusPaymentRequired, msg: over.Error()}
	}

	row, err := h.q.CreateGeneration(ctx, sqlc.CreateGenerationParams{
		ProfileID: v.ProfileID,
		// The style's wording is added here, so the stored prompt is exactly what
		// fal receives (as for a clip made from a template).
		InputPrompt: studio.For(profile.VideoStyle).Compose(shot.Prompt, true),
		ImageUrl:    shot.ImageUrl,
		ImageID:     shot.ImageID,
		EndImageID:  shot.EndImageID,
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
	writeJSON(w, http.StatusOK, map[string]any{"shot": h.shotDTOFor(r.Context(), v.ID, updated)})
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

// takeByShot maps a shot id to the take to use: the attempt the owner chose if
// it succeeded, else the newest successful one. A shot with none is absent.
func (h *Handler) takeByShot(ctx context.Context, videoID pgtype.UUID) (map[string]*sqlc.ListShotTakesRow, error) {
	rows, err := h.q.ListShotTakes(ctx, videoID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*sqlc.ListShotTakesRow, len(rows))
	for i := range rows {
		out[pgconv.UUIDString(rows[i].ShotID)] = &rows[i]
	}
	return out, nil
}

// shotDTOFor builds one shot's DTO with its newest attempt and its take. A
// failure to read them leaves those out rather than failing the whole answer.
func (h *Handler) shotDTOFor(ctx context.Context, videoID pgtype.UUID, s sqlc.VideoShot) shotDTO {
	latest, _ := h.latestByShot(ctx, videoID)
	takes, _ := h.takeByShot(ctx, videoID)
	audio, _ := h.audioByShot(ctx, videoID)
	id := pgconv.UUIDString(s.ID)
	return toShotDTO(s, latest[id], takes[id], audio[id])
}

// listAttempts returns every attempt at a shot, newest first, so the owner can
// compare the takes and pick one.
func (h *Handler) listAttempts(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	rows, err := h.q.ListShotAttempts(r.Context(), shot.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the attempts.")
		return
	}
	out := make([]attemptDTO, len(rows))
	for i, row := range rows {
		out[i] = attemptDTO{
			ID:        pgconv.UUIDString(row.ID),
			Status:    row.Status,
			Output:    row.Output,
			Error:     row.Error,
			Seconds:   int(row.Seconds),
			CreatedAt: pgconv.TimeString(row.CreatedAt),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"attempts": out,
		"chosenId": pgconv.UUIDString(shot.ChosenGenerationID),
	})
}

// setTake picks which attempt is THE clip of a shot (an empty generationId goes
// back to "the newest successful one"). Only a successful attempt of this shot
// can be chosen.
func (h *Handler) setTake(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	var body struct {
		GenerationID string `json:"generationId"`
	}
	if !decode(w, r, &body) {
		return
	}

	var chosen pgtype.UUID // NULL unless one is given
	if body.GenerationID != "" {
		id, err := pgconv.ParseUUID(body.GenerationID)
		if err != nil {
			writeFieldError(w, "generationId", "Invalid attempt id.")
			return
		}
		g, err := h.q.GetGeneration(r.Context(), id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && g.ShotID != shot.ID) {
			writeFieldError(w, "generationId", "That is not an attempt at this shot.")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to check the attempt.")
			return
		}
		if g.Status != "succeeded" {
			writeFieldError(w, "generationId", "Only a finished attempt can be chosen.")
			return
		}
		chosen = id
	}
	updated, err := h.q.SetShotChosen(r.Context(), sqlc.SetShotChosenParams{ID: shot.ID, GenerationID: chosen})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to choose the take.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shot": h.shotDTOFor(r.Context(), v.ID, updated)})
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

// resolveImages resolves the start and end pictures of the input to UUIDs known to exist.
// An empty imageId is a NULL (the shot has no stored picture). It writes the
// error response itself.
func (h *Handler) resolveImages(w http.ResponseWriter, r *http.Request, in shotInput) (start, end pgtype.UUID, ok bool) {
	start, ok = h.resolvePicture(w, r, "imageId", in.ImageID)
	if !ok {
		return start, end, false
	}
	end, ok = h.resolvePicture(w, r, "endImageId", in.EndImageID)
	return start, end, ok
}

// resolvePicture turns one picture id from the input into a UUID that is known
// to exist; "" is a NULL. `field` names it in the error message.
func (h *Handler) resolvePicture(w http.ResponseWriter, r *http.Request, field, raw string) (pgtype.UUID, bool) {
	if raw == "" {
		return pgtype.UUID{}, true
	}
	id, err := pgconv.ParseUUID(raw)
	if err != nil {
		writeFieldError(w, field, "Invalid image id.")
		return pgtype.UUID{}, false
	}
	ok, err := h.q.ImageExists(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check the picture.")
		return pgtype.UUID{}, false
	}
	if !ok {
		writeFieldError(w, field, "That picture no longer exists. Choose it again.")
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

// --- drafting a video with the script helper ----------------------------------

// Drafter writes a shot list for a brief and translates a prompt between Thai
// and English (internal/scriptwriter): both go through the same LLM route.
type Drafter interface {
	Draft(ctx context.Context, b scriptwriter.Brief) (scriptwriter.Draft, error)
	Translate(ctx context.Context, r scriptwriter.TranslateRequest) (scriptwriter.Translation, error)
	Revise(ctx context.Context, r scriptwriter.ReviseRequest) (scriptwriter.Revision, error)
}

// translateTimeout covers one translation (a short text; fal's queue wait is
// still up to 3 minutes in the worst case).
const translateTimeout = 4 * time.Minute

// translate turns one prompt from Thai into English (what the models receive) or
// from English into Thai (so the owner can read what a prompt says). It is
// stateless: the screen decides where the result goes. The reply carries what the
// call cost, as fal reported it, so the screen can show it.
func (h *Handler) translate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind      string `json:"kind"`
		Direction string `json:"direction"`
		Text      string `json:"text"`
	}
	if !decode(w, r, &body) {
		return
	}
	req := scriptwriter.TranslateRequest{
		Kind:      scriptwriter.Kind(body.Kind),
		Direction: scriptwriter.Direction(body.Direction),
		Text:      body.Text,
	}
	if errs := req.Validate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), translateTimeout)
	defer cancel()
	t, err := h.drafter.Translate(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The translation failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": t.Text, "costUsd": t.CostUSD, "model": t.Model})
}

// draftTimeout covers the model's reply (fal's queue wait is up to 3 minutes).
const draftTimeout = 4 * time.Minute

// draft asks the script helper for a shot list and creates a Video from it:
// the scene and each shot's picture prompt, motion prompt and length are filled
// in for the owner to read and change. It makes no pictures and no clips; the
// only cost is the model's reply, which is returned so the screen can show it.
func (h *Handler) draft(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
		Topic     string `json:"topic"`
		Shots     int    `json:"shots"`
		Seconds   int    `json:"seconds"`
		Title     string `json:"title"`
		// Mode is general (default), product or film; ProductID is the catalog
		// product a product video is about (required for that mode).
		Mode      string `json:"mode"`
		ProductID string `json:"productId"`
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
	model := studio.ModelFor(profile.VideoModel)
	if !scriptwriter.ValidMode(body.Mode) {
		writeFieldError(w, "mode", "Choose general, product, film or explainer.")
		return
	}
	if body.Seconds == 0 {
		body.Seconds = model.DefaultSeconds
		if scriptwriter.Mode(body.Mode) == scriptwriter.ModeExplainer {
			body.Seconds = studio.ExplainerSeconds(model.Seconds)
		}
	}
	brief := scriptwriter.Brief{
		Topic: body.Topic, Shots: body.Shots, Seconds: body.Seconds,
		ProfileName: profile.Name, Niche: profile.Niche, Description: profile.Description,
		Mode: scriptwriter.Mode(body.Mode), VideoStyle: profile.VideoStyle,
		SecondsOptions: model.Seconds,
	}
	if brief.Cast, err = h.castNotes(r.Context(), profileID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the cast.")
		return
	}
	if brief.Mode == scriptwriter.ModeProduct {
		product, msg, err := h.productNote(r.Context(), body.ProductID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load the product.")
			return
		}
		if msg != "" {
			writeFieldError(w, "productId", msg)
			return
		}
		brief.Product = product
	}
	errs := brief.Validate()
	if _, bad := errs["seconds"]; !bad && !model.ValidSeconds(body.Seconds) {
		errs["seconds"] = model.Label + " makes shots of " + model.SecondsList() + " seconds."
	}
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	var title string
	if strings.TrimSpace(body.Title) != "" {
		if title, err = validateTitle(body.Title); err != nil {
			writeFieldError(w, "title", err.Error())
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), draftTimeout)
	defer cancel()
	d, err := h.drafter.Draft(ctx, brief)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The draft could not be written: "+err.Error())
		return
	}
	if title == "" {
		title = d.Title
	}
	if title == "" {
		title = brief.Topic
	}
	title = truncateRunes(title, maxTitleRunes)

	v, err := h.q.CreateVideo(r.Context(), sqlc.CreateVideoParams{ProfileID: profileID, Title: title})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "The draft was written but the video was not saved.")
		return
	}
	for _, s := range d.Shots {
		if _, err := h.q.CreateShotWithPlan(r.Context(), sqlc.CreateShotWithPlanParams{
			VideoID: v.ID, Prompt: s.MotionPrompt, Seconds: int16(s.Seconds), PicturePrompt: s.PicturePrompt, EndPicturePrompt: s.EndPicturePrompt,
			PromptTh: s.MotionPromptTh, PicturePromptTh: s.PicturePromptTh, EndPicturePromptTh: s.EndPicturePromptTh,
			NarrationTh: s.NarrationTh,
		}); err != nil {
			_, _ = h.q.DeleteVideo(r.Context(), v.ID) // no half-filled video left behind
			writeError(w, http.StatusInternalServerError, "The draft was written but its shots were not saved.")
			return
		}
	}
	// A narrated shot's length comes from its narration, so add the real ones up.
	totalSeconds, tooLong := 0, 0
	for _, s := range d.Shots {
		totalSeconds += s.Seconds
		if s.NarrationTooLong {
			tooLong++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"video":   toVideoDTO(v, len(d.Shots), totalSeconds),
		"costUsd": d.CostUSD,
		"model":   d.Model,
		// How many shots have a narration too long for the longest allowed shot.
		"tooLongShots": tooLong,
	})
}

// castNotes reads the profile's cast for the script helper (names and notes
// only; the pictures are used later, when a shot's picture is made).
func (h *Handler) castNotes(ctx context.Context, profileID pgtype.UUID) ([]scriptwriter.CastNote, error) {
	members, err := h.q.ListCastMembers(ctx, profileID)
	if err != nil {
		return nil, err
	}
	out := make([]scriptwriter.CastNote, len(members))
	for i, m := range members {
		out[i] = scriptwriter.CastNote{Role: m.Role, Name: m.Name, Note: m.Note}
	}
	return out, nil
}

// productNote reads the catalog product a product video is about, with the
// states its pictures show. A non-empty message is a validation problem for the
// "productId" field (err is only for real failures).
func (h *Handler) productNote(ctx context.Context, rawID string) (*scriptwriter.ProductNote, string, error) {
	id, perr := pgconv.ParseUUID(rawID)
	if perr != nil {
		return nil, "Choose the product this video is about.", nil
	}
	p, err := h.q.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "That product no longer exists.", nil
	}
	if err != nil {
		return nil, "", err
	}
	pics, err := h.q.ListProductPictures(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if len(pics) == 0 {
		return nil, "Add pictures of the product first (Products page), so its real look can be kept.", nil
	}
	states := make([]string, len(pics))
	for i, pic := range pics {
		states[i] = pic.Label
	}
	return &scriptwriter.ProductNote{Name: p.Name, Note: p.Note, States: states}, "", nil
}

// truncateRunes cuts s to at most n characters (not bytes).
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
