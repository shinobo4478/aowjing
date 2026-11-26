package studio

import (
	"math"
	"strings"
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
	plain := r.KeyframePrompt("a cat on a sofa", 0, 0)
	if plain == "a cat on a sofa" || len(plain) <= len("a cat on a sofa") {
		t.Errorf("3d should add its wording, got %q", plain)
	}
	withRefs := r.KeyframePrompt("a cat on a sofa", 2, 0)
	if withRefs == plain || withRefs[:len(keepLookInstruction)] != keepLookInstruction {
		t.Errorf("references should add the keep-the-look instruction first, got %q", withRefs)
	}
	bare := Recipe{}
	if got := bare.KeyframePrompt("x", 0, 0); got != "x" {
		t.Errorf("no suffix, no refs: want unchanged, got %q", got)
	}
}

func TestImageClipUSD(t *testing.T) {
	// 5 s at $0.084/s is $0.42 (the first real clip); float rounding must not leak.
	if got := For("3d").ImageClipUSD(5); got != 0.42 {
		t.Errorf("3d image clip = %v, want 0.42", got)
	}
	if got := (Recipe{}).ImageClipUSD(5); got != 0 {
		t.Errorf("unknown price should be 0, got %v", got)
	}
}

func TestClipSeconds(t *testing.T) {
	for n, want := range map[int]bool{2: false, 3: true, 5: true, 15: true, 16: false, 0: false, -1: false} {
		if got := ValidClipSeconds(n); got != want {
			t.Errorf("ValidClipSeconds(%d) = %v, want %v", n, got, want)
		}
	}
	// Price scales with the length (assumed linear, see ImageClipUSD).
	if got := For("3d").ImageClipUSD(10); got != 0.84 {
		t.Errorf("10 s = %v, want 0.84", got)
	}
}

func TestKeyframePromptScene(t *testing.T) {
	r := For("3d")
	scene := r.KeyframePrompt("same room, camera from the left", 2, 1)
	if scene[:len(keepSceneInstruction)] != keepSceneInstruction {
		t.Errorf("scene pictures should add the keep-the-scene instruction first, got %q", scene)
	}
	// With scene pictures the look instruction is not added on top of it.
	if strings.Contains(scene, keepLookInstruction) {
		t.Errorf("scene instruction should replace the look one, got %q", scene)
	}
	look := r.KeyframePrompt("x", 2, 0)
	if !strings.HasPrefix(look, keepLookInstruction) {
		t.Errorf("profile references alone use the look instruction, got %q", look)
	}
}
