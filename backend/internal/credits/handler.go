// Package credits shows how much prepaid fal.ai credit is left, for the menu
// bar. It uses its own optional key (setting falBillingKey, on fal's BILLING
// preset) so the key that makes clips does not need billing access.
package credits

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

const (
	// cacheFor keeps fal from being asked on every page load. fal documents no
	// rate limit for this endpoint, so stay gentle.
	cacheFor = 30 * time.Second
	// refreshAfter is how soon ?refresh=1 may ask fal again: the screen
	// refreshes right after spending, and a double-click must not hit fal twice.
	refreshAfter = 5 * time.Second
)

// response is the JSON the screen gets. It is always a 200: "no key yet" and
// "fal said no" are states to show quietly in the menu bar, not request errors.
type response struct {
	Configured bool     `json:"configured"`
	Balance    *float64 `json:"balance,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type cached struct {
	key  string // the billing key it was fetched with; a new key invalidates it
	at   time.Time
	resp response
}

type Handler struct {
	q     *sqlc.Queries
	fetch func(ctx context.Context, key string) (generate.FalCredits, error)
	now   func() time.Time

	mu    sync.Mutex
	cache *cached
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{
		q: q,
		fetch: func(ctx context.Context, key string) (generate.FalCredits, error) {
			return generate.NewFalAccount(key).Credits(ctx)
		},
		now: time.Now,
	}
}

// Routes returns the router to mount at /credits.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/fal", h.fal)
	return r
}

func (h *Handler) fal(w http.ResponseWriter, r *http.Request) {
	key, err := h.q.GetSetting(r.Context(), "falBillingKey")
	if errors.Is(err, pgx.ErrNoRows) {
		key = ""
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read settings."})
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		writeJSON(w, http.StatusOK, response{Configured: false})
		return
	}
	writeJSON(w, http.StatusOK, h.balance(r.Context(), key, r.URL.Query().Get("refresh") == "1"))
}

// balance returns the cached answer when it is fresh enough, else asks fal. One
// lock for the whole call: this is a single-admin app, and it keeps two tabs
// from asking fal at the same moment.
func (h *Handler) balance(ctx context.Context, key string, refresh bool) response {
	h.mu.Lock()
	defer h.mu.Unlock()

	maxAge := cacheFor
	if refresh {
		maxAge = refreshAfter
	}
	if c := h.cache; c != nil && c.key == key && h.now().Sub(c.at) < maxAge {
		return c.resp
	}

	resp := response{Configured: true}
	credits, err := h.fetch(ctx, key)
	if err != nil {
		resp.Error = friendly(err)
	} else {
		resp.Balance = &credits.Balance
		resp.Currency = credits.Currency
	}
	h.cache = &cached{key: key, at: h.now(), resp: resp}
	return resp
}

// friendly turns fal's error into a short line for the menu bar. The raw text
// can carry a snippet of fal's reply, which the menu bar has no room for.
func friendly(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "http 401"):
		return "fal rejected the billing key."
	case strings.Contains(msg, "http 403"):
		return "This key cannot read billing. Use a key with the BILLING preset."
	case strings.Contains(msg, "http 429"):
		return "fal is limiting requests; try again shortly."
	default:
		return "Could not read the balance from fal."
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
