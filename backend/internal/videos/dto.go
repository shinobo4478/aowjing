package videos

import (
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// videoDTO is the API JSON shape for one video. ShotCount and TotalSeconds are
// filled in the list; the detail response carries the shots themselves.
type videoDTO struct {
	ID           string `json:"id"`
	ProfileID    string `json:"profileId"`
	Title        string `json:"title"`
	ShotCount    int    `json:"shotCount"`
	TotalSeconds int    `json:"totalSeconds"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// attemptDTO is the newest attempt at a shot (a row of generations).
type attemptDTO struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // pending | succeeded | failed
	Output    string `json:"output"` // the clip URL once succeeded
	Error     string `json:"error"`
	Seconds   int    `json:"seconds"`
	CreatedAt string `json:"createdAt"`
}

type shotDTO struct {
	ID       string `json:"id"`
	VideoID  string `json:"videoId"`
	Position int    `json:"position"`
	Prompt   string `json:"prompt"`
	// PicturePrompt is what the starting picture should show (for Make with AI).
	PicturePrompt string `json:"picturePrompt"`
	// EndPicturePrompt: what changes by the end picture (only a pose or position).
	EndPicturePrompt string `json:"endPicturePrompt"`
	Seconds          int    `json:"seconds"`
	ImageURL         string `json:"imageUrl"`
	ImageID          string `json:"imageId"`
	// EndImageID is the picture the clip should end on, "" if none.
	EndImageID string `json:"endImageId"`
	// Approved: the owner OKed this plan on the storyboard, so it may be made in
	// bulk. Changing the plan clears it.
	Approved bool `json:"approved"`
	// Latest is the newest attempt, nil if the shot has never been generated;
	// Attempts counts all of them (each was paid for).
	Latest   *attemptDTO `json:"latest"`
	Attempts int         `json:"attempts"`
	// Take is the clip to USE for this shot: the attempt the owner chose, else the
	// newest successful one; nil while none has succeeded. TakeChosen says which.
	Take       *attemptDTO `json:"take"`
	TakeChosen bool        `json:"takeChosen"`
}

func toVideoDTO(v sqlc.Video, shotCount, totalSeconds int) videoDTO {
	return videoDTO{
		ID:           pgconv.UUIDString(v.ID),
		ProfileID:    pgconv.UUIDString(v.ProfileID),
		Title:        v.Title,
		ShotCount:    shotCount,
		TotalSeconds: totalSeconds,
		CreatedAt:    pgconv.TimeString(v.CreatedAt),
		UpdatedAt:    pgconv.TimeString(v.UpdatedAt),
	}
}

func toShotDTO(s sqlc.VideoShot, latest *sqlc.ListShotLatestGenerationsRow, take *sqlc.ListShotTakesRow) shotDTO {
	d := shotDTO{
		ID:               pgconv.UUIDString(s.ID),
		VideoID:          pgconv.UUIDString(s.VideoID),
		Position:         int(s.Position),
		Prompt:           s.Prompt,
		PicturePrompt:    s.PicturePrompt,
		EndPicturePrompt: s.EndPicturePrompt,
		Seconds:          int(s.Seconds),
		ImageURL:         s.ImageUrl,
		ImageID:          pgconv.UUIDString(s.ImageID),
		EndImageID:       pgconv.UUIDString(s.EndImageID),
		Approved:         s.Approved,
	}
	if latest != nil {
		d.Attempts = int(latest.Attempts)
		d.Latest = &attemptDTO{
			ID:        pgconv.UUIDString(latest.ID),
			Status:    latest.Status,
			Output:    latest.Output,
			Error:     latest.Error,
			Seconds:   int(latest.Seconds),
			CreatedAt: pgconv.TimeString(latest.CreatedAt),
		}
	}
	if take != nil {
		d.TakeChosen = take.Chosen
		d.Take = &attemptDTO{
			ID:        pgconv.UUIDString(take.ID),
			Status:    take.Status,
			Output:    take.Output,
			Error:     take.Error,
			Seconds:   int(take.Seconds),
			CreatedAt: pgconv.TimeString(take.CreatedAt),
		}
	}
	return d
}
