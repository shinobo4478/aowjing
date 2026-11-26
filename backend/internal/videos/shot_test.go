package videos

import (
	"strings"
	"testing"
)

func TestShotInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		in          shotInput
		wantFields  []string // fields that must have an error
		wantSeconds int      // after normalizing, when valid
	}{
		{"empty is a valid plan, default length", shotInput{}, nil, 5},
		{"explicit length", shotInput{Prompt: "pan left", Seconds: 8}, nil, 8},
		{"shortest", shotInput{Seconds: 3}, nil, 3},
		{"longest", shotInput{Seconds: 15}, nil, 15},
		{"too short", shotInput{Seconds: 2}, []string{"seconds"}, 0},
		{"too long", shotInput{Seconds: 16}, []string{"seconds"}, 0},
		{"negative", shotInput{Seconds: -4}, []string{"seconds"}, 0},
		{"prompt too long", shotInput{Prompt: strings.Repeat("a", 2001)}, []string{"prompt"}, 0},
		{"prompt at the limit", shotInput{Prompt: strings.Repeat("ก", 2000)}, nil, 5},
		{"bad link", shotInput{ImageURL: "ftp://x/y.png"}, []string{"imageUrl"}, 0},
		{"link and picture", shotInput{ImageURL: "https://x.test/a.png", ImageID: "abc"}, []string{"imageId"}, 0},
		{"link only", shotInput{ImageURL: " https://x.test/a.png "}, nil, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			errs := in.normalizeAndValidate()
			for _, f := range tt.wantFields {
				if _, ok := errs[f]; !ok {
					t.Errorf("want an error on %q, got %v", f, errs)
				}
			}
			if len(tt.wantFields) == 0 {
				if len(errs) != 0 {
					t.Fatalf("unexpected errors: %v", errs)
				}
				if in.Seconds != tt.wantSeconds {
					t.Errorf("Seconds = %d, want %d", in.Seconds, tt.wantSeconds)
				}
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	if got, err := validateTitle("  Cat and the bag "); err != nil || got != "Cat and the bag" {
		t.Errorf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("x", 121)} {
		if _, err := validateTitle(bad); err == nil {
			t.Errorf("title %q should be refused", bad)
		}
	}
}

func TestSameSet(t *testing.T) {
	tests := []struct {
		name      string
		want, got []string
		ok        bool
	}{
		{"same order", []string{"a", "b"}, []string{"a", "b"}, true},
		{"swapped", []string{"a", "b", "c"}, []string{"c", "a", "b"}, true},
		{"missing one", []string{"a", "b"}, []string{"a"}, false},
		{"extra one", []string{"a"}, []string{"a", "b"}, false},
		{"repeated id", []string{"a", "b"}, []string{"a", "a"}, false},
		{"unknown id", []string{"a", "b"}, []string{"a", "z"}, false},
		{"both empty", nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameSet(tt.want, tt.got); got != tt.ok {
				t.Errorf("sameSet(%v, %v) = %v, want %v", tt.want, tt.got, got, tt.ok)
			}
		})
	}
}
