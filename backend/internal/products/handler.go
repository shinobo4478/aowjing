// Package products implements the central Shopee product catalog (Phase 3).
// A product belongs to no profile or channel; the collection is /products
// (optionally ?q= to search by name) and items are /products/{id}.
package products

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/shopee"
)

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

// Routes returns the router to mount at /products.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

type productInput struct {
	Name         string `json:"name"`
	ShopeeURL    string `json:"shopeeUrl"`
	AffiliateURL string `json:"affiliateUrl"`
	PriceTHB     int    `json:"priceThb"`
	Note         string `json:"note"`
}

// normalizeAndValidate trims the fields, checks them, and returns the parsed
// Shopee link (canonical URL + ids). The returned map is empty when valid.
func (in *productInput) normalizeAndValidate() (shopee.Link, map[string]string) {
	in.Name = strings.TrimSpace(in.Name)
	in.ShopeeURL = strings.TrimSpace(in.ShopeeURL)
	in.AffiliateURL = strings.TrimSpace(in.AffiliateURL)
	in.Note = strings.TrimSpace(in.Note)

	errs := map[string]string{}
	switch {
	case len(in.Name) < 2:
		errs["name"] = "Name must be at least 2 characters."
	case len(in.Name) > 120:
		errs["name"] = "Name must be 120 characters or fewer."
	}

	link, err := shopee.ParseLink(in.ShopeeURL)
	if err != nil {
		errs["shopeeUrl"] = "Paste a shopee.co.th product link."
	}

	if in.AffiliateURL != "" {
		u, perr := url.Parse(in.AffiliateURL)
		switch {
		case perr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "":
			errs["affiliateUrl"] = "Paste the full affiliate link, starting with https://."
		case len(in.AffiliateURL) > 500:
			errs["affiliateUrl"] = "Affiliate link must be 500 characters or fewer."
		}
	}

	if in.PriceTHB < 0 || in.PriceTHB > 10_000_000 {
		errs["priceThb"] = "Price must be between 0 and 10,000,000 baht."
	}
	if len(in.Note) > 600 {
		errs["note"] = "Note must be 600 characters or fewer."
	}
	return link, errs
}

// idPtrs returns the link's shop/item ids as nullable values for the DB: nil
// for short links, whose ids we don't know.
func idPtrs(l shopee.Link) (shop, item *int64) {
	if !l.HasIDs {
		return nil, nil
	}
	return &l.ShopID, &l.ItemID
}

// duplicateOf reports the name of another catalog product with the same
// Shopee ids, if any. self is the product being edited (zero value for a new
// one) so a product never collides with itself.
func (h *Handler) duplicateOf(ctx context.Context, l shopee.Link, self pgtype.UUID) (string, bool, error) {
	if !l.HasIDs {
		return "", false, nil
	}
	shop, item := idPtrs(l)
	found, err := h.q.FindProductByShopeeIDs(ctx, sqlc.FindProductByShopeeIDsParams{
		ShopeeShopID: shop,
		ShopeeItemID: item,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if self.Valid && found.ID == self {
		return "", false, nil
	}
	return found.Name, true, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := h.q.ListProducts(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list products.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": toDTOs(rows)})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in productInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	link, errs := in.normalizeAndValidate()
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	if name, dup, err := h.duplicateOf(r.Context(), link, pgtype.UUID{}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create product.")
		return
	} else if dup {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"shopeeUrl": "This product is already in the catalog as “" + name + "”."},
		})
		return
	}

	shop, item := idPtrs(link)
	row, err := h.q.CreateProduct(r.Context(), sqlc.CreateProductParams{
		Name:         in.Name,
		ShopeeUrl:    link.URL,
		ShopeeShopID: shop,
		ShopeeItemID: item,
		AffiliateUrl: in.AffiliateURL,
		PriceThb:     int32(in.PriceTHB),
		Note:         in.Note,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create product.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"product": toDTO(row)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id.")
		return
	}
	row, err := h.q.GetProduct(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load product.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": toDTO(row)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id.")
		return
	}

	var in productInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	link, errs := in.normalizeAndValidate()
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	if name, dup, err := h.duplicateOf(r.Context(), link, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update product.")
		return
	} else if dup {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"shopeeUrl": "This product is already in the catalog as “" + name + "”."},
		})
		return
	}

	shop, item := idPtrs(link)
	row, err := h.q.UpdateProduct(r.Context(), sqlc.UpdateProductParams{
		ID:           id,
		Name:         in.Name,
		ShopeeUrl:    link.URL,
		ShopeeShopID: shop,
		ShopeeItemID: item,
		AffiliateUrl: in.AffiliateURL,
		PriceThb:     int32(in.PriceTHB),
		Note:         in.Note,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update product.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": toDTO(row)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id.")
		return
	}
	n, err := h.q.DeleteProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete product.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
