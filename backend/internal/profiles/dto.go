package profiles

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
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
		ID:          uuidString(p.ID),
		Name:        p.Name,
		Niche:       p.Niche,
		Description: p.Description,
		CreatedAt:   timeString(p.CreatedAt),
		UpdatedAt:   timeString(p.UpdatedAt),
	}
}

func toDTOs(ps []sqlc.Profile) []profileDTO {
	out := make([]profileDTO, len(ps))
	for i, p := range ps {
		out[i] = toDTO(p)
	}
	return out
}

// uuidString renders a pgtype.UUID as the canonical 8-4-4-4-12 hex form.
// Done by hand to stay independent of pgtype version differences.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func timeString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// parseUUID converts a path-parameter string into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}
