package scriptwriter

import (
	"context"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

func TestParseThaiTwins(t *testing.T) {
	const raw = `{"title":"t","scene":"S.","sceneTh":"ฉาก.","shots":[
 {"picturePrompt":"a","picturePromptTh":"ภาพ ก","endPicturePrompt":"Only x.","endPicturePromptTh":"เฉพาะ x","motionPrompt":"m1","motionPromptTh":"ขยับ 1"},
 {"picturePrompt":"b","picturePromptTh":"ภาพ ข","endPicturePrompt":"","endPicturePromptTh":"ไม่ควรมี","motionPrompt":"m2","motionPromptTh":"ขยับ 2"},
 {"picturePrompt":"c","picturePromptTh":"ภาพ ค","endPicturePrompt":"Only y.","endPicturePromptTh":"เฉพาะ y","motionPrompt":"m3","motionPromptTh":"ขยับ 3"}]}`
	d, err := parse(raw, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	// The Thai picture prompt gets the Thai scene in front, like the English one.
	if d.Shots[0].PicturePromptTh != "ฉาก. ภาพ ก" {
		t.Errorf("Thai picture prompt = %q", d.Shots[0].PicturePromptTh)
	}
	if d.Shots[0].MotionPromptTh != "ขยับ 1" || d.Shots[0].EndPicturePromptTh != "เฉพาะ x" {
		t.Errorf("shot 1 Thai = %+v", d.Shots[0])
	}
	// A Thai end prompt with no English twin (a cut) is dropped, and nothing
	// follows the last shot.
	if d.Shots[1].EndPicturePromptTh != "" || d.Shots[2].EndPicturePromptTh != "" {
		t.Errorf("stray Thai end prompts kept: %q / %q", d.Shots[1].EndPicturePromptTh, d.Shots[2].EndPicturePromptTh)
	}

	// An older-style reply with no Thai at all still parses, with empty twins.
	d, err = parse(goodReply, 2, 4)
	if err != nil || d.Shots[0].MotionPromptTh != "" || d.Shots[0].PicturePromptTh != "" {
		t.Errorf("a reply without Thai twins should parse with them empty: %v %+v", err, d.Shots)
	}

	// A Thai twin over the length limit is dropped, not fatal.
	long := `{"title":"t","scene":"S.","sceneTh":"ฉาก.","shots":[{"picturePrompt":"a","picturePromptTh":"` +
		strings.Repeat("ก", 2000) + `","motionPrompt":"m","motionPromptTh":"ขยับ"}]}`
	d, err = parse(long, 1, 4)
	if err != nil || d.Shots[0].PicturePromptTh != "" || d.Shots[0].MotionPromptTh != "ขยับ" {
		t.Errorf("an over-long Thai twin should be dropped: %v %+v", err, d.Shots)
	}
}

func TestSystemPromptAsksForThaiTwinsAndItsExampleHasThem(t *testing.T) {
	for _, want := range []string{`"sceneTh"`, `"picturePromptTh"`, `"motionPromptTh"`, `"endPicturePromptTh"`, "Thai twin"} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("the system prompt lost %q", want)
		}
	}
	start := strings.Index(systemPrompt, `{"title":"The bag drop"`)
	d, err := parse(systemPrompt[start:], 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range d.Shots {
		if !hasThai(s.PicturePromptTh) || !hasThai(s.MotionPromptTh) {
			t.Errorf("the worked example's shot %d has no Thai twins: %+v", i+1, s)
		}
	}
	if !hasThai(d.Shots[0].EndPicturePromptTh) || d.Shots[1].EndPicturePromptTh != "" {
		t.Errorf("the example should have a Thai end prompt only where there is an English one: %+v", d.Shots)
	}
}

func TestFakeDraftHasThaiTwins(t *testing.T) {
	res, _ := fakeLLM{}.Complete(context.Background(), generate.LLMRequest{Prompt: userPrompt(Brief{Topic: "x y z", Shots: 2, Seconds: 4})})
	d, err := parse(res.Text, 2, 4)
	if err != nil || !hasThai(d.Shots[0].MotionPromptTh) || !hasThai(d.Shots[0].PicturePromptTh) {
		t.Errorf("fake draft should carry Thai twins: %v %+v", err, d.Shots)
	}
}
