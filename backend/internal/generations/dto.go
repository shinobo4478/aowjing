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
	ImageURL         string `json:"imageUrl"`   // "" for text-to-video
	ImageID          string `json:"imageId"`    // uploaded picture, "" if none
	EndImageID       string `json:"endImageId"` // picture the clip was asked to end on, "" if none
	Seconds          int    `json:"seconds"`    // clip length asked for
	ShotID           string `json:"shotId"`     // the video shot this is an attempt at, "" if none
	Output           string `json:"output"`
	OutputKind       string `json:"outputKind"` // "text" | "video"
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
		ImageURL:         g.ImageUrl,
		ImageID:          pgconv.UUIDString(g.ImageID),
		EndImageID:       pgconv.UUIDString(g.EndImageID),
		Seconds:          int(g.Seconds),
		ShotID:           pgconv.UUIDString(g.ShotID),
		Output:           g.Output,
		OutputKind:       g.OutputKind,
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
		ImageUrl:         r.ImageUrl,
		ImageID:          r.ImageID,
		EndImageID:       r.EndImageID,
		Seconds:          r.Seconds,
		ShotID:           r.ShotID,
		Output:           r.Output,
		OutputKind:       r.OutputKind,
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
		ImageUrl:         r.ImageUrl,
		ImageID:          r.ImageID,
		EndImageID:       r.EndImageID,
		Seconds:          r.Seconds,
		ShotID:           r.ShotID,
		Output:           r.Output,
		OutputKind:       r.OutputKind,
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
