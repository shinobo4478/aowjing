package generate

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Voice speaks text. Like the video, picture and text generators it is a seam:
// nothing outside this package should know which provider answers.
type Voice interface {
	Speak(ctx context.Context, req VoiceRequest) (VoiceResult, error)
}

// The two request shapes fal's text-to-speech endpoints use. They match the
// families in internal/studio (voices.go); a test in internal/voice keeps the
// two lists in step.
const (
	VoiceFamilyMiniMax    = "minimax"
	VoiceFamilyElevenLabs = "elevenlabs"
)

// VoiceRequest is one piece of text to speak.
type VoiceRequest struct {
	// Model is the fal endpoint id, Family how it is called.
	Model  string
	Family string
	// VoiceID is the MiniMax voice id or the ElevenLabs voice name; empty = the
	// endpoint's default.
	VoiceID string
	Text    string
}

// VoiceResult is where the audio is and how long it is.
type VoiceResult struct {
	// AudioURL is a fal file link (kept at least 7 days) or, in fake mode, a data URI.
	AudioURL string
	// DurationMs is the length of the audio, 0 when the endpoint does not say
	// (ElevenLabs on fal returns no duration).
	DurationMs int
}

const falVoiceMaxWait = 3 * time.Minute

// FalVoice speaks through fal's text-to-speech endpoints.
type FalVoice struct {
	fal *falClient
}

func NewFalVoice(apiKey string) *FalVoice { return &FalVoice{fal: newFalClient(apiKey)} }

// voiceInput builds the JSON body for a family. Only what is needed is sent, so
// fal's own defaults apply to the rest. Thai is asked for explicitly: MiniMax has
// a language_boost option, ElevenLabs a language_code (fal's page does not say
// Thai is accepted there, so a refusal comes back as the endpoint's own error).
func voiceInput(req VoiceRequest) (map[string]any, error) {
	switch req.Family {
	case VoiceFamilyMiniMax:
		in := map[string]any{
			// The MiniMax 2.8 endpoints call the text field "prompt".
			"prompt":         req.Text,
			"language_boost": "Thai",
			// The default is a hex string in the reply; a URL is what a browser can play.
			"output_format": "url",
		}
		if req.VoiceID != "" {
			in["voice_setting"] = map[string]any{"voice_id": req.VoiceID}
		}
		return in, nil
	case VoiceFamilyElevenLabs:
		in := map[string]any{"text": req.Text, "language_code": "th", "output_format": ElevenLabsAudioFormat}
		if req.VoiceID != "" {
			in["voice"] = req.VoiceID
		}
		return in, nil
	}
	return nil, errors.New("unknown voice family " + req.Family)
}

type falVoiceResult struct {
	Audio struct {
		URL string `json:"url"`
	} `json:"audio"`
	DurationMs int `json:"duration_ms"`
}

func (v *FalVoice) Speak(ctx context.Context, req VoiceRequest) (VoiceResult, error) {
	if v.fal.apiKey == "" {
		return VoiceResult{}, errNoFalKey
	}
	if strings.TrimSpace(req.Text) == "" {
		return VoiceResult{}, errors.New("there is no text to speak")
	}
	input, err := voiceInput(req)
	if err != nil {
		return VoiceResult{}, err
	}
	sub, err := v.fal.submit(ctx, req.Model, input)
	if err != nil {
		return VoiceResult{}, err
	}
	if err := v.fal.waitForCompletion(ctx, sub.StatusURL, falVoiceMaxWait); err != nil {
		return VoiceResult{}, err
	}
	var res falVoiceResult
	if err := v.fal.result(ctx, sub.ResponseURL, &res); err != nil {
		return VoiceResult{}, err
	}
	if res.Audio.URL == "" {
		return VoiceResult{}, errors.New("the voice model returned no audio")
	}
	return VoiceResult{AudioURL: res.Audio.URL, DurationMs: res.DurationMs}, nil
}

// FakeVoice stands in for fal during local dev without a key (AI_FAKE_FAL=1): it
// answers with one second of silence as a data URI, so the whole screen can be
// tried at no cost.
type FakeVoice struct{}

func (FakeVoice) Speak(_ context.Context, req VoiceRequest) (VoiceResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return VoiceResult{}, errors.New("there is no text to speak")
	}
	return VoiceResult{AudioURL: silentWAVDataURI(), DurationMs: 1000}, nil
}

// silentWAVDataURI is a one-second, 8 kHz, 8-bit mono WAV of silence.
func silentWAVDataURI() string {
	const rate = 8000
	data := make([]byte, rate)
	for i := range data {
		data[i] = 128 // 8-bit PCM silence
	}
	le32 := func(n int) []byte { return []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)} }
	le16 := func(n int) []byte { return []byte{byte(n), byte(n >> 8)} }
	var b []byte
	b = append(b, "RIFF"...)
	b = append(b, le32(36+len(data))...)
	b = append(b, "WAVEfmt "...)
	b = append(b, le32(16)...)   // fmt chunk size
	b = append(b, le16(1)...)    // PCM
	b = append(b, le16(1)...)    // mono
	b = append(b, le32(rate)...) // sample rate
	b = append(b, le32(rate)...) // byte rate
	b = append(b, le16(1)...)    // block align
	b = append(b, le16(8)...)    // bits per sample
	b = append(b, "data"...)
	b = append(b, le32(len(data))...)
	b = append(b, data...)
	return "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(b)
}

// ElevenLabsAudioFormat is the output we ask ElevenLabs for: a 128 kbps CONSTANT
// bitrate MP3. Constant bitrate is what lets ApproxMP3DurationMs work out the
// length from the size, because fal's reply for ElevenLabs carries no duration.
const (
	ElevenLabsAudioFormat = "mp3_44100_128"
	mp3KbpsForFormat      = 128
)

// ApproxMP3DurationMs is the length of a constant-bitrate MP3 from its size:
// bytes x 8 bits / (kbps x 1000 bits per second). Good to about a tenth of a
// second (a tag or header adds a few bytes); not for variable-bitrate files.
func ApproxMP3DurationMs(sizeBytes int) int {
	if sizeBytes <= 0 {
		return 0
	}
	return sizeBytes * 8 / mp3KbpsForFormat // x 8 bits, / (kbps x 1000), x 1000 ms
}

// maxAudioBytes caps a fetched audio file: a few minutes of speech is well under
// 10 MB, so anything bigger is not what was asked for.
const maxAudioBytes = 10 << 20

// FetchAudio reads the audio a voice returned: a fal file link, or (in fake
// mode) a data URI. It returns the bytes and their content type.
func FetchAudio(ctx context.Context, url string) ([]byte, string, error) {
	if rest, ok := strings.CutPrefix(url, "data:"); ok {
		meta, payload, found := strings.Cut(rest, ",")
		if !found || !strings.HasSuffix(meta, ";base64") {
			return nil, "", errors.New("the audio data URI is not base64")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, "", err
		}
		return data, strings.TrimSuffix(meta, ";base64"), nil
	}
	if !strings.HasPrefix(url, "https://") {
		return nil, "", errors.New("the audio link is not an https link")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching the audio: http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAudioBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxAudioBytes {
		return nil, "", errors.New("the audio file is too large")
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "audio/mpeg"
	}
	return data, ct, nil
}
