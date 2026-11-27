package videos

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/shinobo4478/aowjing/backend/internal/generations"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

const (
	// MaxShots is how many shots one video may have. The database also limits
	// the position to 1-12; this is the friendly error first.
	MaxShots = 12

	maxTitleRunes  = 120
	maxPromptRunes = 2000
)

// shotInput is what the screen sends to change a shot: the whole plan of the
// shot at once, not a patch of single fields.
type shotInput struct {
	Prompt  string `json:"prompt"`
	Seconds int    `json:"seconds"`
	// PicturePrompt is what the starting picture should show (used by Make with
	// AI). It is the owner's note to the image model, not sent to the clip model.
	PicturePrompt string `json:"picturePrompt"`
	// EndPicturePrompt says what changes between the starting picture and the end
	// picture (only a pose or position; camera and room stay). Used by Make with AI
	// when making the end picture; not sent to the clip model.
	EndPicturePrompt string `json:"endPicturePrompt"`
	// The Thai versions of the three prompts, which the owner writes and edits;
	// the English ones above are what the models receive. Never sent to a model.
	PromptTh           string `json:"promptTh"`
	PicturePromptTh    string `json:"picturePromptTh"`
	EndPicturePromptTh string `json:"endPicturePromptTh"`
	// ImageURL is a public link the shot starts from; ImageID is a stored
	// picture (uploaded, or made with AI). Give at most one. Neither means the
	// shot has no picture yet, which is fine while planning but cannot be
	// generated.
	ImageURL string `json:"imageUrl"`
	ImageID  string `json:"imageId"`
	// EndImageID is a stored picture the clip should END on (fal's
	// end_image_url): it flows from the starting picture to this one. Only with a
	// starting picture, and a stored picture only (never a link).
	EndImageID string `json:"endImageId"`
}

// normalizeAndValidate trims the input and returns a message per field; an
// empty map means it is valid. ImageID is only checked for being well formed
// here: whether the picture exists is the handler's question.
func (in *shotInput) normalizeAndValidate() map[string]string {
	errs := map[string]string{}

	in.Prompt = strings.TrimSpace(in.Prompt)
	if utf8.RuneCountInString(in.Prompt) > maxPromptRunes {
		errs["prompt"] = "The prompt must be 2000 characters or fewer."
	}

	// The Thai versions: trimmed and length-checked like the English ones.
	for name, field := range map[string]*string{
		"promptTh":           &in.PromptTh,
		"picturePromptTh":    &in.PicturePromptTh,
		"endPicturePromptTh": &in.EndPicturePromptTh,
	} {
		*field = strings.TrimSpace(*field)
		if utf8.RuneCountInString(*field) > maxPromptRunes {
			errs[name] = "The Thai text must be 2000 characters or fewer."
		}
	}
	in.EndPicturePrompt = strings.TrimSpace(in.EndPicturePrompt)
	if utf8.RuneCountInString(in.EndPicturePrompt) > maxPromptRunes {
		errs["endPicturePrompt"] = "The end picture prompt must be 2000 characters or fewer."
	}
	in.PicturePrompt = strings.TrimSpace(in.PicturePrompt)
	if utf8.RuneCountInString(in.PicturePrompt) > maxPromptRunes {
		errs["picturePrompt"] = "The picture prompt must be 2000 characters or fewer."
	}

	if in.Seconds == 0 {
		in.Seconds = studio.DefaultClipSeconds
	}
	if !studio.ValidClipSeconds(in.Seconds) {
		errs["seconds"] = "A shot is 3 to 15 seconds long."
	}

	url, err := generations.ValidateImageURL(in.ImageURL)
	if err != nil {
		errs["imageUrl"] = err.Error()
	}
	in.ImageURL = url
	in.ImageID = strings.TrimSpace(in.ImageID)
	if in.ImageID != "" && in.ImageURL != "" {
		errs["imageId"] = "Give an image link or a picture, not both."
	}
	in.EndImageID = strings.TrimSpace(in.EndImageID)
	if in.EndImageID != "" && in.ImageURL == "" && in.ImageID == "" {
		errs["endImageId"] = "Choose a starting picture before an end picture."
	}
	return errs
}

// validateTitle trims a video title and checks its length.
func validateTitle(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch n := utf8.RuneCountInString(s); {
	case n == 0:
		return "", errors.New("Give the video a title.")
	case n > maxTitleRunes:
		return "", errors.New("The title must be 120 characters or fewer.")
	}
	return s, nil
}

// sameSet reports whether got contains exactly the ids in want, each once, in
// any order. Used to check a reorder names every shot of the video.
func sameSet(want, got []string) bool {
	if len(want) != len(got) {
		return false
	}
	seen := make(map[string]bool, len(want))
	for _, id := range want {
		seen[id] = true
	}
	for _, id := range got {
		if !seen[id] {
			return false
		}
		delete(seen, id) // a repeated id finds nothing the second time
	}
	return len(seen) == 0
}
