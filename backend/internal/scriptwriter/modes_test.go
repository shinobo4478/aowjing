package scriptwriter

import (
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

func TestValidMode(t *testing.T) {
	for s, want := range map[string]bool{
		"": true, "general": true, "product": true, "film": true,
		"Product": false, "ad": false, " ": false,
	} {
		if got := ValidMode(s); got != want {
			t.Errorf("ValidMode(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestSystemForAddsOnlyWhatTheBriefNeeds(t *testing.T) {
	tests := []struct {
		name       string
		brief      Brief
		has        []string
		hasNot     []string
		samePrompt bool // identical to the standing prompt
	}{
		{"plain brief is the standing prompt", Brief{}, nil, nil, true},
		{"general mode adds nothing", Brief{Mode: ModeGeneral, VideoStyle: "3d"}, nil, nil, true},
		{"product mode", Brief{Mode: ModeProduct},
			[]string{"PRODUCT VIDEO", "ONLY in the states listed", "the product"},
			[]string{"SHORT FILM", "PHOTOGRAPHIC", "CAST:"}, false},
		{"film mode", Brief{Mode: ModeFilm},
			[]string{"SHORT FILM", "PUNCHLINE", "No dialogue"},
			[]string{"PRODUCT VIDEO"}, false},
		{"realistic style", Brief{VideoStyle: "realistic"},
			[]string{"PHOTOGRAPHIC LIVE ACTION"}, []string{"PRODUCT VIDEO"}, false},
		{"cast", Brief{Cast: []CastNote{{Role: "hero", Name: "พี่แบด"}}},
			[]string{"CAST:", "the lead character"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := systemFor(tt.brief)
			if !strings.HasPrefix(got, systemPrompt) {
				t.Fatal("the standing prompt must stay at the front")
			}
			if tt.samePrompt && got != systemPrompt {
				t.Errorf("expected the standing prompt unchanged, got %d extra bytes", len(got)-len(systemPrompt))
			}
			for _, w := range tt.has {
				if !strings.Contains(got, w) {
					t.Errorf("system prompt lacks %q", w)
				}
			}
			for _, w := range tt.hasNot {
				if strings.Contains(got, w) {
					t.Errorf("system prompt should not contain %q", w)
				}
			}
		})
	}
}

// The writer tells the model to call the hero "the lead character"; the picture
// step names the picture with the same words. If either side changes, the two
// must change together or the image model gets a prompt that does not match.
func TestCastWordsMatchThePicturePrompt(t *testing.T) {
	picture := studio.Recipe{}.KeyframePrompt("x", 0, []studio.CastRef{
		{Role: studio.RoleHero, Name: "n", Pictures: 1},
	}, false)
	if !strings.Contains(picture, "the lead character") {
		t.Fatalf("the picture prompt no longer says 'the lead character':\n%s", picture)
	}
	if !strings.Contains(castRules, "the lead character") {
		t.Error("castRules must tell the model to say 'the lead character'")
	}
}

func TestUserPromptCastAndProduct(t *testing.T) {
	b := Brief{
		Topic: "x y z", Shots: 3, Seconds: 4, Mode: ModeProduct,
		Cast: []CastNote{
			{Role: "hero", Name: "พี่แบด", Note: "ชุดแดง"},
			{Role: "prop", Name: "ไม้แบด"},
		},
		Product: &ProductNote{Name: "กระติกม่วง", Note: "เก็บความเย็น", States: []string{"cap closed", "cap open", ""}},
	}
	p := userPrompt(b)
	for _, want := range []string{
		`- hero: "พี่แบด" (ชุดแดง)`,
		`- prop: "ไม้แบด"`,
		"PRODUCT:",
		"Owner's note (the only claims the video may make): เก็บความเย็น",
		`for these states: "cap closed", "cap open".`,
		"1 more picture(s) have no state written",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("user prompt lacks %q:\n%s", want, p)
		}
	}
	if !strings.HasSuffix(p, "Reply with the JSON only.") {
		t.Error("the reply instruction must stay last")
	}

	// No note: the model is told to claim nothing.
	b.Product.Note = ""
	if p := userPrompt(b); !strings.Contains(p, "so make no claims about the product") {
		t.Errorf("a product without a note must say to make no claims:\n%s", p)
	}

	// No cast and no product: the prompt is as before.
	plain := userPrompt(Brief{Topic: "x y z", Shots: 3, Seconds: 4})
	if strings.Contains(plain, "CAST") || strings.Contains(plain, "PRODUCT") {
		t.Errorf("a plain brief must not mention cast or product:\n%s", plain)
	}
}

func TestBriefValidateModes(t *testing.T) {
	ok := Brief{Topic: "a topic", Shots: 3, Seconds: 4}
	tests := []struct {
		name      string
		mutate    func(*Brief)
		wantField string
	}{
		{"general", func(b *Brief) {}, ""},
		{"film", func(b *Brief) { b.Mode = ModeFilm }, ""},
		{"unknown mode", func(b *Brief) { b.Mode = "ad" }, "mode"},
		{"product mode needs a product", func(b *Brief) { b.Mode = ModeProduct }, "productId"},
		{"product mode with a product", func(b *Brief) {
			b.Mode = ModeProduct
			b.Product = &ProductNote{Name: "p", States: []string{"a"}}
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := ok
			tt.mutate(&b)
			errs := b.Validate()
			if tt.wantField == "" {
				if len(errs) != 0 {
					t.Fatalf("want valid, got %v", errs)
				}
				return
			}
			if _, has := errs[tt.wantField]; !has {
				t.Fatalf("want an error on %q, got %v", tt.wantField, errs)
			}
		})
	}
}
