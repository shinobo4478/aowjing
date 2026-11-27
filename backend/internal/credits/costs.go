package credits

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// maxCostIDs bounds one call: a profile's clip list is a few dozen rows, and
// every id costs a lookup at fal.
const maxCostIDs = 200

type costsRequest struct {
	GenerationIDs []string `json:"generationIds"`
}

// costsResponse maps generation id -> USD fal charged, for the clips whose bill
// was found by this call. The rest are asked about again on the next call
// (fal's billing can lag a clip by a while). Like the balance, "no key" and "fal said
// no" are states to show quietly, not request errors, so it is always a 200.
type costsResponse struct {
	Configured bool               `json:"configured"`
	Costs      map[string]float64 `json:"costs"`
	Error      string             `json:"error,omitempty"`
}

// costs fills in the real cost of clips from fal's billing events and stores
// it on the clip, so each clip shows what it actually cost. Costs already
// stored are never re-asked, so a call with nothing new does not reach fal.
func (h *Handler) costs(w http.ResponseWriter, r *http.Request) {
	var in costsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body."})
		return
	}
	if len(in.GenerationIDs) > maxCostIDs {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Too many generation ids."})
		return
	}
	ids := make([]pgtype.UUID, 0, len(in.GenerationIDs))
	for _, s := range in.GenerationIDs {
		id, err := pgconv.ParseUUID(s)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid generation id."})
			return
		}
		ids = append(ids, id)
	}

	key, err := h.q.GetSetting(r.Context(), "falBillingKey")
	if errors.Is(err, pgx.ErrNoRows) {
		key = ""
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read settings."})
		return
	}
	key = strings.TrimSpace(key)
	resp := costsResponse{Configured: key != "", Costs: map[string]float64{}}
	if key == "" {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	rows, err := h.q.ListGenerationsToCost(r.Context(), ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read clips."})
		return
	}
	if len(rows) > 0 {
		h.fillCosts(r.Context(), key, rows, &resp)
	}
	writeJSON(w, http.StatusOK, resp)
}

// fillCosts asks fal about the clips in rows and stores the bills it finds.
func (h *Handler) fillCosts(ctx context.Context, key string, rows []sqlc.ListGenerationsToCostRow, resp *costsResponse) {
	byRequest := make(map[string]pgtype.UUID, len(rows))
	requestIDs := make([]string, 0, len(rows))
	oldest := rows[0].CreatedAt.Time
	for _, row := range rows {
		byRequest[row.FalRequestID] = row.ID
		requestIDs = append(requestIDs, row.FalRequestID)
		if row.CreatedAt.Time.Before(oldest) {
			oldest = row.CreatedAt.Time
		}
	}
	// fal searches the last 24 h unless told otherwise and caps a range at 90
	// days; start a little before the oldest clip.
	since := oldest.Add(-time.Hour)
	if floor := h.now().Add(-89 * 24 * time.Hour); since.Before(floor) {
		since = floor
	}

	found, err := h.fetchCosts(ctx, key, requestIDs, since)
	if err != nil {
		resp.Error = friendly(err)
		return
	}
	for requestID, usd := range found {
		id, ok := byRequest[requestID]
		if !ok {
			continue
		}
		if err := h.q.SetGenerationCost(ctx, sqlc.SetGenerationCostParams{ID: id, CostUsd: &usd}); err != nil {
			resp.Error = "Could not save a cost."
			continue
		}
		resp.Costs[pgconv.UUIDString(id)] = usd
	}
}
