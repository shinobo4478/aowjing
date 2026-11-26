// Package studio holds the video "recipes": what each video style means in
// terms of models and prompt wording. A Profile stores only the style's name
// (profiles.video_style); everything the style implies lives in the registry
// below, so swapping a model or tweaking a style's wording is a one-place edit
// that touches no handler and no screen.
//
// Provider-specific code (how to talk to fal) stays in internal/generate, and
// credentials stay in Settings. This package only says WHICH models and WHAT
// extra wording.
//
// Not here yet: the audio route. Every style makes silent clips and Thai
// narration is added separately (CLAUDE.md, Phase 4); it joins the recipe when
// the voice-over step (item 5) exists.
package studio

// Style is the name of a video style, as stored on a Profile.
type Style string

const (
	// Style3D is the Pixar-like 3D look, the first style built.
	Style3D Style = "3d"

	// DefaultStyle is what a profile gets when none is given.
	DefaultStyle = Style3D
)

// Recipe is everything a style implies for making a clip.
type Recipe struct {
	Style Style
	// Label is the name shown to the owner.
	Label string

	// TextModel makes a clip from a prompt alone; ImageModel starts from a
	// picture. Both are fal model ids.
	TextModel  string
	ImageModel string

	// TextSuffix is appended to the prompt of a clip made from text alone: with
	// no picture, the prompt is the only thing that sets the look.
	TextSuffix string
	// ImageSuffix is appended when the clip starts from a picture. The picture
	// already sets the look (a real product photo must not be restyled), so
	// this is only for motion wording, and empty for most styles.
	ImageSuffix string
	// KeyframeSuffix is appended to the prompt of a keyframe picture. Unlike a
	// clip that starts from a picture, here the look is being decided, so the
	// style's wording belongs in the prompt.
	KeyframeSuffix string
}

// recipes is the registry, in the order shown in the UI. To add a style, add a
// Style constant and an entry here (plus its label on the frontend).
var recipes = []Recipe{
	{
		Style: Style3D,
		Label: "3D (Pixar-like)",

		TextModel:  "fal-ai/kling-video/v3/standard/text-to-video",
		ImageModel: "fal-ai/kling-video/v3/standard/image-to-video",

		// Starting wording, not tested against real clips yet: adjust it after
		// seeing what Kling makes of it.
		TextSuffix:     "3D animated film style, Pixar-like characters, soft cinematic lighting, smooth natural motion.",
		KeyframeSuffix: "3D animated film style, Pixar-like characters, soft cinematic lighting.",
	},
}

// Recipes lists every style in display order. The slice is a copy.
func Recipes() []Recipe { return append([]Recipe(nil), recipes...) }

// Valid reports whether s names a known style.
func Valid(s string) bool {
	_, ok := lookup(Style(s))
	return ok
}

// For returns the recipe for a style. An unknown name (which the API never
// stores) falls back to the default style, so a clip can always be made.
func For(s string) Recipe {
	if r, ok := lookup(Style(s)); ok {
		return r
	}
	r, _ := lookup(DefaultStyle)
	return r
}

func lookup(s Style) (Recipe, bool) {
	for _, r := range recipes {
		if r.Style == s {
			return r, true
		}
	}
	return Recipe{}, false
}

// Compose returns the prompt to send: the clip's own prompt plus this style's
// wording for the kind of clip (hasImage = it starts from a picture). With no
// suffix the prompt comes back unchanged.
func (r Recipe) Compose(prompt string, hasImage bool) string {
	suffix := r.TextSuffix
	if hasImage {
		suffix = r.ImageSuffix
	}
	if suffix == "" {
		return prompt
	}
	return prompt + "\n\n" + suffix
}
