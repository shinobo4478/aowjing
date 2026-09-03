package generate

import (
	"context"
	"time"
)

// FakeFalGenerator stands in for FalVideoGenerator during local dev when there
// is no fal.ai key. It waits a few seconds (so the pending -> succeeded flow is
// visible) then returns a well-known public sample video. Enabled via
// AI_FAKE_FAL=1.
type FakeFalGenerator struct{}

func (FakeFalGenerator) Name() string { return "fal" }

const sampleVideoURL = "https://storage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"

func (FakeFalGenerator) Generate(ctx context.Context, _ string) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-time.After(3 * time.Second):
	}
	return Result{
		Output:   sampleVideoURL,
		Kind:     "video",
		Provider: "fal",
		Model:    "fake-kling",
	}, nil
}
