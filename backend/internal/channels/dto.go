package channels

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// channelDTO is the API JSON shape: camelCase, plain strings for ids and
// timestamps, so the frontend's Channel type maps 1:1.
type channelDTO struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profileId"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Handle      string `json:"handle"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func toDTO(c sqlc.Channel) channelDTO {
	return channelDTO{
		ID:          pgconv.UUIDString(c.ID),
		ProfileID:   pgconv.UUIDString(c.ProfileID),
		Name:        c.Name,
		Platform:    c.Platform,
		Handle:      c.Handle,
		Description: c.Description,
		CreatedAt:   pgconv.TimeString(c.CreatedAt),
		UpdatedAt:   pgconv.TimeString(c.UpdatedAt),
	}
}

func toDTOs(cs []sqlc.Channel) []channelDTO {
	out := make([]channelDTO, len(cs))
	for i, c := range cs {
		out[i] = toDTO(c)
	}
	return out
}
