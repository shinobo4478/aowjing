package studio

import (
	"math"
	"strings"
	"testing"
)

func TestVoiceCandidatesAreComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range VoiceCandidates() {
		if v.Key == "" || v.Label == "" || v.Model == "" || v.PricePer1KChars <= 0 || v.Note == "" {
			t.Errorf("candidate %+v has an empty required field", v)
		}
		if seen[v.Key] {
			t.Errorf("voice %q is registered twice", v.Key)
		}
		seen[v.Key] = true
		if v.Family != VoiceMiniMax && v.Family != VoiceElevenLabs {
			t.Errorf("voice %q has an unknown family %q", v.Key, v.Family)
		}
		if v.Family == VoiceMiniMax && !strings.HasPrefix(v.VoiceID, "Thai_") {
			t.Errorf("a MiniMax voice must be one of the Thai system voices, got %q", v.VoiceID)
		}
	}
	got, ok := VoiceCandidateFor("eleven-v3-rachel")
	if !ok {
		t.Fatal("eleven-v3-rachel is missing")
	}
	// fal bills per character: 400 Thai characters at $0.10 per 1000.
	if c := got.CostUSD(strings.Repeat("ก", 400)); math.Abs(c-0.04) > 1e-9 {
		t.Errorf("400 Thai characters on ElevenLabs v3 should cost $0.04, got %v", c)
	}
	if _, ok := VoiceCandidateFor("nope"); ok {
		t.Error("an unknown key must not be found")
	}
}
