package videos

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/spendcap"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// voiceTimeout bounds speaking one shot (fal's queue wait is up to 3 minutes).
const voiceTimeout = 4 * time.Minute

// voiceDTO is the voice-over of a shot: how it was spoken, how long it is, and
// whether it still matches the narration. The audio itself is served by
// GET /videos/{id}/shots/{shotId}/voice/audio.
type voiceDTO struct {
	Voice      string `json:"voice"`      // the voice key it was spoken with
	Text       string `json:"text"`       // exactly what was spoken
	DurationMs int    `json:"durationMs"` // approximate (see shot_audio in schema.sql)
	SizeBytes  int    `json:"sizeBytes"`
	CreatedAt  string `json:"createdAt"`
	// Current: the narration has not changed since this audio was made.
	Current bool `json:"current"`
}

func toVoiceDTO(s sqlc.VideoShot, a *sqlc.ListShotAudioMetaRow) *voiceDTO {
	if a == nil {
		return nil
	}
	return &voiceDTO{
		Voice: a.Voice, Text: a.Text, DurationMs: int(a.DurationMs), SizeBytes: int(a.SizeBytes),
		CreatedAt: pgconv.TimeString(a.CreatedAt), Current: a.Text == s.NarrationTh,
	}
}

// audioByShot maps a shot id to its voice-over (without the bytes). A shot with
// none is absent.
func (h *Handler) audioByShot(ctx context.Context, videoID pgtype.UUID) (map[string]*sqlc.ListShotAudioMetaRow, error) {
	rows, err := h.q.ListShotAudioMeta(ctx, videoID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*sqlc.ListShotAudioMetaRow, len(rows))
	for i := range rows {
		out[pgconv.UUIDString(rows[i].ShotID)] = &rows[i]
	}
	return out, nil
}

// narrator reads the narrator voice chosen in Settings: a listed voice, or one
// the owner named (stored as "custom:<name>"). ok is false when none is chosen
// or the stored value no longer names a voice.
func (h *Handler) narrator(ctx context.Context) (studio.VoiceCandidate, bool, error) {
	key, err := h.q.GetSetting(ctx, "narratorVoice")
	if errors.Is(err, pgx.ErrNoRows) {
		return studio.VoiceCandidate{}, false, nil
	}
	if err != nil {
		return studio.VoiceCandidate{}, false, err
	}
	c, ok := studio.ResolveVoice(strings.TrimSpace(key))
	return c, ok, nil
}

// speaker picks the provider: the fake one in local dev without a key
// (AI_FAKE_FAL=1), else fal with the saved key.
func (h *Handler) speaker(ctx context.Context) (generate.Voice, error) {
	if h.fakeFal {
		return generate.FakeVoice{}, nil
	}
	key, err := h.q.GetSetting(ctx, "falApiKey")
	if errors.Is(err, pgx.ErrNoRows) {
		key, err = "", nil
	}
	if err != nil {
		return nil, err
	}
	return generate.NewFalVoice(strings.TrimSpace(key)), nil // reports a missing key itself
}

// makeVoice speaks the shot's narration with the narrator voice and keeps the
// audio, replacing any earlier one. It is paid for per character, so the daily
// limit is checked first, and the screen asks for confirmation (with the price)
// before calling. It answers with the updated shot.
func (h *Handler) makeVoice(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	text := strings.TrimSpace(shot.NarrationTh)
	if text == "" {
		writeFieldError(w, "narrationTh", "Write the narration for this shot first.")
		return
	}
	if n := utf8.RuneCountInString(text); n > studio.MaxVoiceSampleChars {
		writeFieldError(w, "narrationTh", "The narration must be "+strconv.Itoa(studio.MaxVoiceSampleChars)+" characters or fewer.")
		return
	}
	voice, ok, err := h.narrator(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read the settings.")
		return
	}
	if !ok {
		writeFieldError(w, "voice", "Choose the narrator voice first (Settings, Thai voice samples).")
		return
	}

	if over, err := spendcap.Check(r.Context(), h.q, voice.CostUSD(text), time.Now()); err != nil {
		log.Printf("spendcap: %v", err)
	} else if over != nil {
		writeError(w, http.StatusPaymentRequired, over.Error())
		return
	}

	sp, err := h.speaker(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read the settings.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), voiceTimeout)
	defer cancel()
	res, err := sp.Speak(ctx, generate.VoiceRequest{
		Model: voice.Model, Family: string(voice.Family), VoiceID: voice.VoiceID, Text: text,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "The voice could not be made: "+err.Error())
		return
	}
	data, contentType, err := generate.FetchAudio(ctx, res.AudioURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The voice was made but its audio could not be fetched: "+err.Error())
		return
	}
	durationMs := res.DurationMs
	if durationMs == 0 {
		durationMs = generate.ApproxMP3DurationMs(len(data))
	}
	if err := h.q.UpsertShotAudio(r.Context(), sqlc.UpsertShotAudioParams{
		ShotID: shot.ID, Voice: voice.Key, Text: text, ContentType: contentType,
		SizeBytes: int32(len(data)), DurationMs: int32(durationMs), Data: data,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "The voice was made but not saved.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shot": h.shotDTOFor(r.Context(), v.ID, shot)})
}

// voiceAudio serves a shot's voice-over (to play in the page or to download).
func (h *Handler) voiceAudio(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	a, err := h.q.GetShotAudioData(r.Context(), shot.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "This shot has no voice-over yet.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the audio.")
		return
	}
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(a.Data)))
	// The same URL serves a new audio after it is spoken again; the page adds a
	// version to the link, so the browser may keep each version.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(a.Data)
}

// deleteVoice removes a shot's voice-over (the narration text stays).
func (h *Handler) deleteVoice(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	if _, err := h.q.DeleteShotAudio(r.Context(), shot.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove the voice-over.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
