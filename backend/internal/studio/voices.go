package studio

import "unicode/utf8"

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
const MaxVoiceCandidates = 7

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

// The MiniMax Thai voice ids are from MiniMax's own list of system voices
// (platform.minimax.io, "system voice id"); fal's pages do not say whether its
// endpoints accept them, so the first sample is also the test. ElevenLabs' fal
// pages do not list Thai at all, so those candidates may answer with an error,
// which the screen shows as it is.
var voiceCandidates = []VoiceCandidate{
	{
		Key: "minimax-hd-woman", Label: "MiniMax 2.8 HD · Confident Woman",
		Family: VoiceMiniMax, Model: "fal-ai/minimax/speech-2.8-hd", VoiceID: "Thai_female_1_sample1",
		PricePer1KChars: 0.10,
		Note:            "A Thai voice from MiniMax's list. Thai is a listed language option.",
	},
	{
		Key: "minimax-hd-man", Label: "MiniMax 2.8 HD · Serene Man",
		Family: VoiceMiniMax, Model: "fal-ai/minimax/speech-2.8-hd", VoiceID: "Thai_male_1_sample8",
		PricePer1KChars: 0.10,
		Note:            "A Thai voice from MiniMax's list. Thai is a listed language option.",
	},
	{
		Key: "minimax-hd-energetic", Label: "MiniMax 2.8 HD · Energetic Woman",
		Family: VoiceMiniMax, Model: "fal-ai/minimax/speech-2.8-hd", VoiceID: "Thai_female_2_sample2",
		PricePer1KChars: 0.10,
		Note:            "A livelier Thai voice from MiniMax's list.",
	},
	{
		Key: "minimax-turbo-woman", Label: "MiniMax 2.8 Turbo · Confident Woman",
		Family: VoiceMiniMax, Model: "fal-ai/minimax/speech-2.8-turbo", VoiceID: "Thai_female_1_sample1",
		PricePer1KChars: 0.06,
		Note:            "The cheaper, faster MiniMax model with the same voice: listen for what is lost.",
	},
	{
		Key: "minimax-turbo-man", Label: "MiniMax 2.8 Turbo · Serene Man",
		Family: VoiceMiniMax, Model: "fal-ai/minimax/speech-2.8-turbo", VoiceID: "Thai_male_1_sample8",
		PricePer1KChars: 0.06,
		Note:            "The cheaper, faster MiniMax model with the same voice: listen for what is lost.",
	},
	{
		Key: "eleven-v3", Label: "ElevenLabs v3 · Rachel",
		Family: VoiceElevenLabs, Model: "fal-ai/elevenlabs/tts/eleven-v3", VoiceID: "Rachel",
		PricePer1KChars: 0.10,
		Note:            "fal's page does not say Thai is supported (ElevenLabs' own site does). Rachel is a default English-named voice: it may read Thai with an accent, or fail.",
	},
	{
		Key: "eleven-multilingual-v2", Label: "ElevenLabs Multilingual v2 · Rachel",
		Family: VoiceElevenLabs, Model: "fal-ai/elevenlabs/tts/multilingual-v2", VoiceID: "Rachel",
		PricePer1KChars: 0.10,
		Note:            "Same caveat as v3: Thai is not listed on fal's page.",
	},
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
