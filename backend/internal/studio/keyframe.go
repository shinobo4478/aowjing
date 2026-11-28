package studio

import (
	"strconv"
	"strings"
)

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

// CastRole says what a cast member is in a picture (profile Look tab).
type CastRole string

const (
	// RoleHero is the lead character: the same person or creature in every clip.
	RoleHero CastRole = "hero"
	// RoleCostar is another character (a person, an animal) that appears with it.
	RoleCostar CastRole = "costar"
	// RoleProp is an object that must look exactly like its pictures.
	RoleProp CastRole = "prop"
	// RoleProduct is a catalog product (its pictures live on the Product, not in
	// a profile's cast): the one thing that must look exactly like the real one.
	// Not a valid cast_members role, see ValidCastRole.
	RoleProduct CastRole = "product"
	// RolePlace is a location to keep.
	RolePlace CastRole = "place"
	// RoleStyle is an example frame of the profile's drawing style: the picture
	// model copies its line work, colours and way of drawing, never its content.
	RoleStyle CastRole = "style"
)

// ValidCastRole reports whether s is a known role.
func ValidCastRole(s string) bool {
	switch CastRole(s) {
	case RoleHero, RoleCostar, RoleProp, RolePlace, RoleStyle:
		return true
	}
	return false
}

const (
	// MaxCastMembers is how many cast members a profile may keep.
	MaxCastMembers = 8
	// MaxMemberPictures is how many pictures one member may have (other angles
	// of the same character or object).
	MaxMemberPictures = 4
	// MaxProductPictures is how many pictures one product may have (its angles and
	// states: cap closed, cap open, in a hand).
	MaxProductPictures = 4
	// MaxPictureProducts is how many different products one picture may show,
	// the same cap as the products pinned to a clip.
	MaxPictureProducts = 2
	// MaxRequestPictures is how many reference pictures one picture request may
	// carry in all (scene pictures plus the ticked cast). A cautious cap: the
	// models' own limits are higher but more pictures blur who is who, and no
	// request with more than this has been tried. Raise it after a real test.
	MaxRequestPictures = 6
)

// CastRef is one ticked cast member as the prompt describes it. Pictures is how
// many of its pictures go along (they follow one another in the request).
type CastRef struct {
	Role     CastRole
	Name     string
	Note     string
	Pictures int
}

// rolePhrase is how a role reads in the prompt.
func rolePhrase(r CastRole) string {
	switch r {
	case RoleHero:
		return "the lead character"
	case RoleCostar:
		return "another character"
	case RoleProp:
		return "an object"
	case RoleProduct:
		return "the product"
	case RolePlace:
		return "the location"
	case RoleStyle:
		return "the style reference (copy only its drawing style, line weight and colours, never its content)"
	}
	return "a subject"
}

// MaxSceneReferences is how many pictures of the same scene (other shots of a
// video) one request may carry, on top of the cast's pictures.
const MaxSceneReferences = 2

// keepSceneInstruction is for a picture that must show the SAME scene as other
// shots from another angle or moment: reuse the room, lighting and character,
// change only what the description says. A starting guess, not tested yet.
const keepSceneInstruction = "Keep the same scene as in the first reference picture(s): the same room, furniture, props, lighting, colors and character. Change only what is described below, such as the camera angle or what the character is doing."

// keepSceneSameCameraInstruction is for an END picture: the clip flows from the
// starting picture to it (fal's end_image_url), and the model morphs anything
// that differs, so everything but the subject's pose must stay identical. This
// is the wording that made a clean end picture for the owner's first test; the
// first try without "same camera" made the rug shrink and a stool vanish
// during the clip.
const keepSceneSameCameraInstruction = "Use the first reference picture as the exact starting frame: the same camera position, framing, lens, room layout, furniture, rug, props and lighting; nothing in the background changes. Change only what is described below, such as the character's pose or what it is doing."

// keepCastInstruction closes the description of the cast pictures: each member
// must come out as in its own pictures and must not borrow another's looks. A
// starting guess; whether the models hold a product's logo is being tested.
const keepCastInstruction = "Every character, object, product or place named above must look exactly as in its own picture(s): the same face, hair, body, clothes, colours, shape, materials, logo and text in the same place, none redrawn or restyled, and never mixing one's looks into another."

// keepStyleInstruction closes the description when a style reference is among
// the pictures: draw the new picture the way that frame is drawn.
const keepStyleInstruction = "Draw the new picture in exactly the drawing style of the style reference: the same outline weight and colour, the same flat colour fills and palette, and the same level of simplicity."

// castInstruction names the cast pictures, which come after sceneRefs scene
// pictures: "Picture 3 is "Name", the lead character (note). Pictures 4-5 are …".
// Without it the image model gets an unordered pile and cannot tell who is who.
func castInstruction(sceneRefs int, cast []CastRef) string {
	var parts []string
	next := sceneRefs + 1
	hasStyle, hasSubject := false, false
	for _, c := range cast {
		if c.Pictures < 1 {
			continue
		}
		if c.Role == RoleStyle {
			hasStyle = true
		} else {
			hasSubject = true
		}
		var which, verb string
		if c.Pictures == 1 {
			which, verb = "Picture "+strconv.Itoa(next), "is"
		} else {
			which, verb = "Pictures "+strconv.Itoa(next)+"-"+strconv.Itoa(next+c.Pictures-1), "are"
		}
		next += c.Pictures
		s := which + " " + verb + ` "` + strings.ReplaceAll(c.Name, `"`, "'") + `", ` + rolePhrase(c.Role)
		if n := strings.TrimSpace(c.Note); n != "" {
			s += " (" + n + ")"
		}
		parts = append(parts, s+".")
	}
	if len(parts) == 0 {
		return ""
	}
	out := strings.Join(parts, " ")
	if hasSubject {
		out += " " + keepCastInstruction
	}
	if hasStyle {
		out += " " + keepStyleInstruction
	}
	return out
}

// KeyframePrompt is the prompt to send for a keyframe: the owner's scene
// description, instructions to reuse the reference pictures when there are any,
// and this style's wording (so the picture already has the look the clip should
// keep). sceneRefs pictures of the same scene come first among the references,
// then the ticked cast's, each member's pictures together; the cast is named
// picture by picture. sameCamera (with scene pictures) swaps in the stricter
// wording that keeps the camera and everything but the pose fixed, for an end
// picture.
func (r Recipe) KeyframePrompt(prompt string, sceneRefs int, cast []CastRef, sameCamera bool) string {
	out := prompt
	if c := castInstruction(sceneRefs, cast); c != "" {
		out = c + " " + out
	}
	switch {
	case sceneRefs > 0 && sameCamera:
		out = keepSceneSameCameraInstruction + " " + out
	case sceneRefs > 0:
		out = keepSceneInstruction + " " + out
	}
	if r.KeyframeSuffix != "" {
		out += "\n\n" + r.KeyframeSuffix
	}
	return out
}
