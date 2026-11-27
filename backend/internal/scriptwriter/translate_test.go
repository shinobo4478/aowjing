package scriptwriter

import (
	"context"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

func TestHasThai(t *testing.T) {
	for s, want := range map[string]bool{
		"The cat pounces.":                false,
		"แมวกระโจน":                       true,
		"The cat แมว":                     true,
		"":                                false,
		"Static camera, no zoom. 3D 2026": false,
	} {
		if got := hasThai(s); got != want {
			t.Errorf("hasThai(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestTranslateRequestValidate(t *testing.T) {
	ok := TranslateRequest{Kind: KindMotion, Direction: ToEnglish, Text: "  แมวกระโจน  "}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("a good request was refused: %v", errs)
	}
	if ok.Text != "แมวกระโจน" {
		t.Errorf("text should be trimmed, got %q", ok.Text)
	}
	for name, r := range map[string]TranslateRequest{
		"kind":      {Kind: "video", Direction: ToEnglish, Text: "x"},
		"direction": {Kind: KindMotion, Direction: "sideways", Text: "x"},
		"text":      {Kind: KindMotion, Direction: ToEnglish, Text: "   "},
	} {
		if errs := r.Validate(); errs[name] == "" {
			t.Errorf("want an error on %q, got %v", name, errs)
		}
	}
	long := TranslateRequest{Kind: KindMotion, Direction: ToEnglish, Text: strings.Repeat("ก", 2001)}
	if errs := long.Validate(); errs["text"] == "" {
		t.Error("an over-long text should be refused")
	}
}

func TestTranslateSystemHasTheRulesOfEachKind(t *testing.T) {
	motion := translateSystem(TranslateRequest{Kind: KindMotion, Direction: ToEnglish})
	picture := translateSystem(TranslateRequest{Kind: KindPicture, Direction: ToEnglish})
	end := translateSystem(TranslateRequest{Kind: KindEndPicture, Direction: ToEnglish})
	thai := translateSystem(TranslateRequest{Kind: KindMotion, Direction: ToThai})

	// Every Thai -> English prompt: translate, never invent.
	for name, p := range map[string]string{"motion": motion, "picture": picture, "end": end} {
		if !strings.Contains(p, "add nothing the Thai does not say") {
			t.Errorf("%s: the do-not-invent rule is missing", name)
		}
	}
	if !strings.Contains(motion, "ONLY what moves") || !strings.Contains(motion, "Static camera, no zoom.") {
		t.Errorf("motion rules missing:\n%s", motion)
	}
	if !strings.Contains(picture, `no "3D", "cartoon"`) || !strings.Contains(picture, "Never add text") {
		t.Errorf("picture rules missing:\n%s", picture)
	}
	if !strings.Contains(end, `Start with "Only the ..."`) {
		t.Errorf("end-picture rules missing:\n%s", end)
	}
	// English -> Thai is a plain faithful rendering: none of the kind rules.
	if !strings.Contains(thai, "natural Thai") || strings.Contains(thai, "ONLY what moves") {
		t.Errorf("the English -> Thai instruction is wrong:\n%s", thai)
	}
}

func TestTranslateFrom(t *testing.T) {
	ctx := context.Background()
	toEn := TranslateRequest{Kind: KindMotion, Direction: ToEnglish, Text: "แมวกระโจนใส่ถุง"}
	toTh := TranslateRequest{Kind: KindMotion, Direction: ToThai, Text: "The cat pounces on the bag."}

	t.Run("a clean English answer is used, quotes trimmed", func(t *testing.T) {
		s := &scripted{replies: []string{`"The cat pounces on the bag. Static camera, no zoom."`}}
		got, err := translateFrom(ctx, s, "m", toEn)
		if err != nil || got.Text != "The cat pounces on the bag. Static camera, no zoom." {
			t.Fatalf("got %+v, %v", got, err)
		}
		if got.CostUSD != 0.01 || got.Model != "m" {
			t.Errorf("cost/model = %v / %q", got.CostUSD, got.Model)
		}
		if !strings.HasPrefix(s.prompts[0], "Direction: toEnglish") {
			t.Errorf("the prompt should state the direction first: %q", s.prompts[0])
		}
	})
	t.Run("English with Thai left in it is retried, and the costs add", func(t *testing.T) {
		s := &scripted{replies: []string{"The cat แมว pounces.", "The cat pounces."}}
		got, err := translateFrom(ctx, s, "m", toEn)
		if err != nil || got.Text != "The cat pounces." {
			t.Fatalf("got %+v, %v", got, err)
		}
		if got.CostUSD != 0.02 {
			t.Errorf("cost = %v, want both calls (0.02)", got.CostUSD)
		}
		if !strings.Contains(s.prompts[1], "still contains Thai letters") {
			t.Errorf("the retry should say what was wrong: %q", s.prompts[1])
		}
	})
	t.Run("gives up after two bad answers", func(t *testing.T) {
		s := &scripted{replies: []string{"แมว", "แมวอีก", "The cat."}}
		if _, err := translateFrom(ctx, s, "m", toEn); err == nil {
			t.Fatal("want an error")
		}
		if len(s.prompts) != 2 {
			t.Errorf("asked %d times, want 2", len(s.prompts))
		}
	})
	t.Run("a Thai answer for toThai is used", func(t *testing.T) {
		s := &scripted{replies: []string{"แมวกระโจนใส่ถุง"}}
		got, err := translateFrom(ctx, s, "m", toTh)
		if err != nil || got.Text != "แมวกระโจนใส่ถุง" {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
	t.Run("a Thai request answered in English is refused", func(t *testing.T) {
		s := &scripted{replies: []string{"The cat pounces.", "Still English."}}
		if _, err := translateFrom(ctx, s, "m", toTh); err == nil {
			t.Fatal("want an error when the Thai has no Thai letters")
		}
	})
	t.Run("a provider error is returned at once", func(t *testing.T) {
		s := &scripted{} // no replies: Complete fails
		if _, err := translateFrom(ctx, s, "m", toEn); err == nil {
			t.Fatal("want the provider error")
		}
		if len(s.prompts) != 1 {
			t.Errorf("a provider failure must not be retried, asked %d times", len(s.prompts))
		}
	})
}

func TestFakeLLMTranslates(t *testing.T) {
	en, err := translateFrom(context.Background(), fakeLLM{}, "m", TranslateRequest{Kind: KindPicture, Direction: ToEnglish, Text: "แมว"})
	if err != nil || hasThai(en.Text) {
		t.Errorf("fake English should hold no Thai: %q %v", en.Text, err)
	}
	th, err := translateFrom(context.Background(), fakeLLM{}, "m", TranslateRequest{Kind: KindPicture, Direction: ToThai, Text: "cat"})
	if err != nil || !hasThai(th.Text) {
		t.Errorf("fake Thai should hold Thai: %q %v", th.Text, err)
	}
}

// The scripted double from writer_test.go is a generate.LLM.
var _ generate.LLM = (*scripted)(nil)
