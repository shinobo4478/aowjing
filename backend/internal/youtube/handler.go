// Package youtube connects a channel to a YouTube account over OAuth. Mounted
// at /youtube. Publishing a generation to the connected account is a later
// commit.
package youtube

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

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
	r.Get("/uploads", h.listUploads)
	r.Post("/publish", h.publish)
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

// listUploads returns every YouTube publish under a profile, newest first, so
// the generations view can show a "▶ on YouTube" link.
func (h *Handler) listUploads(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId query parameter is required.")
		return
	}
	rows, err := h.q.ListYouTubeUploadsByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list uploads.")
		return
	}
	out := make([]uploadDTO, len(rows))
	for i, u := range rows {
		out[i] = toUploadDTO(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{"uploads": out})
}

// publish downloads a finished video generation and uploads it to the YouTube
// account connected to the given channel.
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.YouTubeConfigured() {
		writeError(w, http.StatusServiceUnavailable, "YouTube is not configured on the server.")
		return
	}

	var body struct {
		GenerationID string `json:"generationId"`
		ChannelID    string `json:"channelId"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Privacy      string `json:"privacy"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	switch {
	case body.Title == "" || len(body.Title) > 100:
		writeError(w, http.StatusUnprocessableEntity, "Title is required (max 100 characters).")
		return
	case len(body.Description) > 5000:
		writeError(w, http.StatusUnprocessableEntity, "Description must be 5000 characters or fewer.")
		return
	}
	switch body.Privacy {
	case "private", "unlisted", "public":
	default:
		writeError(w, http.StatusUnprocessableEntity, `Privacy must be "private", "unlisted" or "public".`)
		return
	}

	genID, err := pgconv.ParseUUID(body.GenerationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid generationId is required.")
		return
	}
	channelID, err := pgconv.ParseUUID(body.ChannelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid channelId is required.")
		return
	}

	gen, err := h.q.GetGeneration(r.Context(), genID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Generation not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load generation.")
		return
	}
	if gen.Status != "succeeded" || gen.OutputKind != "video" || gen.Output == "" {
		writeError(w, http.StatusUnprocessableEntity, "That generation has no finished video to publish.")
		return
	}

	acct, err := h.q.GetYouTubeAccount(r.Context(), channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "That channel is not connected to YouTube.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the YouTube connection.")
		return
	}

	tok, err := freshToken(r.Context(), h.cfg, acct)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Could not refresh the YouTube token — reconnect the channel.")
		return
	}
	if tok.AccessToken != acct.AccessToken {
		_ = h.q.UpdateYouTubeTokens(r.Context(), sqlc.UpdateYouTubeTokensParams{
			ChannelID:   channelID,
			AccessToken: tok.AccessToken,
			TokenExpiry: pgtype.Timestamptz{Time: tok.Expiry, Valid: true},
		})
	}

	video, err := downloadVideo(r.Context(), gen.Output)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Could not download the generated video.")
		return
	}

	videoID, err := uploadVideo(r.Context(), oauthConfig(h.cfg).Client(r.Context(), tok), videoMeta{
		Title:       body.Title,
		Description: body.Description,
		Privacy:     body.Privacy,
	}, video)
	if err != nil {
		writeError(w, http.StatusBadGateway, "YouTube rejected the upload: "+err.Error())
		return
	}

	up, err := h.q.CreateYouTubeUpload(r.Context(), sqlc.CreateYouTubeUploadParams{
		GenerationID:   genID,
		ChannelID:      channelID,
		YoutubeVideoID: videoID,
		Privacy:        body.Privacy,
		Title:          body.Title,
	})
	if err != nil {
		// The upload landed; we just couldn't record it. Still a success.
		writeJSON(w, http.StatusOK, map[string]any{
			"videoId": videoID,
			"url":     watchURL(videoID),
		})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"upload": toUploadDTO(up)})
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

func watchURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}

type uploadDTO struct {
	ID             string `json:"id"`
	GenerationID   string `json:"generationId"`
	ChannelID      string `json:"channelId"`
	YoutubeVideoID string `json:"youtubeVideoId"`
	URL            string `json:"url"`
	Privacy        string `json:"privacy"`
	Title          string `json:"title"`
	CreatedAt      string `json:"createdAt"`
}

func toUploadDTO(u sqlc.YoutubeUpload) uploadDTO {
	return uploadDTO{
		ID:             pgconv.UUIDString(u.ID),
		GenerationID:   pgconv.UUIDString(u.GenerationID),
		ChannelID:      pgconv.UUIDString(u.ChannelID),
		YoutubeVideoID: u.YoutubeVideoID,
		URL:            watchURL(u.YoutubeVideoID),
		Privacy:        u.Privacy,
		Title:          u.Title,
		CreatedAt:      pgconv.TimeString(u.CreatedAt),
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
