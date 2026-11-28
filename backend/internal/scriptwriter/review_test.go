package scriptwriter

import (
	"context"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

func reviewReq() ReviewRequest {
	return ReviewRequest{
		Brief: Brief{VideoStyle: "2d"},
		Shots: []ReviewShot{
			{Position: 1, PicturePrompt: "A plain background with a scale icon in the center. A stomach icon sits in the middle.", MotionPrompt: "The plate slides into the stomach. Static camera, no zoom.", NarrationTh: "อาหารที่ยังย่อยไม่หมด", Seconds: 6},
			{Position: 2, PicturePrompt: "A character looks at a plate.", MotionPrompt: "The character nods.", NarrationTh: "แคลอรี่ ดีฟิซิท คือการกินน้อยกว่าที่ใช้", Seconds: 5},
		},
	}
}

func TestReviewPromptAndSystem(t *testing.T) {
	p := reviewPrompt(reviewReq())
	for _, want := range []string{"Review this video.", "Shot 1 (6 seconds)", "Shot 2 (5 seconds)", "scale icon in the center", "แคลอรี่ ดีฟิซิท"} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	sys := reviewSystem(reviewReq().Brief)
	for _, want := range []string{"FLAT 2D EDUCATIONAL ANIMATION", "strict reviewer", "narration-loanword", "picture-shows-action"} {
		if !strings.Contains(sys, want) {
			t.Errorf("the system prompt lacks %q", want)
		}
	}
}

func TestParseReview(t *testing.T) {
	raw := `Sure: {"findings":[
	  {"shot":2,"kind":"narration-loanword","messageTh":"ขึ้นต้นด้วยคำทับศัพท์","fixTh":"แก้บท"},
	  {"shot":1,"kind":"stray-object","messageTh":"มีไอคอนตาชั่งกลางภาพ","fixTh":"ตัวการ์ตูนยืนบนตาชั่ง"},
	  {"shot":9,"kind":"camera","messageTh":"ช็อตที่ไม่มี","fixTh":"x"},
	  {"shot":1,"kind":"made-up","messageTh":"ชนิดที่ไม่รู้จัก","fixTh":"x"},
	  {"shot":1,"kind":"camera","messageTh":"english only","fixTh":"x"}]}`
	got, err := parseReview(raw, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Unusable findings (no such shot, unknown kind, no Thai message) are dropped; the rest are in shot order.
	if len(got) != 2 || got[0].Shot != 1 || got[1].Shot != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].FixTh == "" || !FixableKind(got[0].Kind) {
		t.Errorf("a prompt finding keeps its fix: %+v", got[0])
	}
	// A narration finding can only be fixed by the owner: any fix the model wrote is dropped.
	if got[1].FixTh != "" || FixableKind(got[1].Kind) {
		t.Errorf("a narration finding has no fix: %+v", got[1])
	}
	if f, err := parseReview(`{"findings": []}`, 2); err != nil || len(f) != 0 {
		t.Errorf("an empty review is valid: %v %v", f, err)
	}
	if _, err := parseReview("sorry, no", 2); err == nil {
		t.Error("a reply with no JSON must be an error")
	}
}

func TestReviewFromRetriesOnceAndAddsCosts(t *testing.T) {
	llm := &scripted{replies: []string{"not json", `{"findings":[]}`}}
	r, err := reviewFrom(context.Background(), llm, "m", reviewReq())
	if err != nil || r.CostUSD != 0.02 || r.Model != "m" || len(llm.prompts) != 2 {
		t.Fatalf("got %+v, %v, %d calls", r, err, len(llm.prompts))
	}
	if _, err := reviewFrom(context.Background(), &scripted{replies: []string{"x", "y"}}, "m", reviewReq()); err == nil {
		t.Error("two bad replies must fail")
	}
}

func TestFakeLLMReviews(t *testing.T) {
	res, err := fakeLLM{}.Complete(context.Background(), generate.LLMRequest{Prompt: reviewPrompt(reviewReq())})
	if err != nil {
		t.Fatal(err)
	}
	f, err := parseReview(res.Text, 2)
	if err != nil || len(f) != 2 {
		t.Fatalf("the fake review is not usable: %v %+v", err, f)
	}
}
