package products

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// productDTO is the API JSON shape: camelCase, plain strings for ids and
// timestamps. The Shopee ids stay server-side; the UI only needs to know
// whether they were readable (a short link hides them).
type productDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ShopeeURL    string `json:"shopeeUrl"`
	IDsKnown     bool   `json:"idsKnown"`
	AffiliateURL string `json:"affiliateUrl"`
	PriceTHB     int    `json:"priceThb"`
	Note         string `json:"note"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func toDTO(p sqlc.Product) productDTO {
	return productDTO{
		ID:           pgconv.UUIDString(p.ID),
		Name:         p.Name,
		ShopeeURL:    p.ShopeeUrl,
		IDsKnown:     p.ShopeeShopID != nil && p.ShopeeItemID != nil,
		AffiliateURL: p.AffiliateUrl,
		PriceTHB:     int(p.PriceThb),
		Note:         p.Note,
		CreatedAt:    pgconv.TimeString(p.CreatedAt),
		UpdatedAt:    pgconv.TimeString(p.UpdatedAt),
	}
}

func toDTOs(ps []sqlc.Product) []productDTO {
	out := make([]productDTO, len(ps))
	for i, p := range ps {
		out[i] = toDTO(p)
	}
	return out
}
