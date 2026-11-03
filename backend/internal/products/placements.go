package products

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
)

// MaxPerClip is how many products can be pinned to one clip. The database
// enforces it too (position is 1 or 2); this is the friendly error first.
const MaxPerClip = 2

// PlacementHandler pins catalog products to clips (generations). A placement
// belongs to the clip, not to a channel or platform — each platform renders it
// in its own way when the clip is published.
type PlacementHandler struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewPlacementHandler(db *pgxpool.Pool, q *sqlc.Queries) *PlacementHandler {
	return &PlacementHandler{db: db, q: q}
}

// Routes returns the router to mount at /placements.
func (h *PlacementHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Put("/{generationId}", h.set)
	return r
}

// ParseProductIDs checks a requested set of product ids: at most MaxPerClip,
// each a valid UUID, no repeats. Order is kept — it becomes the pin order.
func ParseProductIDs(ids []string) ([]pgtype.UUID, error) {
	if len(ids) > MaxPerClip {
		return nil, errors.New("A clip can have at most 2 products.")
	}
	out := make([]pgtype.UUID, 0, len(ids))
	seen := map[string]bool{}
	for _, s := range ids {
		id, err := pgconv.ParseUUID(s)
		if err != nil {
			return nil, errors.New("Invalid product id.")
		}
		key := pgconv.UUIDString(id)
		if seen[key] {
			return nil, errors.New("The same product can't be pinned twice.")
		}
		seen[key] = true
		out = append(out, id)
	}
	return out, nil
}

var errProductMissing = errors.New("Product not found.")

// list returns every placement under a profile, so the generations table can
// show pinned products without a request per row.
func (h *PlacementHandler) list(w http.ResponseWriter, r *http.Request) {
	profileID, err := pgconv.ParseUUID(r.URL.Query().Get("profileId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "A valid profileId query parameter is required.")
		return
	}
	rows, err := h.q.ListPlacementsByProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list pinned products.")
		return
	}
	out := make([]placementDTO, len(rows))
	for i, row := range rows {
		out[i] = profileRowDTO(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"placements": out})
}

// set replaces the products pinned to one clip with the given list (0–2 ids,
// in pin order). Clearing and re-adding happen in one transaction, so a bad
// id can't leave the clip half-updated.
func (h *PlacementHandler) set(w http.ResponseWriter, r *http.Request) {
	genID, err := pgconv.ParseUUID(chi.URLParam(r, "generationId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid generation id.")
		return
	}

	var body struct {
		ProductIDs []string `json:"productIds"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	ids, err := ParseProductIDs(body.ProductIDs)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"productIds": err.Error()},
		})
		return
	}

	if ok, err := h.q.GenerationExists(r.Context(), genID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to pin products.")
		return
	} else if !ok {
		writeError(w, http.StatusNotFound, "Generation not found.")
		return
	}

	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return replacePlacements(r.Context(), h.q.WithTx(tx), genID, ids)
	})
	if errors.Is(err, errProductMissing) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"productIds": err.Error()},
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to pin products.")
		return
	}

	rows, err := h.q.ListPlacementsByGeneration(r.Context(), genID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Pinned, but failed to read them back.")
		return
	}
	out := make([]placementDTO, len(rows))
	for i, row := range rows {
		out[i] = generationRowDTO(row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"placements": out})
}

// replacePlacements runs inside a transaction (q is bound to it).
func replacePlacements(ctx context.Context, q *sqlc.Queries, genID pgtype.UUID, ids []pgtype.UUID) error {
	for _, id := range ids {
		if _, err := q.GetProduct(ctx, id); errors.Is(err, pgx.ErrNoRows) {
			return errProductMissing
		} else if err != nil {
			return err
		}
	}
	if err := q.ClearPlacements(ctx, genID); err != nil {
		return err
	}
	for i, id := range ids {
		if err := q.AddPlacement(ctx, sqlc.AddPlacementParams{
			GenerationID: genID,
			ProductID:    id,
			Position:     int16(i + 1),
		}); err != nil {
			return err
		}
	}
	return nil
}
