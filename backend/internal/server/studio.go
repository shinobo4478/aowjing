package server

import (
	"encoding/json"
	"net/http"

	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// pricingModel is one video model as the screen needs it: which lengths it
// takes and what a second costs. A price that is not known is left out (null),
// and the screen says so rather than guessing; the screen multiplies by the
// clip's length.
type pricingModel struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Seconds        []int  `json:"seconds"`
	DefaultSeconds int    `json:"defaultSeconds"`
	// USD per second of a silent clip / a clip with sound; null = not known.
	USDPerSecond      *float64 `json:"usdPerSecond"`
	USDPerSecondAudio *float64 `json:"usdPerSecondAudio"`
	// ConfirmedSilent: a real bill matched the silent price. Otherwise it is
	// fal's list price.
	ConfirmedSilent bool `json:"confirmedSilent"`
	// SupportsText: a clip can be made from a prompt alone. EndPicture: a clip
	// can be given an end picture.
	SupportsText bool `json:"supportsText"`
	EndPicture   bool `json:"endPicture"`
}

func known(v float64) *float64 {
	if v > 0 {
		return &v
	}
	return nil
}

// handleStudioPricing tells the screen what each video model takes and costs,
// so the confirmation before spending shows a number from one place (the model
// registry) instead of a copy in the frontend.
func (s *Server) handleStudioPricing(w http.ResponseWriter, _ *http.Request) {
	out := []pricingModel{}
	for _, m := range studio.Models() {
		out = append(out, pricingModel{
			Key: string(m.Key), Label: m.Label,
			Seconds: m.Seconds, DefaultSeconds: m.DefaultSeconds,
			USDPerSecond: known(m.USDPerSecond), USDPerSecondAudio: known(m.USDPerSecondAudio),
			ConfirmedSilent: m.ConfirmedSilent,
			SupportsText:    m.SupportsText(), EndPicture: m.EndPicture,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"models": out})
}
