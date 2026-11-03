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

// listItemDTO is a product as it appears in the catalog list, with how many
// clips it is pinned to.
type listItemDTO struct {
	productDTO
	UsedCount int `json:"usedCount"`
}

func listRowDTOs(rows []sqlc.ListProductsRow) []listItemDTO {
	out := make([]listItemDTO, len(rows))
	for i, r := range rows {
		out[i] = listItemDTO{
			productDTO: toDTO(sqlc.Product{
				ID: r.ID, Name: r.Name, ShopeeUrl: r.ShopeeUrl,
				ShopeeShopID: r.ShopeeShopID, ShopeeItemID: r.ShopeeItemID,
				AffiliateUrl: r.AffiliateUrl, PriceThb: r.PriceThb, Note: r.Note,
				CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
			}),
			UsedCount: int(r.UsedCount),
		}
	}
	return out
}

// usageDTO is one clip a product is pinned to.
type usageDTO struct {
	GenerationID string `json:"generationId"`
	ProfileID    string `json:"profileId"`
	ProfileName  string `json:"profileName"`
	TemplateName string `json:"templateName"` // "" if the template was deleted
	Position     int    `json:"position"`
	CreatedAt    string `json:"createdAt"`
}

func usageDTOs(rows []sqlc.ListProductUsageRow) []usageDTO {
	out := make([]usageDTO, len(rows))
	for i, r := range rows {
		name := ""
		if r.TemplateName != nil {
			name = *r.TemplateName
		}
		out[i] = usageDTO{
			GenerationID: pgconv.UUIDString(r.GenerationID),
			ProfileID:    pgconv.UUIDString(r.ProfileID),
			ProfileName:  r.ProfileName,
			TemplateName: name,
			Position:     int(r.Position),
			CreatedAt:    pgconv.TimeString(r.CreatedAt),
		}
	}
	return out
}

// placementDTO is one product pinned to one clip. Position is 1 or 2.
type placementDTO struct {
	GenerationID string     `json:"generationId"`
	Position     int        `json:"position"`
	Product      productDTO `json:"product"`
}

func profileRowDTO(r sqlc.ListPlacementsByProfileRow) placementDTO {
	return placementDTO{
		GenerationID: pgconv.UUIDString(r.GenerationID),
		Position:     int(r.Position),
		Product: toDTO(sqlc.Product{
			ID: r.ID, Name: r.Name, ShopeeUrl: r.ShopeeUrl,
			ShopeeShopID: r.ShopeeShopID, ShopeeItemID: r.ShopeeItemID,
			AffiliateUrl: r.AffiliateUrl, PriceThb: r.PriceThb, Note: r.Note,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}),
	}
}

func generationRowDTO(r sqlc.ListPlacementsByGenerationRow) placementDTO {
	return placementDTO{
		GenerationID: pgconv.UUIDString(r.GenerationID),
		Position:     int(r.Position),
		Product: toDTO(sqlc.Product{
			ID: r.ID, Name: r.Name, ShopeeUrl: r.ShopeeUrl,
			ShopeeShopID: r.ShopeeShopID, ShopeeItemID: r.ShopeeItemID,
			AffiliateUrl: r.AffiliateUrl, PriceThb: r.PriceThb, Note: r.Note,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}),
	}
}
