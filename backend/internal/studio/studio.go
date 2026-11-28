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

	// StyleRealistic is photographic footage. Its first use is product clips
	// (a Shopee product that must look exactly like the real one); a short-film
	// mode comes later and may split this recipe.
	StyleRealistic Style = "realistic"

	// Style2D is the flat vector cartoon of the owner's educational clips: one
	// look (line, colour, way of telling) shared by every 2D profile.
	Style2D Style = "2d"

	// DefaultStyle is what a profile gets when none is given.
	DefaultStyle = Style3D
)

// Clip lengths in seconds: the widest range any model takes (Kling v3: 3 to 15).
// Each model narrows it (see VideoModel.Seconds: Veo takes 4, 6 or 8), so the
// per-model check is VideoModel.ValidSeconds; this is the outer bound.
const (
	MinClipSeconds     = 3
	MaxClipSeconds     = 15
	DefaultClipSeconds = 5
)

// ValidClipSeconds reports whether n is a length any model could accept.
func ValidClipSeconds(n int) bool { return n >= MinClipSeconds && n <= MaxClipSeconds }

// Recipe is everything a style implies for the wording of a clip. Which model
// makes it is a separate choice (models.go).
type Recipe struct {
	Style Style
	// Label is the name shown to the owner.
	Label string

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

		// Starting wording, not tested against real clips yet: adjust it after
		// seeing what Kling makes of it.
		TextSuffix:     "3D animated film style, Pixar-like characters, soft cinematic lighting, smooth natural motion.",
		KeyframeSuffix: "3D animated film style, Pixar-like characters, soft cinematic lighting.",
	},
	{
		Style: StyleRealistic,
		Label: "Realistic (product / live action)",

		// Photographic, not cinematic ("cinematic" reads as film and fake).
		// A starting guess, not tried on real output yet. ImageSuffix stays
		// empty so a real photo is never restyled.
		TextSuffix:     "Photorealistic, natural light, shot on a phone camera, realistic motion.",
		KeyframeSuffix: "Photorealistic photograph, natural light, realistic materials and skin, no illustration, no 3D render look.",
	},
	{
		Style: Style2D,
		Label: "2D flat vector (educational)",

		// ONE look for every 2D profile, so this is the single place that defines
		// the line, the colour and the way of drawing; change it here and every
		// profile follows. A first guess, not tried on real output yet: AI pictures
		// drift toward 3D shading and clips tend to add depth, so the wording is
		// strict, and a profile's "style" cast picture (an example frame) pins the
		// look harder than words can. The palette is the owner's to replace.
		// Unlike realistic, ImageSuffix is NOT empty: a clip made from a flat
		// picture still needs telling to stay flat and to move like cut-outs.
		TextSuffix:     "Flat 2D vector animation in one consistent style: " + flatLook + " Shapes slide, bounce and scale smoothly like cut-out animation.",
		ImageSuffix:    "Flat 2D vector animation: shapes slide, bounce and scale smoothly like cut-out animation. Keep the drawing flat, with the same outlines and solid colours; no 3D rotation, no shading, no change of perspective.",
		KeyframeSuffix: "Flat 2D vector illustration in one consistent style: " + flatLook,
	},
}

// flatLook is the drawing rules of the 2D style: line, colour, shapes, background.
const flatLook = "uniform medium-weight dark navy outlines with rounded ends, solid flat colour fills only, a limited palette (cream background, navy, teal, coral, sunny yellow, soft purple), matte and two-dimensional, simple geometric shapes, a plain uncluttered background, no gradients, no shading, no 3D look. No text or letters anywhere in the picture."

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
