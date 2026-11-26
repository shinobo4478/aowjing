package generate

import (
	"encoding/json"
	"testing"
)

func TestImageInput(t *testing.T) {
	refs := []string{"data:image/png;base64,AAAA"}

	tests := []struct {
		name    string
		req     ImageRequest
		want    map[string]any // the JSON body fal will receive
		wantErr bool
	}{
		{
			name: "nano-banana from text: aspect ratio string, jpeg",
			req:  ImageRequest{Model: "nano-banana", Prompt: "a cat", Count: 2, Aspect: "9:16"},
			want: map[string]any{
				"prompt": "a cat", "num_images": 2, "aspect_ratio": "9:16", "output_format": "jpeg",
			},
		},
		{
			name: "nano-banana with references adds image_urls",
			req:  ImageRequest{Model: "nano-banana", Prompt: "a cat", Count: 1, Aspect: "9:16", References: refs},
			want: map[string]any{
				"prompt": "a cat", "num_images": 1, "aspect_ratio": "9:16", "output_format": "jpeg",
				"image_urls": refs,
			},
		},
		{
			name: "seedream takes a pixel size, not a ratio, and no output_format",
			req:  ImageRequest{Model: "seedream", Prompt: "a cat", Count: 3, Aspect: "9:16"},
			want: map[string]any{
				"prompt": "a cat", "num_images": 3,
				"image_size": map[string]any{"width": 1152, "height": 2048},
			},
		},
		{
			name: "seedream with references",
			req:  ImageRequest{Model: "seedream", Prompt: "a cat", Count: 1, Aspect: "9:16", References: refs},
			want: map[string]any{
				"prompt": "a cat", "num_images": 1,
				"image_size": map[string]any{"width": 1152, "height": 2048},
				"image_urls": refs,
			},
		},
		{name: "unknown model", req: ImageRequest{Model: "dalle", Prompt: "x", Count: 1, Aspect: "9:16"}, wantErr: true},
		{name: "seedream unknown aspect", req: ImageRequest{Model: "seedream", Prompt: "x", Count: 1, Aspect: "7:5"}, wantErr: true},
		{name: "empty prompt", req: ImageRequest{Model: "nano-banana", Count: 1, Aspect: "9:16"}, wantErr: true},
		{name: "zero count", req: ImageRequest{Model: "nano-banana", Prompt: "x", Aspect: "9:16"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := imageInput(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			// Compare as JSON, which is what goes over the wire.
			got, _ := json.Marshal(input)
			want, _ := json.Marshal(tt.want)
			if string(got) != string(want) {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}
