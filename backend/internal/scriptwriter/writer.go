package scriptwriter

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

const (
	// DefaultModel is used when the owner has not set one in Settings. It is one
	// of the ids fal's openrouter/router page lists as an example; the id is
	// checked by fal at the first call, so a wrong one shows up as a clear error
	// and can be changed in Settings (setting scriptModel) without a release.
	DefaultModel = "anthropic/claude-sonnet-5"

	// MaxShots mirrors videos.MaxShots (a video has at most 12 shots).
	MaxShots = 12

	minTopicRunes = 3
	maxTopicRunes = 1000
)

// Writer drafts shot lists. It reads the fal key and the chosen model from the
// settings store on each call (they may have just been changed).
type Writer struct {
	q       *sqlc.Queries
	fakeFal bool
}

func New(q *sqlc.Queries, fakeFal bool) *Writer { return &Writer{q: q, fakeFal: fakeFal} }

// Validate checks a brief and returns a message per field (empty = valid). The
// handler shows these next to the form fields.
func (b *Brief) Validate() map[string]string {
	errs := map[string]string{}
	b.Topic = strings.TrimSpace(b.Topic)
	if n := len([]rune(b.Topic)); n < minTopicRunes {
		errs["topic"] = "Describe what the video is about (at least 3 characters)."
	} else if n > maxTopicRunes {
		errs["topic"] = "The topic must be 1000 characters or fewer."
	}
	if b.Shots < 1 || b.Shots > MaxShots {
		errs["shots"] = "Choose 1 to 12 shots."
	}
	if !studio.ValidClipSeconds(b.Seconds) {
		errs["seconds"] = "A shot is 3 to 15 seconds long."
	}
	if !ValidMode(string(b.Mode)) {
		errs["mode"] = "Choose general, product, film or explainer."
	}
	if b.Mode == ModeProduct && b.Product == nil {
		errs["productId"] = "Choose the product this video is about."
	}
	return errs
}

// llm picks the provider: the fake one in local dev without a key
// (AI_FAKE_FAL=1), else fal with the saved key.
func (w *Writer) llm(ctx context.Context) (generate.LLM, error) {
	if w.fakeFal {
		return fakeLLM{}, nil
	}
	key, err := w.setting(ctx, "falApiKey")
	if err != nil {
		return nil, err
	}
	return generate.NewFalLLM(key), nil // reports a missing key itself
}

func (w *Writer) setting(ctx context.Context, name string) (string, error) {
	v, err := w.q.GetSetting(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return strings.TrimSpace(v), err
}

// Draft writes the shot list for a brief. If the model's reply cannot be used
// (not JSON, too few shots) it is asked once more, told what was wrong; each
// call costs, and both costs are added into the result. A provider failure is
// returned at once without a retry.
func (w *Writer) Draft(ctx context.Context, b Brief) (Draft, error) {
	llm, err := w.llm(ctx)
	if err != nil {
		return Draft{}, err
	}
	model, err := w.setting(ctx, "scriptModel")
	if err != nil {
		return Draft{}, err
	}
	if model == "" {
		model = DefaultModel
	}

	return draftFrom(ctx, llm, model, b)
}

// draftFrom is the part of Draft that talks to the model: ask, parse, and ask
// once more if the reply is unusable. It is separate so it can be tested with a
// scripted model and no database.
func draftFrom(ctx context.Context, llm generate.LLM, model string, b Brief) (Draft, error) {
	req := generate.LLMRequest{
		Model:       model,
		System:      systemFor(b),
		Prompt:      userPrompt(b),
		MaxTokens:   1000 + 900*b.Shots, // room for the JSON with its Thai twins (Thai costs more tokens)
		Temperature: 0.8,
	}
	opt := parseOptions{}
	if b.Mode == ModeExplainer {
		req.MaxTokens += 300 * b.Shots // the narration of every shot, in Thai
		opt = parseOptions{Narration: true, SecondsOptions: b.SecondsOptions}
	}
	var spent float64
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, err := llm.Complete(ctx, req)
		if err != nil {
			return Draft{}, err
		}
		spent += res.CostUSD
		d, perr := parseWith(res.Text, b.Shots, b.Seconds, opt)
		if perr == nil {
			d.CostUSD, d.Model = spent, model
			return d, nil
		}
		lastErr = perr
		req.Prompt = userPrompt(b) + "\n\nYour previous reply could not be used (" + perr.Error() +
			"). Reply again with the JSON only, with exactly " + strconv.Itoa(b.Shots) + " shots."
	}
	return Draft{}, fmt.Errorf("the draft could not be used after two tries: %w", lastErr)
}

