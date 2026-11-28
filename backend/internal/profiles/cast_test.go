package profiles

import (
	"strings"
	"testing"
)

func TestCastInputValidate(t *testing.T) {
	long := strings.Repeat("ก", maxCastNameRunes+1) // counted in characters, not bytes
	tests := []struct {
		name      string
		in        castInput
		wantField string // "" = valid
	}{
		{"hero", castInput{Role: "hero", Name: "พี่แบด"}, ""},
		{"trims and keeps a note", castInput{Role: "prop", Name: "  กระติก ", Note: " ฝาเปิด "}, ""},
		{"unknown role", castInput{Role: "villain", Name: "x"}, "role"},
		{"empty role", castInput{Name: "x"}, "role"},
		{"blank name", castInput{Role: "hero", Name: "   "}, "name"},
		{"name too long", castInput{Role: "hero", Name: long}, "name"},
		{"name at the limit", castInput{Role: "hero", Name: strings.Repeat("ก", maxCastNameRunes)}, ""},
		{"note too long", castInput{Role: "hero", Name: "x", Note: strings.Repeat("a", maxCastNoteRunes+1)}, "note"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			errs := in.validate()
			if tt.wantField == "" {
				if len(errs) != 0 {
					t.Fatalf("want valid, got %v", errs)
				}
				if in.Name != strings.TrimSpace(in.Name) || in.Note != strings.TrimSpace(in.Note) {
					t.Errorf("name/note not trimmed: %+v", in)
				}
				return
			}
			if _, ok := errs[tt.wantField]; !ok {
				t.Fatalf("want an error on %q, got %v", tt.wantField, errs)
			}
		})
	}
}

// Limits are counted in characters, not bytes: a Thai letter is 3 bytes, so a
// byte count would allow a Thai description of only about 200 letters.
func TestProfileLimitsCountCharactersNotBytes(t *testing.T) {
	thai := strings.Repeat("ก", 600) // 600 characters, 1800 bytes
	in := profileInput{Name: "ช่องหนังสั้น", Niche: "ตลก", Description: thai, Provider: "text", VideoStyle: "3d", VideoModel: "kling-v3"}
	if errs := in.normalizeAndValidate(); errs["description"] != "" || errs["name"] != "" || errs["niche"] != "" {
		t.Errorf("600 Thai characters must be accepted, got %v", errs)
	}
	in.Description = thai + "ก"
	if errs := in.normalizeAndValidate(); errs["description"] == "" {
		t.Error("601 characters must be refused")
	}
	// Two Thai letters are a valid niche (2 characters, 6 bytes); one is not.
	in.Description = ""
	in.Niche = "ก"
	if errs := in.normalizeAndValidate(); errs["niche"] == "" {
		t.Error("a one-letter niche must be refused")
	}
}
