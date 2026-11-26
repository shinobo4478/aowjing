package studio

// Keyframes: the picture a clip starts from. An image model makes a few
// candidates from a prompt (and, when the profile has them, its reference
// pictures so the character stays the same); the owner picks one; it becomes
// the starting image of the image-to-video clip.
//
// The choice of image model is per request, not per style, so the models are a
// registry of their own. Which fields each model's request needs is
// provider-specific and lives in internal/generate, keyed by Key.

// KeyframeAspect is the shape of every keyframe: vertical, for Shorts/Reels.
// Kling accepts width/height ratios from 0.4 to 2.5, and 9:16 (0.5625) is in.
const KeyframeAspect = "9:16"

const (
	// MaxKeyframeImages is how many candidates one request may ask for. Both
	// models allow 4 or more; more is just more cost.
	MaxKeyframeImages = 4

	// MaxReferences is how many reference pictures a profile may keep.
	MaxReferences = 4
)

// KeyframeModel is one image model the owner can choose.
type KeyframeModel struct {
	// Key is the stable name stored on a keyframe request and sent by the UI.
	// internal/generate switches on it to build the right request fields.
	Key   string
	Label string

	// TextModel makes a picture from a prompt alone; EditModel also takes
	// reference pictures. Both are fal model ids.
	TextModel string
	EditModel string

	// PriceUSD is the price of one picture as fal's model page lists it
	// (read 2026-11-26, not yet checked against a real bill).
	PriceUSD float64
}

var keyframeModels = []KeyframeModel{
	{
		Key:       "nano-banana",
		Label:     "Nano Banana (Gemini image)",
		TextModel: "fal-ai/nano-banana",
		EditModel: "fal-ai/nano-banana/edit",
		PriceUSD:  0.0398,
	},
	{
		Key:       "seedream",
		Label:     "Seedream 4",
		TextModel: "fal-ai/bytedance/seedream/v4/text-to-image",
		EditModel: "fal-ai/bytedance/seedream/v4/edit",
		PriceUSD:  0.03,
	},
}

// KeyframeModels lists the models in display order (a copy). The first is the
// default.
func KeyframeModels() []KeyframeModel {
	return append([]KeyframeModel(nil), keyframeModels...)
}

// KeyframeModelFor finds a model by key.
func KeyframeModelFor(key string) (KeyframeModel, bool) {
	for _, m := range keyframeModels {
		if m.Key == key {
			return m, true
		}
	}
	return KeyframeModel{}, false
}

// ModelID is the fal model to call: the edit model when reference pictures go
// with the request, the text model otherwise.
func (m KeyframeModel) ModelID(hasReferences bool) string {
	if hasReferences {
		return m.EditModel
	}
	return m.TextModel
}

// CostUSD estimates the price of asking for count pictures.
func (m KeyframeModel) CostUSD(count int) float64 {
	return m.PriceUSD * float64(count)
}

// keepLookInstruction tells an edit model to reuse what is in the reference
// pictures. A starting guess, not tested on real output yet.
const keepLookInstruction = "Keep the same character and overall look as in the reference pictures."

// KeyframePrompt is the prompt to send for a keyframe: the owner's scene
// description, an instruction to reuse the references when there are any, and
// this style's wording (so the picture already has the look the clip should keep).
func (r Recipe) KeyframePrompt(prompt string, hasReferences bool) string {
	out := prompt
	if hasReferences {
		out = keepLookInstruction + " " + out
	}
	if r.KeyframeSuffix != "" {
		out += "\n\n" + r.KeyframeSuffix
	}
	return out
}
