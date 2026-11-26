package generations

import (
	"errors"
	"net/url"
	"strings"
)

const maxImageURLLen = 1000

// ValidateImageURL trims s and checks it can be handed to fal as a starting
// image: an absolute http(s) URL. An empty string is fine (no image) and comes
// back empty. fal fetches the image itself, so it must be publicly reachable;
// that is not checked here.
func ValidateImageURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > maxImageURLLen {
		return "", errors.New("Image link must be 1000 characters or fewer.")
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", errors.New("Paste the full image link, starting with https://.")
	}
	return s, nil
}
