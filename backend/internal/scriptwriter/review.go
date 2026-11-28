package scriptwriter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

// A review is a second look at a drafted video by the same LLM route, told to act
// as a strict reviewer. The checks in lib/promptLint.ts are patterns; these are the
// ones that need understanding (a start picture that already shows the action, an
// icon in the scene sentence that fights a shot's own picture, a narration that
// starts with a loanword written in Thai letters). It only REPORTS: the screen lists
// the findings and offers "ให้ AI แก้ช็อตนี้" with the suggested fix filled in.

// Finding kinds. The picture/motion kinds can be fixed by rewriting the shot's
// prompts (revise.go); the narration kinds cannot, the owner edits the narration.
const (
	KindPictureShowsAction = "picture-shows-action" // the start picture already shows what the motion begins
	KindTooManyBeats       = "too-many-beats"       // more action than the shot's seconds hold
	KindStrayObject        = "stray-object"         // an object, icon or negative in a prompt that fights the shot's own picture
	KindPictureVsNarration = "picture-vs-narration" // the picture does not illustrate what the narrator says
	KindChain              = "chain"                // an end picture does not match the next shot's start
	KindCamera             = "camera"               // contradicting or revealing camera instructions
	KindCharacter          = "character"            // the recurring character is named or drawn differently
	KindNarrationLoanword  = "narration-loanword"   // the narration starts with a loanword written in Thai letters
	KindNarrationClaim     = "narration-claim"      // a claim, number or absolute statement that should be checked
)

var findingKinds = map[string]bool{
	KindPictureShowsAction: true, KindTooManyBeats: true, KindStrayObject: true,
	KindPictureVsNarration: true, KindChain: true, KindCamera: true, KindCharacter: true,
	KindNarrationLoanword: true, KindNarrationClaim: true,
}

// FixableKind reports whether rewriting the shot's prompts can fix a finding of
// this kind (the narration kinds need the owner to edit the narration).
func FixableKind(kind string) bool {
	return kind != KindNarrationLoanword && kind != KindNarrationClaim
}

const maxFindings = 12

// ReviewShot is one shot as the reviewer sees it.
type ReviewShot struct {
	Position         int
	PicturePrompt    string
	EndPicturePrompt string
	MotionPrompt     string
	NarrationTh      string
	Seconds          int
}

// ReviewRequest is one review of a whole video.
type ReviewRequest struct {
	// Brief carries the profile's style, cast and notes (its Topic, Shots and
	// Seconds are not used).
	Brief Brief
	Shots []ReviewShot
}

// Finding is one problem in one shot.
type Finding struct {
	Shot      int    // the shot's position
	Kind      string // one of the Kind constants
	MessageTh string // what is wrong, one sentence in Thai
	FixTh     string // what the shot should do instead, to feed "revise"; "" for narration kinds
}

// Review is the findings and what the call cost.
type Review struct {
	Findings []Finding
	CostUSD  float64
	Model    string
}

// Validate checks the request.
func (r *ReviewRequest) Validate() map[string]string {
	errs := map[string]string{}
	if len(r.Shots) == 0 {
		errs["shots"] = "The video has no shots to review."
	}
	return errs
}

// reviewTask follows the studio's standing rules (systemFor): the reviewer holds
// the drafted shots to those same rules.
const reviewTask = `

THIS TIME THE TASK IS DIFFERENT. You are a strict reviewer of a drafted video. Hold every shot to the rules above and report only REAL problems you are sure of; being quiet when a shot is fine is correct, and never invent a problem to look useful. Reply with ONE JSON object only:
{"findings": [{"shot": number, "kind": string, "messageTh": string, "fixTh": string}]}
and {"findings": []} when nothing is wrong. At most 12 findings, the most important first. "shot" is the shot's number as shown.
"kind" is exactly one of:
- "picture-shows-action": the starting picture already shows what the motion prompt begins (the picture must be the moment BEFORE the action).
- "too-many-beats": more actions than the shot's seconds hold (more than two beats in 4 to 5 seconds).
- "stray-object": a prompt holds an object, icon, symbol or a negative ("nothing else around") that does not belong to this shot's picture and fights it (two things both "in the centre"; a prop from a scene sentence that was copied in front of every shot).
- "picture-vs-narration": the picture does not illustrate what the narrator says over the shot, or shows something the narration contradicts.
- "chain": a shot has an end picture whose state does not match what the next shot's starting picture shows.
- "camera": camera instructions contradict each other (a move together with "static camera"), or a move reveals what is not in the starting picture.
- "character": the recurring character is named or described differently from shot to shot, or described by looks.
- "narration-loanword": the narration starts with a loanword written in Thai letters (the voice takes its accent from the first words); the first words must be Thai words.
- "narration-claim": the narration states a number, a percentage, a study or an absolute health or money claim that the owner should check; also advice phrased as a command.
"messageTh": one short sentence in Thai saying what is wrong, quoting a few words of the text.
"fixTh": for every kind except the two narration kinds, one instruction in Thai, in the owner's own voice, telling a rewriting assistant what this shot's pictures and motion should show instead (concrete: who, where, what moves). For the two narration kinds use "".`

