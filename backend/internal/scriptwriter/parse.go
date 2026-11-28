package scriptwriter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// maxPromptRunes keeps a drafted prompt inside what a shot accepts (2000), with
// room for the scene line that is put in front of every picture prompt.
const maxPromptRunes = 1900

// maxNarrationRunes mirrors what a shot's narration accepts (studio.MaxVoiceSampleChars).
const maxNarrationRunes = studio.MaxVoiceSampleChars

// Brief is what the owner asks for.
type Brief struct {
	Topic   string
	Shots   int
	Seconds int // the length of each shot

	// Source is material the owner pasted (an article, notes, a script) that the
	// video must be based on: it is the only place facts, numbers and claims may
	// come from. Empty = the model writes from the topic alone.
	Source string

	// What the profile says about itself, so the shots fit its channel.
	ProfileName string
	Niche       string
	Description string

	// Mode is the kind of video (empty = general). VideoStyle is the profile's style
	// name ("3d", "realistic"). Cast is the profile's recurring subjects and Product
	// the catalog product a product video is about; see modes.go.
	Mode       Mode
	VideoStyle string
	Cast       []CastNote
	Product    *ProductNote

	// SecondsOptions are the clip lengths the profile's video model takes. In
	// explainer mode each shot's length is worked out from its narration and
	// rounded up to one of these; Seconds is then the LONGEST a shot may be.
	SecondsOptions []int
}

// parseOptions are the parts of a Brief that decide how a reply is read.
type parseOptions struct {
	// Narration: every shot must carry a usable narrationTh, and its length is
	// worked out from it (explainer mode).
	Narration      bool
	SecondsOptions []int
}

// Shot is one drafted shot, ready to put into a Video.
type Shot struct {
	// PicturePrompt is what THIS shot's picture shows. The scene is kept once on the
	// video (Draft.Scene) and put in front of it when the picture is made.
	PicturePrompt string
	// EndPicturePrompt says what changes by the END picture (only a pose or
	// position), "" when the shot has no end picture because the next shot cuts to
	// a new angle or moment. Never set on the last shot.
	EndPicturePrompt string
	MotionPrompt     string
	Seconds          int

	// The same three prompts in Thai, for the owner to read and edit. A
	// convenience: a missing or unusable one is left empty, not an error, because
	// it can be made later with the translation helper.
	PicturePromptTh    string // the Thai twin of PicturePrompt (no scene)
	EndPicturePromptTh string
	MotionPromptTh     string

	// NarrationTh is what the narrator says over the shot (explainer mode only).
	// Thai letters only; the shot's Seconds is worked out from it.
	NarrationTh string
	// NarrationTooLong is true when the narration does not fit even the longest
	// allowed shot: the screen warns and the owner shortens it.
	NarrationTooLong bool
}

// Draft is a drafted video.
type Draft struct {
	Title   string
	Scene   string
	SceneTh string // the Thai twin of Scene, "" when the model gave none
	Shots   []Shot
	CostUSD float64 // what the LLM call cost, as fal reported it
	Model   string
}

// reply is the JSON shape the model was asked for.
type reply struct {
	Title   string `json:"title"`
	Scene   string `json:"scene"`
	SceneTh string `json:"sceneTh"`
	Shots   []struct {
		PicturePrompt      string `json:"picturePrompt"`
		PicturePromptTh    string `json:"picturePromptTh"`
		EndPicturePrompt   string `json:"endPicturePrompt"`
		EndPicturePromptTh string `json:"endPicturePromptTh"`
		MotionPrompt       string `json:"motionPrompt"`
		MotionPromptTh     string `json:"motionPromptTh"`
		NarrationTh        string `json:"narrationTh"`
	} `json:"shots"`
}

// parse turns the model's text into a Draft. Models often wrap JSON in prose or
// a code fence despite being told not to, so it takes the outermost {...}. It
// refuses a reply with too few shots or an empty prompt rather than filling a
// video with gaps, and drops extra shots. The scene is returned once (Draft.Scene)
// for the video to keep; it is put in front of a picture prompt when the picture
// is made, so the pictures of one video describe the same place.
func parse(raw string, want, seconds int) (Draft, error) {
	return parseWith(raw, want, seconds, parseOptions{})
}

