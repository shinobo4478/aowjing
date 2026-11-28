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

func TestLookNotesLimitCountsCharacters(t *testing.T) {
	base := profileInput{Name: "ช่องการ์ตูน", Niche: "ความรู้", Provider: "text", VideoStyle: "2d", VideoModel: "kling-v3"}
	ok := base
	ok.LookNotes = strings.Repeat("ก", maxLookNotesRunes) // chars, not bytes
	if errs := ok.normalizeAndValidate(); errs["lookNotes"] != "" {
		t.Errorf("%d characters must be accepted, got %v", maxLookNotesRunes, errs)
	}
	long := base
	long.LookNotes = strings.Repeat("a", maxLookNotesRunes+1)
	if errs := long.normalizeAndValidate(); errs["lookNotes"] == "" {
		t.Error("one character over the limit must be refused")
	}
	trim := base
	trim.LookNotes = "  thick outlines  "
	trim.normalizeAndValidate()
	if trim.LookNotes != "thick outlines" {
		t.Errorf("look notes should be trimmed, got %q", trim.LookNotes)
	}
}

func TestNarratorVoiceIsAKnownVoiceOrEmpty(t *testing.T) {
	base := profileInput{Name: "ช่องความรู้", Niche: "การเรียนรู้", Provider: "fal", VideoStyle: "2d", VideoModel: "kling-v3"}
	tests := []struct {
		voice string
		ok    bool
	}{
		{"", true},                 // none chosen yet
		{"eleven-v3-rachel", true}, // a listed voice
		{"  eleven-v3-sarah  ", true},
		{"custom:Alice", true}, // a voice the owner named
		{"custom:21m00Tcm4TlvDq8ikWAM", true},
		{"eleven-v3-aria", false}, // removed after the owner's listening test
		{"custom:bad;name", false},
		{"custom:", false},
		{"nope", false},
	}
	for _, tt := range tests {
		in := base
		in.NarratorVoice = tt.voice
		errs := in.normalizeAndValidate()
		if got := errs["narratorVoice"] == ""; got != tt.ok {
			t.Errorf("narrator %q: valid = %v, want %v (%v)", tt.voice, got, tt.ok, errs)
		}
	}
	in := base
	in.NarratorVoice = "  eleven-v3-roger  "
	in.normalizeAndValidate()
	if in.NarratorVoice != "eleven-v3-roger" {
		t.Errorf("the narrator voice should be trimmed, got %q", in.NarratorVoice)
	}
}
