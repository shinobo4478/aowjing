package server

import (
	"encoding/json"
	"net/http"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// handleStudioPricing tells the screen what a clip costs, so the confirmation
// before spending shows a number from one place (the recipes) instead of a
// copy in the frontend. A style whose price is not known is left out of
// imageClipUsd, and the screen says so rather than guessing.
func (s *Server) handleStudioPricing(w http.ResponseWriter, _ *http.Request) {
	imageClip := map[string]float64{}
	for _, r := range studio.Recipes() {
		if usd := r.ImageClipUSD(); usd > 0 {
			imageClip[string(r.Style)] = usd
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"clipSeconds":  studio.ClipSeconds,
		"imageClipUsd": imageClip,
	})
}