// parseWith is parse with the options of a mode (see parseOptions).
func parseWith(raw string, want, seconds int, opt parseOptions) (Draft, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return Draft{}, errors.New("the model did not answer with JSON")
	}
	var r reply
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return Draft{}, fmt.Errorf("the model's JSON could not be read: %w", err)
	}

	scene := strings.TrimSpace(r.Scene)
	if scene == "" {
		return Draft{}, errors.New("the model's draft has no scene")
	}
	if len(r.Shots) < want {
		return Draft{}, fmt.Errorf("the model wrote %d shots, not the %d asked for", len(r.Shots), want)
	}

	if hasThai(scene) {
		return Draft{}, errors.New("the scene contains Thai letters (it must be English; its Thai twin is sceneTh)")
	}
	d := Draft{Title: strings.TrimSpace(r.Title), Scene: scene, SceneTh: strings.TrimSpace(r.SceneTh)}
	for i := 0; i < want; i++ {
		pic := strings.TrimSpace(r.Shots[i].PicturePrompt)
		motion := strings.TrimSpace(r.Shots[i].MotionPrompt)
		end := strings.TrimSpace(r.Shots[i].EndPicturePrompt)
		if pic == "" || motion == "" {
			return Draft{}, fmt.Errorf("shot %d of the draft is missing a prompt", i+1)
		}
		// Nothing follows the last shot, so an end picture there would have no
		// next shot to meet: drop it rather than keep a note that goes nowhere.
		if i == want-1 {
			end = ""
		}
		// The scene goes in front of the picture prompt when the picture is made, so
		// the two together must still fit what a picture request accepts.
		if utf8.RuneCountInString(scene+" "+pic) > maxPromptRunes ||
			utf8.RuneCountInString(motion) > maxPromptRunes ||
			utf8.RuneCountInString(end) > maxPromptRunes {
			return Draft{}, fmt.Errorf("shot %d of the draft is too long", i+1)
		}
		s := Shot{PicturePrompt: pic, EndPicturePrompt: end, MotionPrompt: motion, Seconds: seconds}

		// A narrated shot: the narration must be speakable Thai, and it (not the
		// brief) sets how long the shot is.
		if opt.Narration {
			narration := strings.TrimSpace(r.Shots[i].NarrationTh)
			if perr := narrationProblem(narration); perr != nil {
				return Draft{}, fmt.Errorf("the narration of shot %d %w", i+1, perr)
			}
			if utf8.RuneCountInString(narration) > maxNarrationRunes {
				return Draft{}, fmt.Errorf("the narration of shot %d is too long", i+1)
			}
			s.NarrationTh = narration
			if secs, fits := studio.SecondsForNarration(narration, opt.SecondsOptions); secs > 0 {
				s.Seconds, s.NarrationTooLong = secs, !fits
			}
		}

		// The Thai versions: kept only if they are usable.
		s.PicturePromptTh = strings.TrimSpace(r.Shots[i].PicturePromptTh)
		s.MotionPromptTh = strings.TrimSpace(r.Shots[i].MotionPromptTh)
		if i != want-1 {
			s.EndPicturePromptTh = strings.TrimSpace(r.Shots[i].EndPicturePromptTh)
		}
		// A Thai prompt with no English twin (a cut shot has no end prompt) is
		// meaningless; one that is over the length limit is dropped, not fatal.
		if end == "" {
			s.EndPicturePromptTh = ""
		}
		for _, th := range []*string{&s.PicturePromptTh, &s.EndPicturePromptTh, &s.MotionPromptTh} {
			if utf8.RuneCountInString(*th) > maxPromptRunes {
				*th = ""
			}
		}
		d.Shots = append(d.Shots, s)
	}
	return d, nil
}
