package studio

import "unicode"

// SpeechCharsPerSecond is how many Thai characters (white space not counted) a
// narrator speaks in a second. A GUESS used before any audio exists (the same
// number as SPEECH_CHARS_PER_SECOND in frontend/lib/narration.ts): once a shot
// has real audio its measured length is what counts. Adjust both after a few
// real voice-overs.
const SpeechCharsPerSecond = 11

// NarrationTailSeconds is the silence left after the last word, so a clip does
// not cut the voice off.
const NarrationTailSeconds = 0.5

// SpeechChars counts the characters that are spoken: everything but white space.
func SpeechChars(text string) int {
	n := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// NarrationBudget is the most narration characters that fit a shot of maxSeconds
// (the tail included). It is what the script helper is told to stay under.
func NarrationBudget(maxSeconds int) int {
	b := int((float64(maxSeconds) - NarrationTailSeconds) * SpeechCharsPerSecond)
	if b < 0 {
		return 0
	}
	return b
}

// SecondsForNarration is the shortest clip length, among the lengths a video
// model takes, that holds the narration plus the tail. fits is false when even
// the longest is too short: the longest is returned then, and the narration
// should be shortened. With no options it returns 0, false.
func SecondsForNarration(text string, options []int) (seconds int, fits bool) {
	if len(options) == 0 {
		return 0, false
	}
	need := float64(SpeechChars(text))/SpeechCharsPerSecond + NarrationTailSeconds
	longest := options[0]
	best := 0
	for _, s := range options {
		if s > longest {
			longest = s
		}
		if float64(s) >= need && (best == 0 || s < best) {
			best = s
		}
	}
	if best == 0 {
		return longest, false
	}
	return best, true
}

// DefaultExplainerSeconds is the longest shot offered for a narrated video when
// the owner does not choose: 8 s holds about 80 characters of Thai narration.
const DefaultExplainerSeconds = 8

// ExplainerSeconds is the default longest shot for a narrated video on a model
// that takes the given lengths: DefaultExplainerSeconds if the model takes it,
// else the longest length under it (or the shortest, if all are over).
func ExplainerSeconds(options []int) int {
	best, shortest := 0, 0
	for _, s := range options {
		if shortest == 0 || s < shortest {
			shortest = s
		}
		if s <= DefaultExplainerSeconds && s > best {
			best = s
		}
	}
	if best == 0 {
		return shortest
	}
	return best
}
