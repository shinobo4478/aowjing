package scriptwriter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

const goodReply = `{"title":"The bag drop","scene":"A sunny room.","shots":[
 {"picturePrompt":"Close-up of the cat.","motionPrompt":"The bag drops. Static camera, no zoom."},
 {"picturePrompt":"The cat crouched.","motionPrompt":"The cat pounces. Static camera, no zoom."}]}`

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr string // a substring of the error, "" = ok
		check   func(t *testing.T, d Draft)
	}{
		{
			name: "plain json", raw: goodReply, want: 2,
			check: func(t *testing.T, d Draft) {
				if d.Title != "The bag drop" || len(d.Shots) != 2 {
					t.Fatalf("got %+v", d)
				}
				// The scene is put in front of every picture prompt.
				if d.Shots[0].PicturePrompt != "A sunny room. Close-up of the cat." {
					t.Errorf("picture prompt = %q", d.Shots[0].PicturePrompt)
				}
				// The motion prompt is left as written, and every shot gets the asked length.
				if d.Shots[1].MotionPrompt != "The cat pounces. Static camera, no zoom." || d.Shots[1].Seconds != 4 {
					t.Errorf("shot 2 = %+v", d.Shots[1])
				}
			},
		},
		{
			name: "wrapped in a code fence and prose", want: 2,
			raw: "Sure! Here is the draft:\n```json\n" + goodReply + "\n```\nHope it helps.",
		},
		{
			name: "extra shots are dropped", want: 1, raw: goodReply,
			check: func(t *testing.T, d Draft) {
				if len(d.Shots) != 1 {
					t.Errorf("want 1 shot, got %d", len(d.Shots))
				}
			},
		},
		{name: "too few shots", want: 3, raw: goodReply, wantErr: "2 shots, not the 3"},
		{name: "not json", want: 1, raw: "I cannot do that.", wantErr: "JSON"},
		{name: "broken json", want: 1, raw: `{"title": "x", "shots": [}`, wantErr: "could not be read"},
		{name: "no scene", want: 1, raw: `{"title":"x","scene":" ","shots":[{"picturePrompt":"a","motionPrompt":"b"}]}`, wantErr: "no scene"},
		{
			name: "empty prompt", want: 1, wantErr: "missing a prompt",
			raw: `{"title":"x","scene":"s","shots":[{"picturePrompt":"a","motionPrompt":""}]}`,
		},
		{
			name: "too long", want: 1, wantErr: "too long",
			raw: `{"title":"x","scene":"s","shots":[{"picturePrompt":"` + strings.Repeat("a", 2000) + `","motionPrompt":"b"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parse(tt.raw, tt.want, 4)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, d)
			}
		})
	}
}

func TestBriefValidate(t *testing.T) {
	ok := Brief{Topic: "a cat and a bag", Shots: 3, Seconds: 4}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("a good brief was refused: %v", errs)
	}
	for name, b := range map[string]Brief{
		"topic":   {Topic: "ab", Shots: 3, Seconds: 4},
		"shots":   {Topic: "a cat", Shots: 13, Seconds: 4},
		"seconds": {Topic: "a cat", Shots: 3, Seconds: 2},
	} {
		if errs := b.Validate(); errs[name] == "" {
			t.Errorf("want an error on %q, got %v", name, errs)
		}
	}
}

func TestUserPromptCarriesTheBrief(t *testing.T) {
	p := userPrompt(Brief{Topic: "แมวกับถุง", Shots: 4, Seconds: 5, ProfileName: "Fal", Niche: "3D Cute Pet", Description: "cheerful"})
	for _, want := range []string{"แมวกับถุง", "Number of shots: 4", "about 5 seconds", "Niche: 3D Cute Pet", "cheerful"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt is missing %q:\n%s", want, p)
		}
	}
}

// scripted answers a fixed list of replies, one per call, and records the prompts.
type scripted struct {
	replies []string
	prompts []string
}

func (s *scripted) Complete(_ context.Context, req generate.LLMRequest) (generate.LLMResult, error) {
	s.prompts = append(s.prompts, req.Prompt)
	if len(s.replies) == 0 {
		return generate.LLMResult{}, errors.New("no more replies")
	}
	r := s.replies[0]
	s.replies = s.replies[1:]
	return generate.LLMResult{Text: r, CostUSD: 0.01}, nil
}

func TestFakeLLMAnswersWhatWasAsked(t *testing.T) {
	b := Brief{Topic: "anything", Shots: 5, Seconds: 4}
	res, err := fakeLLM{}.Complete(context.Background(), generate.LLMRequest{Prompt: userPrompt(b)})
	if err != nil {
		t.Fatal(err)
	}
	d, err := parse(res.Text, 5, 4)
	if err != nil || len(d.Shots) != 5 {
		t.Fatalf("fake draft not usable: %v %+v", err, d)
	}
}

func TestRetryLoopShape(t *testing.T) {
	b := Brief{Topic: "x y z", Shots: 2, Seconds: 4}

	t.Run("second try works and costs add up", func(t *testing.T) {
		s := &scripted{replies: []string{"not json", goodReply}}
		d, err := draftFrom(context.Background(), s, "test-model", b)
		if err != nil || len(d.Shots) != 2 {
			t.Fatalf("got %v %+v", err, d)
		}
		if d.CostUSD != 0.02 {
			t.Errorf("cost = %v, want both calls (0.02)", d.CostUSD)
		}
		if len(s.prompts) != 2 || !strings.Contains(s.prompts[1], "could not be used") {
			t.Errorf("the retry should say what was wrong: %q", s.prompts)
		}
	})
	t.Run("gives up after two tries", func(t *testing.T) {
		s := &scripted{replies: []string{"nope", "still nope", goodReply}}
		if _, err := draftFrom(context.Background(), s, "test-model", b); err == nil {
			t.Fatal("want an error after two bad replies")
		}
		if len(s.prompts) != 2 {
			t.Errorf("asked %d times, want 2", len(s.prompts))
		}
	})
}
