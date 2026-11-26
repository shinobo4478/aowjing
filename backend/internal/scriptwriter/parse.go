package scriptwriter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxPromptRunes keeps a drafted prompt inside what a shot accepts (2000), with
// room for the scene line that is put in front of every picture prompt.
const maxPromptRunes = 1900

// Brief is what the owner asks for.
type Brief struct {
	Topic   string
	Shots   int
	Seconds int // the length of each shot

	// What the profile says about itself, so the shots fit its channel.
	ProfileName string
	Niche       string
	Description string
}

// Shot is one drafted shot, ready to put into a Video.
type Shot struct {
	// PicturePrompt already has the scene in front of it.
	PicturePrompt string
	MotionPrompt  string
	Seconds       int
}

// Draft is a drafted video.
type Draft struct {
	Title   string
	Scene   string
	Shots   []Shot
	CostUSD float64 // what the LLM call cost, as fal reported it
	Model   string
}

// reply is the JSON shape the model was asked for.
type reply struct {
	Title string `json:"title"`
	Scene string `json:"scene"`
	Shots []struct {
		PicturePrompt string `json:"picturePrompt"`
		MotionPrompt  string `json:"motionPrompt"`
	} `json:"shots"`
}

// parse turns the model's text into a Draft. Models often wrap JSON in prose or
// a code fence despite being told not to, so it takes the outermost {...}. It
// refuses a reply with too few shots or an empty prompt rather than filling a
// video with gaps, and drops extra shots. Every picture prompt gets the scene in
// front of it, so the pictures of one video describe the same place.
func parse(raw string, want, seconds int) (Draft, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return Draft{}, errors.New("the model did not answer with JSON")
	}
	var r reply
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return Draft{}, fmt.Errorf("the model's JSON could not be read: %w", err)
	}

	scene := strings.TrimSpace(r.Scene)
	if scene == "" {
		return Draft{}, errors.New("the model's draft has no scene")
	}
	if len(r.Shots) < want {
		return Draft{}, fmt.Errorf("the model wrote %d shots, not the %d asked for", len(r.Shots), want)
	}

	d := Draft{Title: strings.TrimSpace(r.Title), Scene: scene}
	for i := 0; i < want; i++ {
		pic := strings.TrimSpace(r.Shots[i].PicturePrompt)
		motion := strings.TrimSpace(r.Shots[i].MotionPrompt)
		if pic == "" || motion == "" {
			return Draft{}, fmt.Errorf("shot %d of the draft is missing a prompt", i+1)
		}
		full := scene + " " + pic
		if utf8.RuneCountInString(full) > maxPromptRunes || utf8.RuneCountInString(motion) > maxPromptRunes {
			return Draft{}, fmt.Errorf("shot %d of the draft is too long", i+1)
		}
		d.Shots = append(d.Shots, Shot{PicturePrompt: full, MotionPrompt: motion, Seconds: seconds})
	}
	return d, nil
}
