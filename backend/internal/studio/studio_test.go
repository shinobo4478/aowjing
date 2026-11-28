package studio

import (
	"strings"
	"testing"
)

func TestValidAndFor(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantValid bool
		wantStyle Style // what For returns
	}{
		{"3d", "3d", true, Style3D},
		{"unknown is not valid", "claymation", false, DefaultStyle},
		{"empty is not valid", "", false, DefaultStyle},
		{"case matters", "3D", false, DefaultStyle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Valid(tt.in); got != tt.wantValid {
				t.Errorf("Valid(%q) = %v, want %v", tt.in, got, tt.wantValid)
			}
			if got := For(tt.in).Style; got != tt.wantStyle {
				t.Errorf("For(%q).Style = %q, want %q", tt.in, got, tt.wantStyle)
			}
		})
	}
}

// Every registered recipe must be usable: named and reachable through its own
// name. Catches a half-filled entry when a style is
// added.
func TestRegistryIsComplete(t *testing.T) {
	seen := map[Style]bool{}
	for _, r := range Recipes() {
		if r.Style == "" || r.Label == "" {
			t.Errorf("recipe %+v has an empty required field", r)
		}
		if seen[r.Style] {
			t.Errorf("style %q is registered twice", r.Style)
		}
		seen[r.Style] = true
		if !Valid(string(r.Style)) {
			t.Errorf("style %q is registered but not Valid", r.Style)
		}
	}
	if !Valid(string(DefaultStyle)) {
		t.Errorf("DefaultStyle %q is not registered", DefaultStyle)
	}
}

func TestCompose(t *testing.T) {
	r := Recipe{TextSuffix: "look words", ImageSuffix: "motion words"}
	bare := Recipe{}

	tests := []struct {
		name     string
		recipe   Recipe
		prompt   string
		hasImage bool
		want     string
	}{
		{"text clip gets the text suffix", r, "a cat", false, "a cat\n\nlook words"},
		{"image clip gets the image suffix", r, "slow zoom", true, "slow zoom\n\nmotion words"},
		{"no suffix leaves the prompt alone (text)", bare, "a cat", false, "a cat"},
		{"no suffix leaves the prompt alone (image)", bare, "slow zoom", true, "slow zoom"},
		{"image clip ignores the text suffix", Recipe{TextSuffix: "look words"}, "slow zoom", true, "slow zoom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.recipe.Compose(tt.prompt, tt.hasImage); got != tt.want {
				t.Errorf("Compose = %q, want %q", got, tt.want)
			}
		})
	}
}

// The 3D recipe must not restyle a starting picture (a real product photo).
func TestStyle3DKeepsPicturesAsTheyAre(t *testing.T) {
	if s := For("3d").ImageSuffix; s != "" {
		t.Errorf("3d ImageSuffix = %q, want empty", s)
	}
}

func TestRealisticKeepsProductsAsTheyAre(t *testing.T) {
	r := For("realistic")
	if r.Style != StyleRealistic {
		t.Fatalf("For(realistic).Style = %q", r.Style)
	}
	// A real product photo must never be restyled by a clip.
	if r.ImageSuffix != "" {
		t.Errorf("realistic ImageSuffix = %q, want empty", r.ImageSuffix)
	}
	if strings.Contains(strings.ToLower(r.KeyframeSuffix), "cinematic") {
		t.Errorf("realistic pictures should not ask for a cinematic look: %q", r.KeyframeSuffix)
	}
}

func TestStyle2DIsFlatAndStaysFlatInClips(t *testing.T) {
	r := For("2d")
	if r.Style != Style2D || !Valid("2d") {
		t.Fatalf("2d must be a valid style, got %+v", r)
	}
	for name, s := range map[string]string{"KeyframeSuffix": r.KeyframeSuffix, "TextSuffix": r.TextSuffix} {
		for _, want := range []string{"Flat 2D vector", "no gradients", "no shading", "No text or letters"} {
			if !strings.Contains(s, want) {
				t.Errorf("%s lacks %q: %s", name, want, s)
			}
		}
	}
	// Unlike realistic, a clip from a flat picture is still told to stay flat.
	if !strings.Contains(r.ImageSuffix, "no 3D rotation") || !strings.Contains(r.ImageSuffix, "no shading") {
		t.Errorf("2d ImageSuffix must keep the clip flat: %q", r.ImageSuffix)
	}
}

// Each profile has its OWN look: the recipe holds only what all flat drawings
// share (no palette, no outline colour), and WithLook adds a profile's notes.
func TestRecipeHoldsNoPaletteAndWithLookAddsTheProfilesOwn(t *testing.T) {
	r := For("2d")
	for _, banned := range []string{"navy", "teal", "coral", "cream", "palette"} {
		if strings.Contains(strings.ToLower(r.KeyframeSuffix), banned) {
			t.Errorf("the 2d recipe must not fix a colour (%q): each profile has its own: %s", banned, r.KeyframeSuffix)
		}
	}
	notes := "thick black outlines, warm pastel palette: peach, mint, butter yellow"
	got := r.WithLook("  " + notes + "  ")
	for name, s := range map[string]string{"KeyframeSuffix": got.KeyframeSuffix, "TextSuffix": got.TextSuffix} {
		if !strings.HasSuffix(s, "This channel's own look: "+notes) {
			t.Errorf("%s should end with the profile's notes: %s", name, s)
		}
	}
	// A clip made from a picture is left alone: the picture carries the look.
	if got.ImageSuffix != r.ImageSuffix {
		t.Errorf("ImageSuffix must not change: %q", got.ImageSuffix)
	}
	// The original recipe is not changed (Recipe is a value).
	if strings.Contains(r.KeyframeSuffix, "own look") {
		t.Error("WithLook must return a copy, not edit the registry")
	}
	// Empty or blank notes change nothing.
	for _, blank := range []string{"", "   ", "\n"} {
		if w := r.WithLook(blank); w.KeyframeSuffix != r.KeyframeSuffix || w.TextSuffix != r.TextSuffix {
			t.Errorf("WithLook(%q) should change nothing", blank)
		}
	}
	// The notes reach the final picture prompt.
	if p := got.KeyframePrompt("a cat teaches", 0, nil, false); !strings.Contains(p, notes) {
		t.Errorf("the keyframe prompt must carry the notes: %s", p)
	}
}
