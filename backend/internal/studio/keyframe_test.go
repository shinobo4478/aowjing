package studio

import (
	"math"
	"testing"
)

func TestKeyframeModels(t *testing.T) {
	models := KeyframeModels()
	if len(models) < 2 {
		t.Fatalf("want at least nano-banana and seedream, got %d models", len(models))
	}
	seen := map[string]bool{}
	for _, m := range models {
		if m.Key == "" || m.TextModel == "" || m.EditModel == "" || m.PriceUSD <= 0 {
			t.Errorf("model %+v is incomplete", m)
		}
		if seen[m.Key] {
			t.Errorf("duplicate key %q", m.Key)
		}
		seen[m.Key] = true
		if got, ok := KeyframeModelFor(m.Key); !ok || got.Key != m.Key {
			t.Errorf("KeyframeModelFor(%q) failed", m.Key)
		}
	}
	if _, ok := KeyframeModelFor("dalle"); ok {
		t.Error("unknown key must not be found")
	}
}

func TestKeyframeModelIDAndCost(t *testing.T) {
	m, _ := KeyframeModelFor("nano-banana")
	if got := m.ModelID(false); got != "fal-ai/nano-banana" {
		t.Errorf("text model = %q", got)
	}
	if got := m.ModelID(true); got != "fal-ai/nano-banana/edit" {
		t.Errorf("edit model = %q", got)
	}
	if got := m.CostUSD(3); math.Abs(got-0.1194) > 1e-9 {
		t.Errorf("CostUSD(3) = %v, want 0.1194", got)
	}
}

func TestKeyframePrompt(t *testing.T) {
	r := For("3d")
	plain := r.KeyframePrompt("a cat on a sofa", false)
	if plain == "a cat on a sofa" || len(plain) <= len("a cat on a sofa") {
		t.Errorf("3d should add its wording, got %q", plain)
	}
	withRefs := r.KeyframePrompt("a cat on a sofa", true)
	if withRefs == plain || withRefs[:len(keepLookInstruction)] != keepLookInstruction {
		t.Errorf("references should add the keep-the-look instruction first, got %q", withRefs)
	}
	bare := Recipe{}
	if got := bare.KeyframePrompt("x", false); got != "x" {
		t.Errorf("no suffix, no refs: want unchanged, got %q", got)
	}
}
