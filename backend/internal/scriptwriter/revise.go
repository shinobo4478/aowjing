package scriptwriter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

// Revising one shot: the owner sees a drafted shot that does not show what he
// meant (for example the stomach icon beside the character instead of on it) and
// says in a sentence what he wants. The same LLM route rewrites the shot's three
// prompts under the studio's rules. The result is only suggested: the screen puts
// it into the shot's boxes and the owner saves it (or not).

const (
	minInstructionRunes = 3
	maxInstructionRunes = 600
)

// ShotText is the part of a shot the model sees.
type ShotText struct {
	PicturePrompt    string // English, as stored (it may start with the scene sentence)
	EndPicturePrompt string
	MotionPrompt     string
	NarrationTh      string
}

// ReviseRequest is one revision.
type ReviseRequest struct {
	// Instruction is what the owner wants, in his own words (Thai or English).
	Instruction string
	// Brief carries what the profile says about itself (mode, style, cast, notes);
	// its Topic, Shots and Seconds are not used.
	Brief Brief
	Shot  ShotText
	// Previous and Next are the neighbouring shots' starting pictures, given for
	// context only ("" at the ends of the video).
	Previous, Next string
	// HasNext: a shot that is not the last may chain into the next one with an end picture.
	HasNext bool
}

// Revision is the rewritten shot and what it cost.
type Revision struct {
	PicturePrompt, PicturePromptTh       string
	EndPicturePrompt, EndPicturePromptTh string
	MotionPrompt, MotionPromptTh         string
	CostUSD                              float64
	Model                                string
}

// Validate checks the instruction and returns a message per field.
func (r *ReviseRequest) Validate() map[string]string {
	errs := map[string]string{}
	r.Instruction = strings.TrimSpace(r.Instruction)
	if n := utf8.RuneCountInString(r.Instruction); n < minInstructionRunes {
		errs["instruction"] = "Say what the shot should show (at least 3 characters)."
	} else if n > maxInstructionRunes {
		errs["instruction"] = "The instruction must be 600 characters or fewer."
	}
	return errs
}

// reviseTask is added after the studio's standing rules (systemFor), which are
// written for a whole video: this says what is different when one shot is rewritten.
const reviseTask = `

THIS TIME THE TASK IS DIFFERENT. You are not writing a whole video: you REWRITE ONE EXISTING SHOT, following the owner's instruction. Everything above still applies to the prompts you write, except the JSON shape. Reply with ONE JSON object only, no "scene", no "shots":
{"picturePrompt": string, "picturePromptTh": string, "endPicturePrompt": string, "endPicturePromptTh": string, "motionPrompt": string, "motionPromptTh": string}
- The owner's instruction decides what the shot shows. Follow it exactly; change only what it needs and keep the rest of the current prompts (the same character noun, framing, style of wording).
- "picturePrompt" must be COMPLETE: it also states the background. The current text may start with a scene sentence that was copied in front of every shot; keep its background part (the place, the light), but drop anything in it that is an object, an icon or a negative that does not belong to THIS shot's picture, and never leave two things both "in the centre".
- The picture is the moment BEFORE the action; what changes goes in "motionPrompt" and, only if the next shot carries straight on from this one, in "endPicturePrompt" (starting "Only the ..."). Otherwise "endPicturePrompt" and "endPicturePromptTh" are "".
- The narration is what the narrator says over this shot: the picture illustrates it. Never change or repeat it in the prompts.
- The neighbouring shots are shown for context only: do not rewrite them, and keep this shot consistent with them (same character, same drawing style).
- English fields hold no Thai letters; every non-empty English field has its Thai twin, a faithful natural Thai rendering.`

func reviseSystem(b Brief) string { return systemFor(b) + reviseTask }

