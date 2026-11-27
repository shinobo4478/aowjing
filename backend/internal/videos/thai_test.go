package videos

import (
	"strings"
	"testing"
)

func TestShotInputThaiPrompts(t *testing.T) {
	in := shotInput{
		PromptTh:           "  แมวกระโจนใส่ถุง  ",
		PicturePromptTh:    "แมวนั่งกลางพรม",
		EndPicturePromptTh: "",
	}
	if errs := in.normalizeAndValidate(); len(errs) != 0 {
		t.Fatalf("Thai prompts were refused: %v", errs)
	}
	if in.PromptTh != "แมวกระโจนใส่ถุง" {
		t.Errorf("Thai text should be trimmed, got %q", in.PromptTh)
	}

	// Length is counted in characters, not bytes: 2000 Thai letters (6000 bytes) fit,
	// 2001 do not.
	fits := shotInput{PromptTh: strings.Repeat("ก", 2000)}
	if errs := fits.normalizeAndValidate(); errs["promptTh"] != "" {
		t.Errorf("2000 Thai characters should fit: %v", errs)
	}
	for name, in := range map[string]shotInput{
		"promptTh":           {PromptTh: strings.Repeat("ก", 2001)},
		"picturePromptTh":    {PicturePromptTh: strings.Repeat("ก", 2001)},
		"endPicturePromptTh": {EndPicturePromptTh: strings.Repeat("ก", 2001)},
	} {
		in := in
		if errs := in.normalizeAndValidate(); errs[name] == "" {
			t.Errorf("want an error on %q, got %v", name, errs)
		}
	}
}