func reviewSystem(b Brief) string { return systemFor(b) + reviewTask }

// reviewPrompt is the user message of a review. It starts with "Review this video."
// which is also how the fake model (local dev) recognises it.
func reviewPrompt(r ReviewRequest) string {
	var s strings.Builder
	s.WriteString("Review this video.\n\n")
	for _, sh := range r.Shots {
		fmt.Fprintf(&s, "Shot %d (%d seconds)\n", sh.Position, sh.Seconds)
		fmt.Fprintf(&s, "- picturePrompt: %s\n", sh.PicturePrompt)
		fmt.Fprintf(&s, "- endPicturePrompt: %s\n", orNone(sh.EndPicturePrompt))
		fmt.Fprintf(&s, "- motionPrompt: %s\n", sh.MotionPrompt)
		if n := strings.TrimSpace(sh.NarrationTh); n != "" {
			fmt.Fprintf(&s, "- narration (Thai): %s\n", n)
		}
		s.WriteString("\n")
	}
	if r.Brief.ProfileName != "" || r.Brief.Niche != "" || r.Brief.Description != "" {
		s.WriteString("The channel this is for:\n")
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

// parseReview reads the model's reply. Findings that cannot be used (a shot that
// does not exist, an unknown kind, no Thai message) are dropped rather than
// failing the whole review; a reply that is not JSON at all is an error.
func parseReview(raw string, shots int) ([]Finding, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, errors.New("the model did not answer with JSON")
	}
	var r struct {
		Findings []struct {
			Shot      int    `json:"shot"`
			Kind      string `json:"kind"`
			MessageTh string `json:"messageTh"`
			FixTh     string `json:"fixTh"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return nil, fmt.Errorf("the model's JSON could not be read: %w", err)
	}
	var out []Finding
	for _, f := range r.Findings {
		msg := strings.TrimSpace(f.MessageTh)
		if f.Shot < 1 || f.Shot > shots || !findingKinds[f.Kind] || !hasThai(msg) || utf8.RuneCountInString(msg) > 600 {
			continue
		}
		fix := strings.TrimSpace(f.FixTh)
		if !FixableKind(f.Kind) || !hasThai(fix) || utf8.RuneCountInString(fix) > maxInstructionRunes {
			fix = ""
		}
		out = append(out, Finding{Shot: f.Shot, Kind: f.Kind, MessageTh: msg, FixTh: fix})
		if len(out) == maxFindings {
			break
		}
	}
	// By shot, so the screen can read them top to bottom; the model's own order
	// (most important first) is kept within a shot.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Shot < out[j].Shot })
	return out, nil
}

// reviewFrom talks to the model: ask, parse, and ask once more if the reply is
// not JSON. Both costs are added. Separate from Review so it can be tested with a
// scripted model and no database.
func reviewFrom(ctx context.Context, llm generate.LLM, model string, r ReviewRequest) (Review, error) {
	req := generate.LLMRequest{
		Model:       model,
		System:      reviewSystem(r.Brief),
		Prompt:      reviewPrompt(r),
		MaxTokens:   4000,
		Temperature: 0.2, // a reviewer should be steady, not creative
	}
	var spent float64
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, err := llm.Complete(ctx, req)
		if err != nil {
			return Review{}, err
		}
		spent += res.CostUSD
		findings, perr := parseReview(res.Text, len(r.Shots))
		if perr == nil {
			return Review{Findings: findings, CostUSD: spent, Model: model}, nil
		}
		lastErr = perr
		req.Prompt = reviewPrompt(r) + "\n\nYour previous reply could not be used (" + perr.Error() + "). Reply again with the JSON only."
	}
	return Review{}, fmt.Errorf("the review could not be used after two tries: %w", lastErr)
}

// Review reviews a drafted video. The model is the script model (setting
// scriptModel), else the default: the settings are read on each call.
func (w *Writer) Review(ctx context.Context, r ReviewRequest) (Review, error) {
	llm, err := w.llm(ctx)
	if err != nil {
		return Review{}, err
	}
	model, err := w.setting(ctx, "scriptModel")
	if err != nil {
		return Review{}, err
	}
	if model == "" {
		model = DefaultModel
	}
	return reviewFrom(ctx, llm, model, r)
}
