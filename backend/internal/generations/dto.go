package generations

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// generationDTO is the API JSON shape for one generation record.
type generationDTO struct {
	ID               string `json:"id"`
	ProfileID        string `json:"profileId"`
	PromptTemplateID string `json:"promptTemplateId"` // "" if the template was deleted
	TemplateName     string `json:"templateName"`     // "" if the template was deleted
	InputPrompt      string `json:"inputPrompt"`
	Output           string `json:"output"`
	Status           string `json:"status"`
	Error            string `json:"error"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	CreatedAt        string `json:"createdAt"`
}

func modelDTO(g sqlc.Generation, templateName string) generationDTO {
	return generationDTO{
		ID:               pgconv.UUIDString(g.ID),
		ProfileID:        pgconv.UUIDString(g.ProfileID),
		PromptTemplateID: pgconv.UUIDString(g.PromptTemplateID),
		TemplateName:     templateName,
		InputPrompt:      g.InputPrompt,
		Output:           g.Output,
		Status:           g.Status,
		Error:            g.Error,
		Provider:         g.Provider,
		Model:            g.Model,
		CreatedAt:        pgconv.TimeString(g.CreatedAt),
	}
}

func listRowDTO(r sqlc.ListGenerationsByProfileRow) generationDTO {
	return modelDTO(sqlc.Generation{
		ID:               r.ID,
		ProfileID:        r.ProfileID,
		PromptTemplateID: r.PromptTemplateID,
		InputPrompt:      r.InputPrompt,
		Output:           r.Output,
		Status:           r.Status,
		Error:            r.Error,
		Provider:         r.Provider,
		Model:            r.Model,
		CreatedAt:        r.CreatedAt,
	}, deref(r.TemplateName))
}

func getRowDTO(r sqlc.GetGenerationRow) generationDTO {
	return modelDTO(sqlc.Generation{
		ID:               r.ID,
		ProfileID:        r.ProfileID,
		PromptTemplateID: r.PromptTemplateID,
		InputPrompt:      r.InputPrompt,
		Output:           r.Output,
		Status:           r.Status,
		Error:            r.Error,
		Provider:         r.Provider,
		Model:            r.Model,
		CreatedAt:        r.CreatedAt,
	}, deref(r.TemplateName))
}

func listRowDTOs(rs []sqlc.ListGenerationsByProfileRow) []generationDTO {
	out := make([]generationDTO, len(rs))
	for i, r := range rs {
		out[i] = listRowDTO(r)
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
