package scriptwriter

import (
	"context"
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

const goodRevision = `{"picturePrompt":"A flat character stands on a flat scale, a stomach icon on its belly.","picturePromptTh":"ตัวการ์ตูนยืนบนตาชั่ง มีไอคอนกระเพาะที่ท้อง","endPicturePrompt":"","endPicturePromptTh":"","motionPrompt":"The food icon slides out along an arrow. Static camera, no zoom.","motionPromptTh":"ไอคอนอาหารเลื่อนออกตามลูกศร กล้องนิ่ง ไม่ซูม"}`

func reviseReq() ReviseRequest {
	return ReviseRequest{
		Instruction: "ตัวการ์ตูนยืนบนตาชั่ง มีกระเพาะบนตัว",
		Brief:       Brief{Mode: ModeExplainer, VideoStyle: "2d"},
		Shot: ShotText{
			PicturePrompt: "A plain flat pastel background with a scale icon in the center. A stomach icon sits in the middle.",
			MotionPrompt:  "The plate icon slides into the stomach. Static camera, no zoom.",
			NarrationTh:   "อาหารที่ยังย่อยไม่หมดก็มีน้ำหนักอยู่ในตัวเรา",
		},
		Previous: "Shot before.", Next: "Shot after.", HasNext: true,
	}
}

func TestReviseValidate(t *testing.T) {
	for _, c := range []struct {
		in   string
		bad  bool
		name string
	}{{"ab", true, "too short"}, {"  ตัวการ์ตูนยืนบนตาชั่ง ", false, "ok"}, {strings.Repeat("ก", 601), true, "too long"}} {
		r := ReviseRequest{Instruction: c.in}
		if got := len(r.Validate()) > 0; got != c.bad {
			t.Errorf("%s: invalid = %v, want %v", c.name, got, c.bad)
		}
	}
}

func TestRevisePromptHasWhatTheModelNeeds(t *testing.T) {
	p := revisePrompt(reviseReq())
	for _, want := range []string{
		"Revise this one shot.", "ตัวการ์ตูนยืนบนตาชั่ง มีกระเพาะบนตัว",
		"scale icon in the center", "The plate icon slides", "อาหารที่ยังย่อยไม่หมด",
		"Shot before.", "Shot after.",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
	last := reviseReq()
	last.HasNext = false
	if !strings.Contains(revisePrompt(last), "LAST shot") {
		t.Error("the last shot must be told it has no end picture")
	}
	// The studio's rules are in the system prompt, so a revision follows them too.
	sys := reviseSystem(reviseReq().Brief)
	if !strings.Contains(sys, "FLAT 2D EDUCATIONAL ANIMATION") || !strings.Contains(sys, "REWRITE ONE EXISTING SHOT") {
		t.Error("the system prompt must hold the style rules and the revise task")
	}
}

func TestParseRevision(t *testing.T) {
	r, err := parseRevision("Here you go: "+goodRevision, true)
	if err != nil || r.PicturePromptTh == "" || r.EndPicturePrompt != "" {
		t.Fatalf("got %+v, %v", r, err)
	}
	for _, c := range []struct{ name, raw, want string }{
		{"no json", "sorry", "JSON"},
		{"empty picture", `{"picturePrompt":"","motionPrompt":"x"}`, "empty"},
		{"thai in english", `{"picturePrompt":"ตัวการ์ตูน","picturePromptTh":"ก","motionPrompt":"m","motionPromptTh":"ก"}`, "Thai letters"},
		{"missing twin", `{"picturePrompt":"p","picturePromptTh":"","motionPrompt":"m","motionPromptTh":"ก"}`, "no Thai text"},
	} {
		if _, err := parseRevision(c.raw, true); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
	// The last shot has no next shot to meet, so an end picture is dropped.
	withEnd := strings.Replace(goodRevision, `"endPicturePrompt":"","endPicturePromptTh":""`, `"endPicturePrompt":"Only the arrow moves.","endPicturePromptTh":"เฉพาะลูกศรที่ขยับ"`, 1)
	if r, err := parseRevision(withEnd, false); err != nil || r.EndPicturePrompt != "" {
		t.Errorf("the last shot kept an end picture: %+v, %v", r, err)
	}
	if r, err := parseRevision(withEnd, true); err != nil || r.EndPicturePrompt == "" {
		t.Errorf("a chained shot lost its end picture: %+v, %v", r, err)
	}
}

func TestReviseFromRetriesOnceAndAddsCosts(t *testing.T) {
	llm := &scripted{replies: []string{"not json", goodRevision}}
	r, err := reviseFrom(context.Background(), llm, "m", reviseReq())
	if err != nil || r.CostUSD != 0.02 || r.Model != "m" || len(llm.prompts) != 2 {
		t.Fatalf("got %+v, %v, %d calls", r, err, len(llm.prompts))
	}
	if !strings.Contains(llm.prompts[1], "could not be used") {
		t.Error("the retry must say what was wrong")
	}
	if _, err := reviseFrom(context.Background(), &scripted{replies: []string{"x", "y"}}, "m", reviseReq()); err == nil {
		t.Error("two bad replies must fail")
	}
}

func TestFakeLLMRevisesAShot(t *testing.T) {
	res, err := fakeLLM{}.Complete(context.Background(), generate.LLMRequest{Prompt: revisePrompt(reviseReq())})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseRevision(res.Text, true); err != nil {
		t.Fatalf("the fake revision is not usable: %v", err)
	}
}
