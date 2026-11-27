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
	plain := r.KeyframePrompt("a cat on a sofa", 0, nil, false)
	if plain == "a cat on a sofa" || len(plain) <= len("a cat on a sofa") {
		t.Errorf("3d should add its wording, got %q", plain)
	}
	bare := Recipe{}
	if got := bare.KeyframePrompt("x", 0, nil, false); got != "x" {
		t.Errorf("no suffix, no refs: want unchanged, got %q", got)
	}
}

func TestKeyframePromptCastNamesEachPicture(t *testing.T) {
	bare := Recipe{}
	cast := []CastRef{
		{Role: RoleHero, Name: "Bad", Note: "red kit", Pictures: 2},
		{Role: RoleProp, Name: `Bottle "X"`, Pictures: 1},
		{Role: RoleCostar, Name: "Skipped", Pictures: 0}, // no pictures: not described
	}
	got := bare.KeyframePrompt("Bad drinks", 0, cast, false)
	for _, want := range []string{
		`Pictures 1-2 are "Bad", the lead character (red kit).`,
		`Picture 3 is "Bottle 'X'", an object.`,
		keepCastInstruction,
		"Bad drinks",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Skipped") {
		t.Errorf("a member with no pictures must not be described:\n%s", got)
	}
	if !strings.HasSuffix(got, "Bad drinks") {
		t.Errorf("the description should come last, got %q", got)
	}
}

func TestKeyframePromptCastFollowsScenePictures(t *testing.T) {
	bare := Recipe{}
	got := bare.KeyframePrompt("x", 2, []CastRef{{Role: RoleHero, Name: "Bad", Pictures: 1}}, false)
	if !strings.HasPrefix(got, keepSceneInstruction) {
		t.Errorf("scene instruction should come first, got %q", got)
	}
	if !strings.Contains(got, `Picture 3 is "Bad"`) {
		t.Errorf("cast numbering should start after the 2 scene pictures, got %q", got)
	}
}

func TestClipSeconds(t *testing.T) {
	// The outer bound: any model's range.
	for n, want := range map[int]bool{2: false, 3: true, 5: true, 15: true, 16: false, 0: false, -1: false} {
		if got := ValidClipSeconds(n); got != want {
			t.Errorf("ValidClipSeconds(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestKeyframePromptScene(t *testing.T) {
	r := For("3d")
	scene := r.KeyframePrompt("same room, camera from the left", 1, nil, false)
	if scene[:len(keepSceneInstruction)] != keepSceneInstruction {
		t.Errorf("scene pictures should add the keep-the-scene instruction first, got %q", scene)
	}
	// No cast ticked: no cast wording at all.
	if strings.Contains(scene, keepCastInstruction) {
		t.Errorf("no cast, so no cast instruction, got %q", scene)
	}
}

func TestKeyframePromptSameCamera(t *testing.T) {
	r := For("3d")
	got := r.KeyframePrompt("the cat crouches", 1, nil, true)
	if !strings.HasPrefix(got, keepSceneSameCameraInstruction) {
		t.Errorf("an end picture should use the same-camera instruction, got %q", got)
	}
	if strings.Contains(got, "camera angle") {
		t.Errorf("same-camera wording must not invite a new camera angle, got %q", got)
	}
	// Without scene pictures there is nothing to keep fixed: the flag changes nothing.
	if a, b := r.KeyframePrompt("x", 0, nil, true), r.KeyframePrompt("x", 0, nil, false); a != b {
		t.Errorf("same-camera without scene pictures should be ignored:\n%q\n%q", a, b)
	}
}

func TestValidCastRole(t *testing.T) {
	for s, want := range map[string]bool{"hero": true, "costar": true, "prop": true, "place": true, "": false, "Hero": false, "villain": false} {
		if got := ValidCastRole(s); got != want {
			t.Errorf("ValidCastRole(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestKeyframePromptNamesProductPictures(t *testing.T) {
	bare := Recipe{}
	cast := []CastRef{
		{Role: RoleHero, Name: "Bad", Pictures: 2},
		{Role: RoleProduct, Name: "Bottle", Note: "cap open", Pictures: 1},
		{Role: RoleProduct, Name: "Bottle", Note: "in hand", Pictures: 1},
	}
	got := bare.KeyframePrompt("Bad drinks from the bottle", 1, cast, false)
	for _, want := range []string{
		`Pictures 2-3 are "Bad", the lead character.`,
		`Picture 4 is "Bottle", the product (cap open).`,
		`Picture 5 is "Bottle", the product (in hand).`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt lacks %q:\n%s", want, got)
		}
	}
}

func TestProductIsNotACastRole(t *testing.T) {
	// A product's pictures live on the Product; a cast member cannot be one.
	if ValidCastRole(string(RoleProduct)) {
		t.Error("product must not be a valid cast_members role (the table's CHECK would refuse it)")
	}
}
