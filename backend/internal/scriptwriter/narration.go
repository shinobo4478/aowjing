package scriptwriter

import (
	"errors"
	"strings"
	"unicode"
)

// narrationProblem returns what is wrong with a drafted narration, or nil. A
// narration is spoken by a Thai voice that takes its accent from the first
// words and drifts into English sounds on Latin letters or digits, so the
// narration must be Thai letters only and must start with a Thai letter. The
// parser asks the model again (once) when a narration fails, telling it why.
func narrationProblem(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("is empty")
	}
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Latin, r):
			return errors.New("contains Latin letters (write every word in Thai letters)")
		case unicode.IsDigit(r):
			return errors.New("contains digits (spell numbers out in Thai words)")
		}
	}
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if !unicode.Is(unicode.Thai, r) {
			return errors.New("does not start with a Thai letter")
		}
		break
	}
	return nil
}
