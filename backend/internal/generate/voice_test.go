package generate

import (
	"reflect"
	"testing"
)

func TestVoiceInputMiniMax(t *testing.T) {
	got, err := voiceInput(VoiceRequest{Family: VoiceFamilyMiniMax, VoiceID: "Thai_female_1_sample1", Text: "สวัสดี"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"prompt":         "สวัสดี", // the 2.8 endpoints call the text field "prompt"
		"language_boost": "Thai",
		"output_format":  "url", // the default is hex, which a browser cannot play
		"voice_setting":  map[string]any{"voice_id": "Thai_female_1_sample1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("minimax body:\n got %v\nwant %v", got, want)
	}
	// No voice id: the endpoint's own default voice, so no voice_setting at all.
	got, _ = voiceInput(VoiceRequest{Family: VoiceFamilyMiniMax, Text: "x"})
	if _, has := got["voice_setting"]; has {
		t.Errorf("no voice id must send no voice_setting: %v", got)
	}
}

func TestVoiceInputElevenLabs(t *testing.T) {
	got, err := voiceInput(VoiceRequest{Family: VoiceFamilyElevenLabs, VoiceID: "Rachel", Text: "สวัสดี"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"text": "สวัสดี", "language_code": "th", "voice": "Rachel"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("elevenlabs body:\n got %v\nwant %v", got, want)
	}
}

func TestVoiceInputUnknownFamily(t *testing.T) {
	if _, err := voiceInput(VoiceRequest{Family: "nope", Text: "x"}); err == nil {
		t.Error("an unknown family must be an error")
	}
}
