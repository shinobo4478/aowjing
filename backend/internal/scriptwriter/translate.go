package scriptwriter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

// The owner reads and writes Thai; the picture and clip models are only promised
// to understand English. So each prompt of a shot exists twice: the Thai the
// owner edits, and the English the models receive. Translate makes one from the
// other with the same LLM route the draft uses (fal's openrouter/router, billed
// per token on the fal account).

// Kind says what a prompt is for, because each kind has its own rules: a clip
// prompt describes only movement, a picture prompt no style words, an end
// picture prompt only what changes.
type Kind string

const (
	KindMotion     Kind = "motion"
	KindPicture    Kind = "picture"
	KindEndPicture Kind = "endPicture"
)

// Direction is which way to translate.
type Direction string

const (
	ToEnglish Direction = "toEnglish"
	ToThai    Direction = "toThai"
)

const maxTranslateRunes = 2000

// TranslateRequest is one translation.
type TranslateRequest struct {
	Kind      Kind
	Direction Direction
	Text      string
}

// Translation is the result and what it cost.
type Translation struct {
	Text    string
	CostUSD float64
	Model   string
}

// Validate checks the request and returns a message per field.
func (r *TranslateRequest) Validate() map[string]string {
	errs := map[string]string{}
	r.Text = strings.TrimSpace(r.Text)
	switch r.Kind {
	case KindMotion, KindPicture, KindEndPicture:
	default:
		errs["kind"] = "Unknown prompt kind."
	}
	switch r.Direction {
	case ToEnglish, ToThai:
	default:
		errs["direction"] = "Unknown direction."
	}
	if r.Text == "" {
		errs["text"] = "There is nothing to translate."
	} else if len([]rune(r.Text)) > maxTranslateRunes {
		errs["text"] = "The text must be 2000 characters or fewer."
	}
	return errs
}

// hasThai reports whether s contains Thai letters. It is how a translation to
// English is checked (it must hold none) and how the screen can tell that an
// English box holds Thai by mistake.
func hasThai(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Thai) {
			return true
		}
	}
	return false
}

// toEnglishSystem is the standing instruction for Thai -> English. It is a
// translation, not a rewrite: the owner decides what happens in the shot.
const toEnglishSystem = `You translate a short Thai note into an English prompt for an AI picture or video model. Reply with the English prompt ONLY: no quotes, no notes, no explanation, no Thai.

General rules:
- Keep every detail the Thai says, and add nothing the Thai does not say: do not invent objects, actions, emotions or places.
- Plain, concrete English, in the order the Thai gives.
- Refer to the main character with the same simple noun the Thai uses (for example "the cat").
- If the Thai says an exact word in quotes or a name, keep it as it is.`

// kindRules adds what is specific to each kind of prompt (Thai -> English).
var kindRules = map[Kind]string{
	KindMotion:     `This is a CLIP MOTION prompt for an image-to-video model: the picture already fixes how everything looks, so write ONLY what moves and how the camera moves, in order, and how the clip ends. Never describe appearance (colour, breed, clothes, style). Keep it to one to three sentences. If the Thai does not mention the camera, end with "Static camera, no zoom."`,
	KindPicture:    `This is a PICTURE prompt for an image model: write what the still picture shows at its start: who or what is in it, what they are doing, the framing (close-up, medium, wide) and the mood. Never use a style word (no "3D", "cartoon", "animated", "rendered", "Pixar", "illustration", "realistic", "photo"). Never add text, signs or logos to the picture. Never describe the character's looks (breed, colours, face shape, fur): a reference picture fixes that.`,
	KindEndPicture: `This is an END PICTURE prompt: the clip flows from the starting picture to an end picture, and anything that differs between the two gets morphed, so this prompt says ONLY what changes between them: a pose or a position. Start with "Only the ..." and name just what changes. Never mention a change of camera, room or lighting.`,
}

// toThaiSystem is for English -> Thai: a faithful explanation, so a reader who
// is weaker in English can check what a prompt says.
const toThaiSystem = `You translate an English prompt for an AI picture or video model into clear, natural Thai, so that a reader who does not read English understands exactly what the prompt asks for. Reply with the Thai ONLY: no quotes, no notes, no English explanation.

- Keep every concrete detail, in the same order; add and drop nothing.
- Keep the instructions as instructions (for example "static camera, no zoom" becomes "กล้องนิ่ง ไม่ซูม").
- Plain everyday Thai, not stiff or literal.`

func translateSystem(r TranslateRequest) string {
	if r.Direction == ToThai {
		return toThaiSystem
	}
	return toEnglishSystem + "\n\n" + kindRules[r.Kind]
}

// translateFrom is the part that talks to the model: ask, check the answer, and
// ask once more (told what was wrong) if a translation to English still holds
// Thai or one to Thai holds none. Both costs are added. It is separate from
// Translate so it can be tested with a scripted model and no database.
func translateFrom(ctx context.Context, llm generate.LLM, model string, r TranslateRequest) (Translation, error) {
	req := generate.LLMRequest{
		Model:       model,
		System:      translateSystem(r),
		Prompt:      "Direction: " + string(r.Direction) + "\n\n" + r.Text,
		MaxTokens:   1500,
		Temperature: 0.2, // a translation should not wander
	}
	var spent float64
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, err := llm.Complete(ctx, req)
		if err != nil {
			return Translation{}, err
		}
		spent += res.CostUSD
		out := strings.Trim(strings.TrimSpace(res.Text), "\"“”")
		out = strings.TrimSpace(out)
		switch {
		case out == "":
			lastErr = errors.New("the model returned nothing")
		case r.Direction == ToEnglish && hasThai(out):
			lastErr = errors.New("the English still contains Thai letters")
		case r.Direction == ToThai && !hasThai(out):
			lastErr = errors.New("the Thai has no Thai letters")
		case len([]rune(out)) > maxTranslateRunes:
			lastErr = errors.New("the translation is too long")
		default:
			return Translation{Text: out, CostUSD: spent, Model: model}, nil
		}
		req.Prompt = "Direction: " + string(r.Direction) + "\n\n" + r.Text +
			"\n\nYour previous reply was refused (" + lastErr.Error() + "). Reply again with the translation only."
	}
	return Translation{}, fmt.Errorf("the translation could not be used after two tries: %w", lastErr)
}

// Translate translates one prompt. The model is the owner's translation model
// (setting translateModel), else the script model (scriptModel), else the
// default: the settings are read on each call.
func (w *Writer) Translate(ctx context.Context, r TranslateRequest) (Translation, error) {
	llm, err := w.llm(ctx)
	if err != nil {
		return Translation{}, err
	}
	model, err := w.setting(ctx, "translateModel")
	if err != nil {
		return Translation{}, err
	}
	if model == "" {
		if model, err = w.setting(ctx, "scriptModel"); err != nil {
			return Translation{}, err
		}
	}
	if model == "" {
		model = DefaultModel
	}
	return translateFrom(ctx, llm, model, r)
}
