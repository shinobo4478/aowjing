// Package voice serves the Thai voice samples: the owner types (or keeps) a Thai
// text, picks candidate voices, and every candidate speaks the same text so the
// owner can choose by ear. Nothing is saved: the audio stays on fal (its links
// last at least 7 days), which is all a listening test needs. Choosing a voice
// and speaking real narration come later (CLAUDE.md, Thai voice-over plan).
package voice

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/spendcap"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// sampleTimeout bounds one candidate (fal's queue wait is up to 3 minutes).
const sampleTimeout = 4 * time.Minute

type Handler struct {
	q       *sqlc.Queries
	fakeFal bool
}

func NewHandler(q *sqlc.Queries, fakeFal bool) *Handler { return &Handler{q: q, fakeFal: fakeFal} }

// Routes returns the router to mount at /voice.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/candidates", h.candidates)
	r.Post("/samples", h.samples)
	return r
}

type candidateDTO struct {
	Key             string  `json:"key"`
	Label           string  `json:"label"`
	PricePer1KChars float64 `json:"pricePer1kChars"`
	Note            string  `json:"note"`
}

// candidates lists the voices and the limits, so the screen never repeats them
// (and shows the price before anything is spent).
func (h *Handler) candidates(w http.ResponseWriter, _ *http.Request) {
	cs := studio.VoiceCandidates()
	out := make([]candidateDTO, len(cs))
	for i, c := range cs {
		out[i] = candidateDTO{Key: c.Key, Label: c.Label, PricePer1KChars: c.PricePer1KChars, Note: c.Note}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidates": out,
		"minChars":   studio.MinVoiceSampleChars,
		"maxChars":   studio.MaxVoiceSampleChars,
		"maxVoices":  studio.MaxVoiceCandidates,
	})
}

type sampleResult struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	OK         bool    `json:"ok"`
	AudioURL   string  `json:"audioUrl,omitempty"`
	DurationMs int     `json:"durationMs"`
	CostUSD    float64 `json:"costUsd"` // fal's list price for this text; 0 when it failed
	Error      string  `json:"error,omitempty"`
}

// samples makes every chosen candidate speak the same text. A candidate that
// fails (ElevenLabs may refuse Thai) comes back with its own error and does not
// fail the others. It is paid for per character, so the daily limit is checked
// first, and the screen asks for confirmation before calling.
func (h *Handler) samples(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text       string   `json:"text"`
		Candidates []string `json:"candidates"`
		// CustomVoices are voice names or ElevenLabs voice ids the owner typed in (see
		// studio.CustomVoice), spoken in ElevenLabs v3 like the listed ones.
		CustomVoices []string `json:"customVoices"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	text, cands, errs := validate(body.Text, body.Candidates, body.CustomVoices)
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}

	var total float64
	for _, c := range cands {
		total += c.CostUSD(text)
	}
	if over, err := spendcap.Check(r.Context(), h.q, total, time.Now()); err != nil {
		log.Printf("spendcap: %v", err)
	} else if over != nil {
		writeError(w, http.StatusPaymentRequired, over.Error())
		return
	}

	speaker, err := h.speaker(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read the settings.")
		return
	}
	results := runSamples(r.Context(), speaker, cands, text)
	var spent float64
	for _, res := range results {
		spent += res.CostUSD
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"results":      results,
		"chars":        utf8.RuneCountInString(text),
		"totalCostUsd": spent,
	})
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

// validate trims the text and resolves the candidate keys, returning a message
// per field (empty = valid).
func validate(rawText string, keys, custom []string) (string, []studio.VoiceCandidate, map[string]string) {
	errs := map[string]string{}
	text := strings.TrimSpace(rawText)
	switch n := utf8.RuneCountInString(text); {
	case n < studio.MinVoiceSampleChars:
		errs["text"] = "Write the Thai text to speak (at least " + strconv.Itoa(studio.MinVoiceSampleChars) + " characters)."
	case n > studio.MaxVoiceSampleChars:
		errs["text"] = "The text must be " + strconv.Itoa(studio.MaxVoiceSampleChars) + " characters or fewer."
	}
	var cands []studio.VoiceCandidate
	seen := map[string]bool{}
	for _, k := range keys {
		c, ok := studio.VoiceCandidateFor(k)
		if !ok {
			errs["candidates"] = "Unknown voice."
			break
		}
		if !seen[k] {
			seen[k] = true
			cands = append(cands, c)
		}
	}
	// The voices the owner typed in. A bad name is refused with its own message;
	// a name already in the list (or typed twice) is counted once.
	for _, name := range custom {
		c, ok := studio.CustomVoice(name)
		if !ok {
			errs["customVoices"] = "A voice name or id may have only letters, digits, spaces, \"_\" and \"-\" (up to " + strconv.Itoa(studio.MaxCustomVoiceChars) + " characters)."
			break
		}
		if !seen[c.Key] {
			seen[c.Key] = true
			cands = append(cands, c)
		}
	}
	if _, bad := errs["candidates"]; !bad && errs["customVoices"] == "" {
		switch {
		case len(cands) == 0:
			errs["candidates"] = "Choose at least one voice."
		case len(cands) > studio.MaxVoiceCandidates:
			errs["candidates"] = "Choose at most " + strconv.Itoa(studio.MaxVoiceCandidates) + " voices."
		}
	}
	return text, cands, errs
}

// runSamples speaks the text with every candidate at once and returns the
// results in the candidates' order. It is separate from the handler so it can be
// tested with a scripted voice and no database.
func runSamples(ctx context.Context, v generate.Voice, cands []studio.VoiceCandidate, text string) []sampleResult {
	results := make([]sampleResult, len(cands))
	var wg sync.WaitGroup
	for i, c := range cands {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, sampleTimeout)
			defer cancel()
			res := sampleResult{Key: c.Key, Label: c.Label}
			out, err := v.Speak(cctx, generate.VoiceRequest{
				Model: c.Model, Family: string(c.Family), VoiceID: c.VoiceID, Text: text,
			})
			if err != nil {
				res.Error = err.Error()
			} else {
				res.OK, res.AudioURL, res.DurationMs, res.CostUSD = true, out.AudioURL, out.DurationMs, c.CostUSD(text)
			}
			results[i] = res // each goroutine writes its own index
		}()
	}
	wg.Wait()
	return results
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
