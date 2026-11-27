package server

import (
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

func TestLivePriceCheck(t *testing.T) {
	veo := studio.ModelFor("veo-3.1")
	prices := map[string]float64{
		"fal-ai/veo3.1":                0.40,
		"fal-ai/veo3.1/image-to-video": 0.40,
	}
	if got := liveFor(veo, prices); got != 0.40 {
		t.Fatalf("liveFor = %v, want 0.40", got)
	}
	if changed(veo, 0.40) {
		t.Error("an unchanged price must not be flagged")
	}
	if !changed(veo, 0.50) {
		t.Error("a raised price must be flagged")
	}
	if changed(veo, 0) {
		t.Error("an unknown live price (0) must not be flagged")
	}
	// The top price among a model's endpoints wins.
	prices["fal-ai/veo3.1/first-last-frame-to-video"] = 0.45
	if got := liveFor(veo, prices); got != 0.45 {
		t.Errorf("liveFor with a dearer endpoint = %v, want 0.45", got)
	}
}
