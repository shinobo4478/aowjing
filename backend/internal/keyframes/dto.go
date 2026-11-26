package keyframes

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// keyframeDTO is the API JSON shape for one keyframe request. Once it has
// succeeded, imageIds are the candidate pictures (GET /images/{id} serves each).
type keyframeDTO struct {
	ID             string   `json:"id"`
	ProfileID      string   `json:"profileId"`
	Prompt         string   `json:"prompt"` // exactly what was sent
	Model          string   `json:"model"`
	Count          int      `json:"count"`
	ReferenceCount int      `json:"referenceCount"`
	Status         string   `json:"status"` // pending | succeeded | failed
	Error          string   `json:"error"`
	ImageIDs       []string `json:"imageIds"`
	CreatedAt      string   `json:"createdAt"`
}

func toDTO(k sqlc.Keyframe, imageIDs []pgtype.UUID) keyframeDTO {
	ids := make([]string, len(imageIDs))
	for i, id := range imageIDs {
		ids[i] = pgconv.UUIDString(id)
	}
	return keyframeDTO{
		ID:             pgconv.UUIDString(k.ID),
		ProfileID:      pgconv.UUIDString(k.ProfileID),
		Prompt:         k.Prompt,
		Model:          k.Model,
		Count:          int(k.Count),
		ReferenceCount: int(k.ReferenceCount),
		Status:         k.Status,
		Error:          k.Error,
		ImageIDs:       ids,
		CreatedAt:      pgconv.TimeString(k.CreatedAt),
	}
}
