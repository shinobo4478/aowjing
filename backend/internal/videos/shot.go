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
	// ImageURL is a public link the shot starts from; ImageID is a stored
	// picture (uploaded, or made with AI). Give at most one. Neither means the
	// shot has no picture yet, which is fine while planning but cannot be
	// generated.
	ImageURL string `json:"imageUrl"`
	ImageID  string `json:"imageId"`
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