// fakeLLM stands in for fal during local dev without a key: it answers with a
// well-formed draft of the number of shots asked, so the whole flow can be tried
// at no cost.
type fakeLLM struct{}

var shotsLine = regexp.MustCompile(`Number of shots: (\d+)`)

func (fakeLLM) Complete(_ context.Context, req generate.LLMRequest) (generate.LLMResult, error) {
	// A translation (see translate.go): answer in the language asked for, so the
	// checks that a translation holds (no Thai in English, some Thai in Thai) pass.
	if strings.HasPrefix(req.Prompt, "Direction: toEnglish") {
		return generate.LLMResult{Text: "Fake English translation."}, nil
	}
	if strings.HasPrefix(req.Prompt, "Direction: toThai") {
		return generate.LLMResult{Text: "คำแปลภาษาไทยจำลอง"}, nil
	}

	// A revision of one shot (see revise.go).
	if strings.HasPrefix(req.Prompt, "Revise this one shot.") {
		return generate.LLMResult{Text: `{"picturePrompt":"Fake revised picture prompt: a flat character stands on a flat scale, a stomach icon drawn on its belly.","picturePromptTh":"พรอมต์ภาพที่แก้แล้วจำลอง: ตัวการ์ตูนยืนบนตาชั่ง มีไอคอนกระเพาะที่ท้อง","endPicturePrompt":"","endPicturePromptTh":"","motionPrompt":"The food icon slides out along an arrow. Static camera, no zoom.","motionPromptTh":"ไอคอนอาหารเลื่อนออกตามลูกศร กล้องนิ่ง ไม่ซูม"}`}, nil
	}

	n := 3
	if m := shotsLine.FindStringSubmatch(req.Prompt); m != nil {
		n, _ = strconv.Atoi(m[1])
	}
	var shots []string
	for i := 1; i <= n; i++ {
		// Shot 1 chains into shot 2 (it has an end prompt); the rest cut.
		end, endTh := "", ""
		if i == 1 && n > 1 {
			end = "Only the fake cat's pose changes in fake shot 1."
			endTh = "เปลี่ยนเฉพาะท่าของแมวจำลองในช็อตจำลองที่ 1"
		}
		// The narration only matters in explainer mode (the parser ignores it
		// otherwise). Each has a different length, so the shots get different
		// lengths; it starts with a Thai word and holds no Latin letters.
		narration := "ช็อตจำลองนี้อธิบายเรื่องหนึ่งอย่างสั้นๆ" + strings.Repeat(" และอธิบายต่อไปอีกนิด", i%3)
		shots = append(shots, fmt.Sprintf(
			`{"picturePrompt":"Fake picture prompt for shot %d.","picturePromptTh":"พรอมต์ภาพจำลองของช็อต %d","endPicturePrompt":"%s","endPicturePromptTh":"%s","motionPrompt":"Fake motion for shot %d. Static camera, no zoom.","motionPromptTh":"การเคลื่อนไหวจำลองของช็อต %d กล้องนิ่ง ไม่ซูม","narrationTh":"%s"}`,
			i, i, end, endTh, i, i, narration))
	}
	text := `{"title":"Fake draft","scene":"A fake sunny room with a sofa and a window.","sceneTh":"ห้องจำลองที่แดดส่อง มีโซฟาและหน้าต่าง","shots":[` + strings.Join(shots, ",") + `]}`
	return generate.LLMResult{Text: text}, nil
}
