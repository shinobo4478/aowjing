package profiles

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// profileDTO is the JSON shape the API exposes. It is deliberately decoupled
// from sqlc.Profile: camelCase keys, plain strings for the id and timestamps,
// so the frontend's Profile type maps 1:1.
type profileDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Niche       string `json:"niche"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func toDTO(p sqlc.Profile) profileDTO {
	return profileDTO{
		ID:          pgconv.UUIDString(p.ID),
		Name:        p.Name,
		Niche:       p.Niche,
		Description: p.Description,
		CreatedAt:   pgconv.TimeString(p.CreatedAt),
		UpdatedAt:   pgconv.TimeString(p.UpdatedAt),
	}
}

func toDTOs(ps []sqlc.Profile) []profileDTO {
	out := make([]profileDTO, len(ps))
	for i, p := range ps {
		out[i] = toDTO(p)
	}
	return out
}
