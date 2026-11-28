package videos

import (
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

func TestToVoiceDTO(t *testing.T) {
	shot := sqlc.VideoShot{NarrationTh: "สวัสดี"}
	if got := toVoiceDTO(shot, nil); got != nil {
		t.Errorf("a shot with no audio has no voice, got %+v", got)
	}
	audio := &sqlc.ListShotAudioMetaRow{Voice: "eleven-v3-rachel", Text: "สวัสดี", DurationMs: 4200, SizeBytes: 67200}
	got := toVoiceDTO(shot, audio)
	if got == nil || !got.Current || got.DurationMs != 4200 || got.Voice != "eleven-v3-rachel" {
		t.Fatalf("matching narration: %+v", got)
	}
	// The narration changed after the audio was made: the audio is out of date.
	shot.NarrationTh = "สวัสดีทุกคน"
	if got := toVoiceDTO(shot, audio); got == nil || got.Current {
		t.Errorf("edited narration must mark the audio not current: %+v", got)
	}
}

func TestNarrationLimitCountsCharacters(t *testing.T) {
	in := shotInput{NarrationTh: strings.Repeat("ก", studio.MaxVoiceSampleChars)} // chars, not bytes
	if errs := in.normalizeAndValidate(); errs["narrationTh"] != "" {
		t.Errorf("%d Thai characters must be accepted, got %v", studio.MaxVoiceSampleChars, errs)
	}
	in = shotInput{NarrationTh: strings.Repeat("ก", studio.MaxVoiceSampleChars+1)}
	if errs := in.normalizeAndValidate(); errs["narrationTh"] == "" {
		t.Error("one character over the limit must be refused")
	}
	in = shotInput{NarrationTh: "  พูดช้าๆ  "}
	in.normalizeAndValidate()
	if in.NarrationTh != "พูดช้าๆ" {
		t.Errorf("narration should be trimmed, got %q", in.NarrationTh)
	}
}
