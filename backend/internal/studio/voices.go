package studio

import (
	"strings"
	"unicode/utf8"
)

// Thai voice-over: the voices the owner can listen to before choosing one. The
// clips are silent (the video models' own audio is not Thai), so narration comes
// from a text-to-speech model; nothing is chosen yet, this registry feeds the
// "voice samples" screen, where the SAME Thai text is spoken by each candidate so
// the owner picks by ear (CLAUDE.md, "The 2D style and Thai voice-over").
//
// Every candidate runs on fal with the one key already in Settings. Which fields
// each family's request needs lives in internal/generate (voice.go), keyed by
// Family, like the picture models.

// VoiceFamily says how a candidate's fal endpoint is called.
type VoiceFamily string

const (
	// VoiceMiniMax: field `prompt`, `language_boost`, `voice_setting.voice_id`;
	// the reply carries the audio URL and `duration_ms`.
	VoiceMiniMax VoiceFamily = "minimax"
	// VoiceElevenLabs: field `text`, `voice` (a name), `language_code`; the reply
	// carries the audio URL and no duration.
	VoiceElevenLabs VoiceFamily = "elevenlabs"
)

// MaxVoiceSampleChars caps the text of one samples request. Every candidate
// speaks all of it, so the price is the length times the candidates.
const MaxVoiceSampleChars = 1500

// MinVoiceSampleChars is the shortest text worth judging a voice by.
const MinVoiceSampleChars = 20

// MaxVoiceCandidates is how many voices one samples request may ask for.
const MaxVoiceCandidates = 12

// VoiceCandidate is one voice the owner can listen to.
type VoiceCandidate struct {
	// Key is the stable name the screen sends.
	Key   string
	Label string
	// Family and Model say how to call it (Model is the fal endpoint id).
	Family VoiceFamily
	Model  string
	// VoiceID is the MiniMax voice id or the ElevenLabs voice name; empty means the
	// endpoint's own default.
	VoiceID string
	// PricePer1KChars is what fal lists, in US dollars per 1000 characters.
	PricePer1KChars float64
	// Note says what is and is not known about this voice for Thai, in plain
	// words for the screen.
	Note string
}

// CostUSD estimates the price of speaking text: fal bills per character.
func (v VoiceCandidate) CostUSD(text string) float64 {
	return v.PricePer1KChars * float64(utf8.RuneCountInString(text)) / 1000
}

// The owner listened to MiniMax 2.8 (HD and Turbo, the Thai system voices) and to
// ElevenLabs v3 on 2026-11-28 with the same Thai text: MiniMax misread a lot,
// ElevenLabs v3 read it without a mistake with an acceptable accent. So the
// registry now holds ElevenLabs voices, to choose a narrator's gender and tone by
// ear; MiniMax stays supported by internal/generate but is not offered. Voice names
// are from fal's examples (the `voice` field is a free string); none is documented
// as Thai-specific, so how each reads Thai is the owner's to judge.
var voiceCandidates = []VoiceCandidate{
	elevenV3("rachel", "Rachel", "Heard 2026-11-28: read the Thai text without a mistake, accent acceptable."),
	elevenV3("sarah", "Sarah", "Heard 2026-11-28 and kept by the owner."),
	elevenV3("roger", "Roger", "Heard 2026-11-28 and kept by the owner."),
	elevenV3("chris", "Chris", "Heard 2026-11-28 and kept by the owner."),
}

// elevenV3 is an ElevenLabs v3 voice by name. An empty note gets the standard one.
func elevenV3(slug, name, note string) VoiceCandidate {
	if note == "" {
		note = "An ElevenLabs v3 voice by name. How it reads Thai is yours to judge: listen to it against Rachel."
	}
	return VoiceCandidate{
		Key: "eleven-v3-" + slug, Label: "ElevenLabs v3 · " + name,
		Family: VoiceElevenLabs, Model: "fal-ai/elevenlabs/tts/eleven-v3", VoiceID: name,
		PricePer1KChars: 0.10, Note: note,
	}
}

// VoiceCandidates lists every candidate in display order. The slice is a copy.
func VoiceCandidates() []VoiceCandidate { return append([]VoiceCandidate(nil), voiceCandidates...) }

// VoiceCandidateFor finds a candidate by key.
func VoiceCandidateFor(key string) (VoiceCandidate, bool) {
	for _, v := range voiceCandidates {
		if v.Key == key {
			return v, true
		}
	}
	return VoiceCandidate{}, false
}

// MaxCustomVoiceChars caps a voice name or id the owner types in.
const MaxCustomVoiceChars = 64

// CustomVoice makes a candidate for an ElevenLabs v3 voice the owner names
// themselves: a name from fal's examples, or an ElevenLabs voice id (for example
// a Voice Library voice). ElevenLabs' docs say the Default voices expire on
// 2026-12-31 and are replaced by "Infinity" voices, whose names and ids those
// pages do not give, and fal's page does not say whether its `voice` field takes
// ids at all; this lets the owner try one and hear whether it works. The name is
// sent to fal as it is, so only letters, digits, spaces, "_" and "-" are allowed.
func CustomVoice(name string) (VoiceCandidate, bool) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxCustomVoiceChars {
		return VoiceCandidate{}, false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ', r == '_', r == '-':
		default:
			return VoiceCandidate{}, false
		}
	}
	return VoiceCandidate{
		Key: "custom:" + name, Label: "ElevenLabs v3 · " + name + " (your voice)",
		Family: VoiceElevenLabs, Model: "fal-ai/elevenlabs/tts/eleven-v3", VoiceID: name,
		PricePer1KChars: 0.10,
		Note:            "A voice you named. If fal does not know it, it answers with an error and costs nothing.",
	}, true
}
