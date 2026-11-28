package generate

import (
	"context"
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
	want := map[string]any{"text": "สวัสดี", "language_code": "th", "voice": "Rachel", "output_format": "mp3_44100_128"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("elevenlabs body:\n got %v\nwant %v", got, want)
	}
}

func TestVoiceInputUnknownFamily(t *testing.T) {
	if _, err := voiceInput(VoiceRequest{Family: "nope", Text: "x"}); err == nil {
		t.Error("an unknown family must be an error")
	}
}

func TestApproxMP3DurationMs(t *testing.T) {
	// 128 kbps = 16000 bytes per second.
	for size, want := range map[int]int{0: 0, -5: 0, 16000: 1000, 80000: 5000, 160000: 10000, 8000: 500} {
		if got := ApproxMP3DurationMs(size); got != want {
			t.Errorf("ApproxMP3DurationMs(%d) = %d, want %d", size, got, want)
		}
	}
}

func TestFetchAudioDataURI(t *testing.T) {
	data, ct, err := FetchAudio(context.Background(), silentWAVDataURI())
	if err != nil || ct != "audio/wav" || len(data) < 8000 || string(data[:4]) != "RIFF" {
		t.Fatalf("fake audio: %d bytes, %q, %v", len(data), ct, err)
	}
	for _, bad := range []string{"data:audio/wav,notbase64", "http://x/y.mp3", "ftp://x", "", "data:audio/wav;base64,!!!"} {
		if _, _, err := FetchAudio(context.Background(), bad); err == nil {
			t.Errorf("FetchAudio(%q) should fail", bad)
		}
	}
}
