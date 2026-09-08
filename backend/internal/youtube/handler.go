// Package youtube connects a channel to a YouTube account over OAuth. Mounted
// at /youtube. Publishing a generation to the connected account is a later
// commit.
package youtube

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/config"
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

type Handler struct {
	q   *sqlc.Queries
	cfg config.Config
}

func NewHandler(q *sqlc.Queries, cfg config.Config) *Handler {
	return &Handler{q: q, cfg: cfg}
}

// Routes returns the router to mount at /youtube.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/status", h.status)
	r.Get("/oauth/start", h.start)
	r.Get("/oauth/callback", h.callback)
	r.Delete("/accounts/{channelId}", h.disconnect)
	return r
}

// status tells the frontend whether the server has a YouTube client at all
// and, if a channelId is given, whether that channel is connected.
func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.YouTubeConfigured() {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "connected": false})
		return
	}
	channelID, err := pgconv.ParseUUID(r.URL.Query().Get("channelId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid channelId query parameter is required.")
		return
	}

	acct, err := h.q.GetYouTubeAccount(r.Context(), channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"configured": true, "connected": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load YouTube status.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"connected":  true,
		"account":    toAccountDTO(acct),
	})
}

// start redirects the browser to Google's consent screen. The target channel
// is carried in a signed state parameter.
func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.YouTubeConfigured() {
		writeError(w, http.StatusServiceUnavailable, "YouTube is not configured on the server.")
		return
	}
	channelID := r.URL.Query().Get("channelId")
	if _, err := pgconv.ParseUUID(channelID); err != nil {
		writeError(w, http.StatusBadRequest, "A valid channelId query parameter is required.")
		return
	}
	state := signState(h.cfg.AdminPassword, channelID)
	http.Redirect(w, r, authCodeURL(h.cfg, state), http.StatusFound)
}

// callback is where Google sends the browser back with an auth code. It's a
// top-level navigation, so failures redirect to the app with ?yt=error rather
// than returning JSON.
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	redirectApp := func(path string) {
		http.Redirect(w, r, h.cfg.CORSOrigin+path, http.StatusFound)
	}
	fail := func(reason string) {
		redirectApp("/profiles?yt=error&reason=" + url.QueryEscape(reason))
	}

	if !h.cfg.YouTubeConfigured() {
		writeError(w, http.StatusServiceUnavailable, "YouTube is not configured on the server.")
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		fail(e) // user denied consent, etc.
		return
	}

	channelIDStr, ok := verifyState(h.cfg.AdminPassword, r.URL.Query().Get("state"))
	if !ok {
		fail("bad-state")
		return
	}
	channelID, err := pgconv.ParseUUID(channelIDStr)
	if err != nil {
		fail("bad-channel")
		return
	}
	ch, err := h.q.GetChannel(r.Context(), channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail("channel-not-found")
		return
	}
	if err != nil {
		fail("lookup")
		return
	}

	tok, err := oauthConfig(h.cfg).Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		fail("token-exchange")
		return
	}

	yc, err := fetchChannel(r.Context(), oauthConfig(h.cfg).Client(r.Context(), tok))
	if err != nil {
		fail("channel-info")
		return
	}

	if _, err := h.q.UpsertYouTubeAccount(r.Context(), sqlc.UpsertYouTubeAccountParams{
		ChannelID:        channelID,
		YoutubeChannelID: yc.ID,
		YoutubeTitle:     yc.Title,
		AccessToken:      tok.AccessToken,
		RefreshToken:     tok.RefreshToken,
		TokenExpiry:      pgtype.Timestamptz{Time: tok.Expiry, Valid: true},
		Scope:            r.URL.Query().Get("scope"),
	}); err != nil {
		fail("save")
		return
	}

	redirectApp("/profiles/" + pgconv.UUIDString(ch.ProfileID) + "?yt=connected")
}

// disconnect drops the stored tokens for a channel. The grant still exists on
// the user's Google account until they remove it there.
func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	channelID, err := pgconv.ParseUUID(chi.URLParam(r, "channelId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid channel id.")
		return
	}
	n, err := h.q.DeleteYouTubeAccount(r.Context(), channelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to disconnect.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "No YouTube connection for that channel.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- DTO + helpers ---------------------------------------------------

type accountDTO struct {
	ChannelID        string `json:"channelId"`
	YoutubeChannelID string `json:"youtubeChannelId"`
	YoutubeTitle     string `json:"youtubeTitle"`
	ConnectedAt      string `json:"connectedAt"`
	UpdatedAt        string `json:"updatedAt"`
}

func toAccountDTO(a sqlc.YoutubeAccount) accountDTO {
	return accountDTO{
		ChannelID:        pgconv.UUIDString(a.ChannelID),
		YoutubeChannelID: a.YoutubeChannelID,
		YoutubeTitle:     a.YoutubeTitle,
		ConnectedAt:      pgconv.TimeString(a.ConnectedAt),
		UpdatedAt:        pgconv.TimeString(a.UpdatedAt),
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
