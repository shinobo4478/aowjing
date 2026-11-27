package generate

import (
	"encoding/json"
	"testing"
)

func TestFalPlan(t *testing.T) {
	g := NewFalVideoGenerator("key", FalVideoSpec{Wire: "kling", TextModel: "fal/text-model", ImageModel: "fal/image-model", EndPicture: true})

	tests := []struct {
		name      string
		req       Request
		wantModel string
		want      map[string]any // the JSON body fal will receive
	}{
		{
			name:      "no image uses text-to-video, prompt only",
			req:       Request{Prompt: "a cat"},
			wantModel: "fal/text-model",
			want:      map[string]any{"prompt": "a cat"},
		},
		{
			name: "an end picture is sent as end_image_url",
			req: Request{
				Prompt: "pounce", ImageURL: "https://example.com/a.jpg",
				EndImageURL: "https://example.com/b.jpg", Seconds: 4,
			},
			wantModel: "fal/image-model",
			want: map[string]any{
				"prompt":          "pounce",
				"start_image_url": "https://example.com/a.jpg",
				"end_image_url":   "https://example.com/b.jpg",
				"duration":        "4",
				"generate_audio":  false,
			},
		},
		{
			name:      "an end picture without a start picture is ignored (text-to-video)",
			req:       Request{Prompt: "a cat", EndImageURL: "https://example.com/b.jpg"},
			wantModel: "fal/text-model",
			want:      map[string]any{"prompt": "a cat"},
		},
		{
			name:      "an explicit length is sent as a string",
			req:       Request{Prompt: "slow zoom", ImageURL: "https://example.com/p.jpg", Seconds: 8},
			wantModel: "fal/image-model",
			want: map[string]any{
				"prompt":          "slow zoom",
				"start_image_url": "https://example.com/p.jpg",
				"duration":        "8",
				"generate_audio":  false,
			},
		},
		{
			name:      "image uses image-to-video, silent, duration as a string",
			req:       Request{Prompt: "slow zoom", ImageURL: "https://example.com/p.jpg"},
			wantModel: "fal/image-model",
			want: map[string]any{
				"prompt":          "slow zoom",
				"start_image_url": "https://example.com/p.jpg",
				"duration":        "5",
				"generate_audio":  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, input, err := g.plan(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			// Compare as JSON, which is what actually goes over the wire:
			// it shows duration is a string and generate_audio a real false.
			got, _ := json.Marshal(input)
			want, _ := json.Marshal(tt.want)
			if string(got) != string(want) {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}

func TestFalPlanVeo(t *testing.T) {
	full := NewFalVideoGenerator("key", FalVideoSpec{
		Wire: "veo", TextModel: "fal/veo", ImageModel: "fal/veo-i2v", FirstLastModel: "fal/veo-flf",
		EndPicture: true, DefaultSeconds: 4,
	})
	fast := NewFalVideoGenerator("key", FalVideoSpec{Wire: "veo", ImageModel: "fal/veo-fast-i2v", DefaultSeconds: 4})
	withSound := NewFalVideoGenerator("key", FalVideoSpec{Wire: "veo", ImageModel: "fal/veo-i2v", DefaultSeconds: 4, Audio: true})

	tests := []struct {
		name      string
		g         *FalVideoGenerator
		req       Request
		wantModel string
		want      map[string]any
		wantErr   bool
	}{
		{
			name:      "a picture uses image_url and a length like 4s, 9:16 at 720p, silent",
			g:         full,
			req:       Request{Prompt: "slow zoom", ImageURL: "https://example.com/p.jpg", Seconds: 6},
			wantModel: "fal/veo-i2v",
			want: map[string]any{
				"prompt": "slow zoom", "image_url": "https://example.com/p.jpg",
				"aspect_ratio": "9:16", "duration": "6s", "resolution": "720p", "generate_audio": false,
			},
		},
		{
			name:      "no length uses the model default",
			g:         full,
			req:       Request{Prompt: "x", ImageURL: "https://example.com/p.jpg"},
			wantModel: "fal/veo-i2v",
			want: map[string]any{
				"prompt": "x", "image_url": "https://example.com/p.jpg",
				"aspect_ratio": "9:16", "duration": "4s", "resolution": "720p", "generate_audio": false,
			},
		},
		{
			name:      "an end picture switches to the first-last-frame endpoint",
			g:         full,
			req:       Request{Prompt: "pounce", ImageURL: "https://example.com/a.jpg", EndImageURL: "https://example.com/b.jpg", Seconds: 8},
			wantModel: "fal/veo-flf",
			want: map[string]any{
				"prompt": "pounce", "first_frame_url": "https://example.com/a.jpg", "last_frame_url": "https://example.com/b.jpg",
				"aspect_ratio": "9:16", "duration": "8s", "resolution": "720p", "generate_audio": false,
			},
		},
		{
			name:      "text-only works on a model with a text endpoint",
			g:         full,
			req:       Request{Prompt: "a cat", Seconds: 4},
			wantModel: "fal/veo",
			want: map[string]any{
				"prompt": "a cat", "aspect_ratio": "9:16", "duration": "4s", "resolution": "720p", "generate_audio": false,
			},
		},
		{
			name:      "the sound toggle is passed on",
			g:         withSound,
			req:       Request{Prompt: "x", ImageURL: "https://example.com/p.jpg", Seconds: 4},
			wantModel: "fal/veo-i2v",
			want: map[string]any{
				"prompt": "x", "image_url": "https://example.com/p.jpg",
				"aspect_ratio": "9:16", "duration": "4s", "resolution": "720p", "generate_audio": true,
			},
		},
		{name: "text-only on a model without a text endpoint is refused", g: fast, req: Request{Prompt: "a cat"}, wantErr: true},
		{
			name:    "an end picture on a model without the endpoint is refused, not dropped",
			g:       fast,
			req:     Request{Prompt: "x", ImageURL: "https://example.com/a.jpg", EndImageURL: "https://example.com/b.jpg"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, input, err := tt.g.plan(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want an error, got model %q", model)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			got, _ := json.Marshal(input)
			want, _ := json.Marshal(tt.want)
			if string(got) != string(want) {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}
