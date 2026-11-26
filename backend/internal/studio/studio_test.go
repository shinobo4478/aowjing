package studio

import "testing"

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

// Every registered recipe must be usable: named, with both models set, and
// reachable through its own name. Catches a half-filled entry when a style is
// added.
func TestRegistryIsComplete(t *testing.T) {
	seen := map[Style]bool{}
	for _, r := range Recipes() {
		if r.Style == "" || r.Label == "" || r.TextModel == "" || r.ImageModel == "" {
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
