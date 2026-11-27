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
