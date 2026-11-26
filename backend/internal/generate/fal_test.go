package generate

import (
	"encoding/json"
	"testing"
)

func TestFalPlan(t *testing.T) {
	g := NewFalVideoGenerator("key", "fal/text-model", "fal/image-model")

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
			model, input := g.plan(tt.req)
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