// revisePrompt is the user message of a revision. It starts with "Revise this one
// shot." which is also how the fake model (local dev) recognises it.
func revisePrompt(r ReviseRequest) string {
	var s strings.Builder
	s.WriteString("Revise this one shot.\n\n")
	fmt.Fprintf(&s, "The owner's instruction (any language): %s\n\n", r.Instruction)
	s.WriteString("The shot now:\n")
	fmt.Fprintf(&s, "- picturePrompt: %s\n", r.Shot.PicturePrompt)
	fmt.Fprintf(&s, "- endPicturePrompt: %s\n", orNone(r.Shot.EndPicturePrompt))
	fmt.Fprintf(&s, "- motionPrompt: %s\n", r.Shot.MotionPrompt)
	if n := strings.TrimSpace(r.Shot.NarrationTh); n != "" {
		fmt.Fprintf(&s, "- narration (Thai, spoken over the shot): %s\n", n)
	}
	if r.Previous != "" {
		fmt.Fprintf(&s, "\nThe previous shot's starting picture (context only): %s\n", r.Previous)
	}
	if r.Next != "" {
		fmt.Fprintf(&s, "The next shot's starting picture (context only): %s\n", r.Next)
	}
	if !r.HasNext {
		s.WriteString("\nThis is the LAST shot: it has no end picture.\n")
	}
	if r.Brief.ProfileName != "" || r.Brief.Niche != "" || r.Brief.Description != "" {
		s.WriteString("\nThe channel this is for:\n")
		if r.Brief.ProfileName != "" {
			fmt.Fprintf(&s, "- Name: %s\n", r.Brief.ProfileName)
		}
		if r.Brief.Niche != "" {
			fmt.Fprintf(&s, "- Niche: %s\n", r.Brief.Niche)
		}
		if r.Brief.Description != "" {
			fmt.Fprintf(&s, "- Notes (tone, audience, do and don't): %s\n", r.Brief.Description)
		}
	}
	s.WriteString(userCastBlock(r.Brief.Cast))
	s.WriteString("\nReply with the JSON only.")
	return s.String()
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

// parseRevision reads the model's reply. A usable one has a picture prompt and a
// motion prompt, no Thai letters in any English field, and a Thai twin for every
// English text; an end picture is dropped when the shot has no next shot.
func parseRevision(raw string, hasNext bool) (Revision, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return Revision{}, errors.New("the model did not answer with JSON")
	}
	var r struct {
		PicturePrompt      string `json:"picturePrompt"`
		PicturePromptTh    string `json:"picturePromptTh"`
		EndPicturePrompt   string `json:"endPicturePrompt"`
		EndPicturePromptTh string `json:"endPicturePromptTh"`
		MotionPrompt       string `json:"motionPrompt"`
		MotionPromptTh     string `json:"motionPromptTh"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return Revision{}, fmt.Errorf("the model's JSON could not be read: %w", err)
	}
	out := Revision{
		PicturePrompt:      strings.TrimSpace(r.PicturePrompt),
		PicturePromptTh:    strings.TrimSpace(r.PicturePromptTh),
		EndPicturePrompt:   strings.TrimSpace(r.EndPicturePrompt),
		EndPicturePromptTh: strings.TrimSpace(r.EndPicturePromptTh),
		MotionPrompt:       strings.TrimSpace(r.MotionPrompt),
		MotionPromptTh:     strings.TrimSpace(r.MotionPromptTh),
	}
	if !hasNext {
		out.EndPicturePrompt, out.EndPicturePromptTh = "", ""
	}
	if out.PicturePrompt == "" || out.MotionPrompt == "" {
		return Revision{}, errors.New("the picture prompt or the motion prompt is empty")
	}
	for name, en := range map[string]string{"picturePrompt": out.PicturePrompt, "endPicturePrompt": out.EndPicturePrompt, "motionPrompt": out.MotionPrompt} {
		if hasThai(en) {
			return Revision{}, fmt.Errorf("%s contains Thai letters", name)
		}
		if utf8.RuneCountInString(en) > maxPromptRunes {
			return Revision{}, fmt.Errorf("%s is too long", name)
		}
	}
	for name, p := range map[string][2]string{
		"picturePromptTh":    {out.PicturePrompt, out.PicturePromptTh},
		"endPicturePromptTh": {out.EndPicturePrompt, out.EndPicturePromptTh},
		"motionPromptTh":     {out.MotionPrompt, out.MotionPromptTh},
	} {
		if p[0] != "" && !hasThai(p[1]) {
			return Revision{}, fmt.Errorf("%s has no Thai text", name)
		}
		if p[0] == "" {
			continue
		}
		if utf8.RuneCountInString(p[1]) > maxPromptRunes {
			return Revision{}, fmt.Errorf("%s is too long", name)
		}
	}
	return out, nil
}

// reviseFrom talks to the model: ask, parse, and ask once more (told what was
// wrong) if the reply cannot be used. Both costs are added. Separate from Revise
// so it can be tested with a scripted model and no database.
func reviseFrom(ctx context.Context, llm generate.LLM, model string, r ReviseRequest) (Revision, error) {
	req := generate.LLMRequest{
		Model:       model,
		System:      reviseSystem(r.Brief),
		Prompt:      revisePrompt(r),
		MaxTokens:   3000, // one shot with its Thai twins
		Temperature: 0.6,
	}
	var spent float64
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, err := llm.Complete(ctx, req)
		if err != nil {
			return Revision{}, err
		}
		spent += res.CostUSD
		rev, perr := parseRevision(res.Text, r.HasNext)
		if perr == nil {
			rev.CostUSD, rev.Model = spent, model
			return rev, nil
		}
		lastErr = perr
		req.Prompt = revisePrompt(r) + "\n\nYour previous reply could not be used (" + perr.Error() +
			"). Reply again with the JSON only."
	}
	return Revision{}, fmt.Errorf("the revision could not be used after two tries: %w", lastErr)
}

// Revise rewrites one shot. The model is the script model (setting scriptModel),
// else the default: the settings are read on each call.
func (w *Writer) Revise(ctx context.Context, r ReviseRequest) (Revision, error) {
	llm, err := w.llm(ctx)
	if err != nil {
		return Revision{}, err
	}
	model, err := w.setting(ctx, "scriptModel")
	if err != nil {
		return Revision{}, err
	}
	if model == "" {
		model = DefaultModel
	}
	return reviseFrom(ctx, llm, model, r)
}
