package server

import (
	"encoding/json"
	"net/http"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// handleStudioPricing tells the screen what a clip costs, so the confirmation
// before spending shows a number from one place (the recipes) instead of a
// copy in the frontend. A style whose price is not known is left out of
// imageUsdPerSecond, and the screen says so rather than guessing. The screen
// multiplies by the clip's length.
func (s *Server) handleStudioPricing(w http.ResponseWriter, _ *http.Request) {
	perSecond := map[string]float64{}
	for _, r := range studio.Recipes() {
		if usd := r.ImageClipUSD(1); usd > 0 {
			perSecond[string(r.Style)] = usd
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"defaultSeconds":    studio.DefaultClipSeconds,
		"minSeconds":        studio.MinClipSeconds,
		"maxSeconds":        studio.MaxClipSeconds,
		"imageUsdPerSecond": perSecond,
	})
}
