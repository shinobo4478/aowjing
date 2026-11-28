package scriptwriter

import (
	"fmt"
	"strings"
)

// Mode is what kind of video the owner wants drafted. It changes the story
// structure the model is told to follow; the output shape is the same.
type Mode string

const (
	// ModeGeneral is the original behaviour: a short story from a topic.
	ModeGeneral Mode = "general"
	// ModeProduct is a clip about a real product that must look like the real
	// one: hook, natural use, the message through action, a calm product shot.
	ModeProduct Mode = "product"
	// ModeFilm is a short satirical film told in visual gags (no dialogue; the
	// owner adds Thai captions in the editor).
	ModeFilm Mode = "film"
)

// ValidMode reports whether s names a mode. The empty string is accepted and
// means ModeGeneral.
func ValidMode(s string) bool {
	switch Mode(s) {
	case "", ModeGeneral, ModeProduct, ModeFilm:
		return true
	}
	return false
}

// CastNote is one member of the profile's cast as the model is told about it.
// The reference pictures themselves are not sent to the writer (it only writes
// text); the picture step sends them, and names them there.
type CastNote struct {
	Role string // hero, costar, prop, place
	Name string // as the owner typed it, often Thai
	Note string
}

// ProductNote is the catalog product a product video is about.
type ProductNote struct {
	Name string
	// Note is the owner's own note on the product; the only source of claims the
	// video may make about it.
	Note string
	// States are the labels of the product's pictures ("cap closed", "cap open"):
	// the only states the video may show it in.
	States []string
}

// modeRules is the structure and the do's and don'ts of a mode, appended to the
// standing system prompt. Written as rules the studio learned: the picture models
// cannot hold text, hands or fast actions, and an edit model copies the state
// of its reference.
func modeRules(m Mode) string {
	switch m {
	case ModeProduct:
		return `
FORMAT: PRODUCT VIDEO. The video is about one real product that is described under PRODUCT below. The picture models are given real photos of it, so the product on screen will look like the real one only if you follow these rules:
- Structure: shot 1 is a hook: a relatable problem or moment that a viewer recognises in one second (the product may be absent or small in it). The middle shots show the main character using the product in a natural everyday moment, and the benefit shows through ACTION, never through text or a spoken claim. The LAST shot is a calm, clear shot of the product itself (static camera, nothing else going on), and may lead back into shot 1 so the video loops.
- Refer to the product in English as "the product" plus a plain generic noun for what it is (for example "the bottle"); never use its brand or model name. Never describe how it looks (colour, shape, logo): the reference pictures fix that.
- Show the product ONLY in the states listed under PRODUCT. A shot that needs a state that is not listed must be written so that state is never needed. A shot whose action needs the product in a listed state must say that state in its picturePrompt.
- Show the product from the side or at medium distance. Never ask for a close-up of its label, logo or any writing, and never for hands that turn it around or inspect it.
- Claim nothing about the product beyond the owner's note under PRODUCT: no invented features, numbers, health claims or comparisons.
- Keep every action simple enough for an AI clip: one person, plain movements such as sitting, standing, walking a few steps, picking something up, drinking, looking, smiling.`
	case ModeFilm:
		return `
FORMAT: SHORT FILM WITH A JOKE. A short story with a dry, satirical twist, told only in pictures and action.
- Structure: SETUP (an ordinary moment that shows what the main character wants), ESCALATION (a small obstacle that gets worse), PUNCHLINE (an unexpected visual gag that turns the situation upside down), LAST BEAT (a reaction or an ironic closing image). Spread these over the shots in that order, giving the punchline the room it needs.
- No dialogue and no text: the joke must be understood from the pictures and the action alone (the owner adds captions in the editor). Deadpan humour: the main character plays it straight.
- One location, one main character and at most one other character, so every shot can look like the same film.
- The satire is aimed at behaviour or situations, never at real people, brands or groups of people; nothing harmful or demeaning.
- Keep every action simple enough for an AI clip: plain movements, one clear beat per shot.`
	}
	return ""
}

// realisticRules are added whatever the mode when the profile's style is
// realistic: the pictures are photographs, not drawings.
const realisticRules = `
STYLE: PHOTOGRAPHIC LIVE ACTION. The pictures are photographs of real life, so people and things must behave like real life: believable light, believable physics, ordinary places. Avoid what AI footage gets wrong: fast hand actions, crowds, mirrors, screens, readable text, and people seen from several angles in one shot. Prefer one clear subject and slow, simple movement.`

// castRules tells the writer about the profile's cast: who exists and how to
// refer to each in the English text. The reference pictures fix how they look,
// and the picture step names each picture ("Picture 1 is "Name", the lead
// character"), so the English prompts must use role words, not names: the
// English fields must hold no Thai letters, and a name is often Thai.
const castRules = `
CAST: the channel has these recurring subjects. Reference pictures fix how each one looks, and the studio sends those pictures when it makes a shot's picture, so use them wherever the story needs them (a shot may show any of them, or none).
- In the English fields refer to them ONLY like this: the hero is "the lead character"; another character or a prop is a plain generic noun for what it is, taken from its name or note, never its looks (for example "the cat", "the coach", "the bottle"), and if two would get the same noun add "first" and "second"; a place is "the location". Never write their names in the English fields (names may be Thai and the English must hold no Thai letters), and never describe how they look.
- In the Thai twins, write the names as given.`

// userCastBlock lists the cast for the user prompt.
func userCastBlock(cast []CastNote) string {
	if len(cast) == 0 {
		return ""
	}
	var s strings.Builder
	s.WriteString("\nCAST of this channel:\n")
	for _, c := range cast {
		fmt.Fprintf(&s, "- %s: %q", c.Role, c.Name)
		if n := strings.TrimSpace(c.Note); n != "" {
			fmt.Fprintf(&s, " (%s)", n)
		}
		s.WriteString("\n")
	}
	return s.String()
}

// userProductBlock describes the product for the user prompt.
func userProductBlock(p *ProductNote) string {
	if p == nil {
		return ""
	}
	var s strings.Builder
	s.WriteString("\nPRODUCT:\n")
	fmt.Fprintf(&s, "- Name (for your understanding only; never write it in the English fields): %s\n", p.Name)
	if n := strings.TrimSpace(p.Note); n != "" {
		fmt.Fprintf(&s, "- Owner's note (the only claims the video may make): %s\n", n)
	} else {
		s.WriteString("- Owner's note: none, so make no claims about the product.\n")
	}
	if len(p.States) > 0 {
		var labelled []string
		unlabelled := 0
		for _, st := range p.States {
			if st = strings.TrimSpace(st); st != "" {
				labelled = append(labelled, fmt.Sprintf("%q", st))
			} else {
				unlabelled++
			}
		}
		if len(labelled) > 0 {
			fmt.Fprintf(&s, "- Pictures of the real product exist for these states: %s.\n", strings.Join(labelled, ", "))
		}
		if unlabelled > 0 {
			fmt.Fprintf(&s, "- %d more picture(s) have no state written; assume only the plain, normal look.\n", unlabelled)
		}
	}
	return s.String()
}

// systemFor is the standing instruction for one brief: the base prompt plus the
// rules of the chosen mode, of the profile's style and, when it has a cast, how
// to refer to it.
func systemFor(b Brief) string {
	s := systemPrompt
	if r := modeRules(b.Mode); r != "" {
		s += "\n" + r
	}
	if b.VideoStyle == "realistic" {
		s += "\n" + realisticRules
	}
	if len(b.Cast) > 0 {
		s += "\n" + castRules
	}
	return s
}
