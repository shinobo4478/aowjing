package prompttemplates

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// templateDTO is the API JSON shape: camelCase, plain strings for ids and
// timestamps, matching the frontend's PromptTemplate type.
type templateDTO struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profileId"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func toDTO(t sqlc.PromptTemplate) templateDTO {
	return templateDTO{
		ID:          pgconv.UUIDString(t.ID),
		ProfileID:   pgconv.UUIDString(t.ProfileID),
		Name:        t.Name,
		Body:        t.Body,
		Description: t.Description,
		CreatedAt:   pgconv.TimeString(t.CreatedAt),
		UpdatedAt:   pgconv.TimeString(t.UpdatedAt),
	}
}

func toDTOs(ts []sqlc.PromptTemplate) []templateDTO {
	out := make([]templateDTO, len(ts))
	for i, t := range ts {
		out[i] = toDTO(t)
	}
	return out
}
