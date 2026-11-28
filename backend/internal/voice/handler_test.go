package voice

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

func TestValidate(t *testing.T) {
	long := strings.Repeat("ก", studio.MaxVoiceSampleChars+1) // counted in characters, not bytes
	ok := strings.Repeat("ก", studio.MinVoiceSampleChars)
	tests := []struct {
		name      string
		text      string
		keys      []string
		wantField string // "" = valid
		wantN     int
	}{
		{"valid", ok, []string{"eleven-v3-sarah", "eleven-v3-rachel"}, "", 2},
		{"text is trimmed before counting", "   " + ok + "  ", []string{"eleven-v3-rachel"}, "", 1},
		{"too short", "สวัสดี", []string{"eleven-v3-rachel"}, "text", 0},
		{"too long", long, []string{"eleven-v3-rachel"}, "text", 0},
		{"exactly the maximum", strings.Repeat("ก", studio.MaxVoiceSampleChars), []string{"eleven-v3-rachel"}, "", 1},
		{"no voice", ok, nil, "candidates", 0},
		{"unknown voice", ok, []string{"nope"}, "candidates", 0},
		{"a repeat is counted once", ok, []string{"eleven-v3-rachel", "eleven-v3-rachel"}, "", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cands, errs := validate(tt.text, tt.keys)
			if tt.wantField == "" {
				if len(errs) != 0 || len(cands) != tt.wantN {
					t.Fatalf("want valid with %d voices, got %d voices, errors %v", tt.wantN, len(cands), errs)
				}
				return
			}
			if _, has := errs[tt.wantField]; !has {
				t.Fatalf("want an error on %q, got %v", tt.wantField, errs)
			}
		})
	}
}

func TestValidateRefusesMoreVoicesThanTheCap(t *testing.T) {
	all := studio.VoiceCandidates()
	if len(all) > studio.MaxVoiceCandidates {
		t.Fatalf("the registry (%d) holds more voices than one request may ask for (%d)", len(all), studio.MaxVoiceCandidates)
	}
}

// scripted is a Voice that answers per voice id, so each case is deterministic.
type scripted map[string]func() (generate.VoiceResult, error)

func (s scripted) Speak(_ context.Context, req generate.VoiceRequest) (generate.VoiceResult, error) {
	return s[req.VoiceID]()
}

func TestRunSamplesKeepsOrderAndOneFailureDoesNotSpoilTheOthers(t *testing.T) {
	cands := []studio.VoiceCandidate{
		{Key: "a", Label: "A", Family: studio.VoiceMiniMax, Model: "m1", VoiceID: "va", PricePer1KChars: 0.10},
		{Key: "b", Label: "B", Family: studio.VoiceElevenLabs, Model: "m2", VoiceID: "vb", PricePer1KChars: 0.10},
		{Key: "c", Label: "C", Family: studio.VoiceMiniMax, Model: "m3", VoiceID: "vc", PricePer1KChars: 0.06},
	}
	text := strings.Repeat("ก", 1000)
	v := scripted{
		"va": func() (generate.VoiceResult, error) {
			return generate.VoiceResult{AudioURL: "u-a", DurationMs: 61000}, nil
		},
		"vb": func() (generate.VoiceResult, error) {
			return generate.VoiceResult{}, errors.New("language not supported")
		},
		"vc": func() (generate.VoiceResult, error) { return generate.VoiceResult{AudioURL: "u-c"}, nil },
	}
	got := runSamples(context.Background(), v, cands, text)
	if len(got) != 3 || got[0].Key != "a" || got[1].Key != "b" || got[2].Key != "c" {
		t.Fatalf("results must keep the candidates' order, got %+v", got)
	}
	if !got[0].OK || got[0].AudioURL != "u-a" || got[0].DurationMs != 61000 {
		t.Errorf("a: %+v", got[0])
	}
	if got[1].OK || got[1].Error != "language not supported" || got[1].CostUSD != 0 {
		t.Errorf("a failed voice carries its error and costs nothing: %+v", got[1])
	}
	// 1000 characters: $0.10 per 1K = 0.10; $0.06 per 1K = 0.06.
	if math.Abs(got[0].CostUSD-0.10) > 1e-9 || math.Abs(got[2].CostUSD-0.06) > 1e-9 {
		t.Errorf("costs: %v and %v", got[0].CostUSD, got[2].CostUSD)
	}
}

// internal/generate and internal/studio each spell the two request families; a
// request with a family generate does not know would only fail at call time.
func TestFamiliesAgreeBetweenStudioAndGenerate(t *testing.T) {
	if string(studio.VoiceMiniMax) != generate.VoiceFamilyMiniMax || string(studio.VoiceElevenLabs) != generate.VoiceFamilyElevenLabs {
		t.Fatalf("family names differ: studio %q/%q, generate %q/%q",
			studio.VoiceMiniMax, studio.VoiceElevenLabs, generate.VoiceFamilyMiniMax, generate.VoiceFamilyElevenLabs)
	}
}

func TestFakeVoiceAnswersWithPlayableAudio(t *testing.T) {
	res, err := generate.FakeVoice{}.Speak(context.Background(), generate.VoiceRequest{Text: "สวัสดี"})
	if err != nil || !strings.HasPrefix(res.AudioURL, "data:audio/wav;base64,") || res.DurationMs != 1000 {
		t.Fatalf("fake voice: %+v, %v", res, err)
	}
	if _, err := (generate.FakeVoice{}).Speak(context.Background(), generate.VoiceRequest{Text: "  "}); err == nil {
		t.Error("fake voice must refuse an empty text, like the real one")
	}
}
