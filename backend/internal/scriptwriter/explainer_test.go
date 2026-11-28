package scriptwriter

import (
	"context"
	"strings"
	"testing"
)

func TestNarrationProblem(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string // a part of the message; "" = fine
	}{
		{"plain Thai", "การกินน้อยกว่าที่ร่างกายใช้ในแต่ละวัน เรียกว่า แคลอรี่ ดีฟิซิท", ""},
		{"Thai with a leading space", "  คำว่า แคลอรี่ ดีฟิซิท", ""},
		{"empty", "  ", "empty"},
		{"Latin letters", "คำว่า calorie deficit", "Latin"},
		{"Latin first", "TDEE คือพลังงานทั้งวัน", "Latin"},
		{"ASCII digits", "ไขมัน 1 กิโลกรัม", "digits"},
		{"Thai digits", "ไขมัน ๑ กิโลกรัม", "digits"},
		{"starts with a symbol", "…แล้วก็ต่อ", "start with a Thai"},
		{"percent sign only", "%ร้อย", "start with a Thai"},
	}
	for _, tt := range tests {
		err := narrationProblem(tt.text)
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: unexpected problem %v", tt.name, err)
		case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%s: got %v, want a problem mentioning %q", tt.name, err, tt.want)
		}
	}
}

func TestValidModeKnowsExplainer(t *testing.T) {
	if !ValidMode("explainer") {
		t.Error("explainer should be a mode")
	}
}

func TestSystemForExplainerCarriesTheNarrationRules(t *testing.T) {
	s := systemFor(Brief{Mode: ModeExplainer, VideoStyle: "2d"})
	for _, want := range []string{
		"NARRATED EXPLAINER", `"narrationTh"`, "START every narration with a Thai word",
		"No Latin letters and no digits", "Never invent statistics", "FLAT 2D EDUCATIONAL ANIMATION",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("explainer system prompt is missing %q", want)
		}
	}
	if strings.Contains(systemFor(Brief{Mode: ModeFilm}), "NARRATED EXPLAINER") {
		t.Error("another mode must not carry the explainer rules")
	}
}

func TestUserPromptGivesTheNarrationBudget(t *testing.T) {
	p := userPrompt(Brief{Mode: ModeExplainer, Topic: "ขาดดุลแคลอรี่", Shots: 4, Seconds: 8})
	if !strings.Contains(p, "at most 8 seconds") || !strings.Contains(p, "at most 82 characters") {
		t.Errorf("the prompt should state the ceiling and the budget:\n%s", p)
	}
	if strings.Contains(userPrompt(Brief{Topic: "x", Shots: 2, Seconds: 8}), "narrationTh") {
		t.Error("a plain brief must not mention narration")
	}
}

// explainerReply builds a reply with one narration per shot.
func explainerReply(narrations ...string) string {
	var shots []string
	for _, n := range narrations {
		shots = append(shots, `{"picturePrompt":"p","motionPrompt":"m. Static camera, no zoom.","narrationTh":"`+n+`"}`)
	}
	return `{"title":"t","scene":"A plain room.","shots":[` + strings.Join(shots, ",") + `]}`
}

func TestParseWithNarrationSetsTheLengthFromIt(t *testing.T) {
	short := "การนอนหลับช่วยให้ร่างกายฟื้นตัว" // 26 chars → 2.9 s → 4 on a Veo-like list
	long := strings.Repeat("ก", 70)            // 6.9 s → 8
	tooLong := strings.Repeat("ก", 200)        // never fits
	opt := parseOptions{Narration: true, SecondsOptions: []int{4, 6, 8}}
	d, err := parseWith(explainerReply(short, long, tooLong), 3, 8, opt)
	if err != nil {
		t.Fatal(err)
	}
	got := []int{d.Shots[0].Seconds, d.Shots[1].Seconds, d.Shots[2].Seconds}
	if got[0] != 4 || got[1] != 8 || got[2] != 8 {
		t.Errorf("lengths = %v, want [4 8 8]", got)
	}
	if d.Shots[0].NarrationTooLong || d.Shots[1].NarrationTooLong || !d.Shots[2].NarrationTooLong {
		t.Errorf("only the 200-character narration is too long: %+v", d.Shots)
	}
	if d.Shots[0].NarrationTh != short {
		t.Errorf("narration not kept: %q", d.Shots[0].NarrationTh)
	}
}

func TestParseWithoutNarrationOptionIgnoresIt(t *testing.T) {
	d, err := parse(explainerReply("calorie 7700"), 1, 5) // would be refused in explainer mode
	if err != nil {
		t.Fatal(err)
	}
	if d.Shots[0].NarrationTh != "" || d.Shots[0].Seconds != 5 {
		t.Errorf("a plain parse must ignore narration: %+v", d.Shots[0])
	}
}

func TestParseWithNarrationRefusesUnspeakableText(t *testing.T) {
	opt := parseOptions{Narration: true, SecondsOptions: []int{4, 6, 8}}
	for name, n := range map[string]string{
		"latin":  "คำว่า calorie deficit",
		"digits": "ไขมัน 7700 แคลอรี่",
		"empty":  "",
	} {
		if _, err := parseWith(explainerReply(n), 1, 8, opt); err == nil || !strings.Contains(err.Error(), "narration of shot 1") {
			t.Errorf("%s: want a refusal naming the narration, got %v", name, err)
		}
	}
	// A missing field is as bad as an empty one.
	raw := `{"title":"t","scene":"s","shots":[{"picturePrompt":"p","motionPrompt":"m"}]}`
	if _, err := parseWith(raw, 1, 8, opt); err == nil {
		t.Error("a shot with no narrationTh must be refused in explainer mode")
	}
}

func TestDraftFromAsksAgainWhenTheNarrationHasLatinLetters(t *testing.T) {
	good := explainerReply("การกินน้อยกว่าที่ใช้ เรียกว่า แคลอรี่ ดีฟิซิท")
	llm := &scripted{replies: []string{explainerReply("calorie deficit คือการกินน้อย"), good}}
	b := Brief{Mode: ModeExplainer, Topic: "ขาดดุลแคลอรี่", Shots: 1, Seconds: 8, SecondsOptions: []int{4, 6, 8}}
	d, err := draftFrom(context.Background(), llm, "m", b)
	if err != nil {
		t.Fatal(err)
	}
	if len(llm.prompts) != 2 || !strings.Contains(llm.prompts[1], "Latin letters") {
		t.Errorf("the second try should be told what was wrong: %v", llm.prompts)
	}
	if d.CostUSD != 0.02 {
		t.Errorf("both costs are added, got %v", d.CostUSD)
	}
}

func TestFakeLLMNarratesAndPassesTheCheck(t *testing.T) {
	b := Brief{Mode: ModeExplainer, Topic: "anything", Shots: 4, Seconds: 8, SecondsOptions: []int{3, 4, 5, 6, 7, 8}}
	d, err := draftFrom(context.Background(), fakeLLM{}, "fake", b)
	if err != nil {
		t.Fatalf("the fake draft must pass the explainer checks: %v", err)
	}
	for i, s := range d.Shots {
		if s.NarrationTh == "" || s.Seconds < 3 || s.Seconds > 8 {
			t.Errorf("shot %d: %+v", i+1, s)
		}
	}
	// Different narrations give different lengths (so the screen shows it working).
	if d.Shots[0].Seconds == d.Shots[1].Seconds && d.Shots[1].Seconds == d.Shots[2].Seconds {
		t.Errorf("the fake narrations should differ in length: %+v", d.Shots)
	}
}
