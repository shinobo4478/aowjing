package server

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
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
	// ConfirmedSilent / ConfirmedAudio: a real bill matched that price.
	// Otherwise it is fal's list price.
	ConfirmedSilent bool `json:"confirmedSilent"`
	ConfirmedAudio  bool `json:"confirmedAudio"`
	// LiveUSDPerSecond is what fal's pricing API lists for the model right now:
	// ONE number per endpoint, the top of its range (the price with sound), so it
	// is a ceiling for a silent clip. null = fal could not be asked (no key, or
	// it did not answer).
	LiveUSDPerSecond *float64 `json:"liveUsdPerSecond"`
	// PriceChanged: the live price differs from the one in our table (the one
	// with sound, which is what the live number is). The estimate on screen may
	// then be wrong, and the screen says so.
	PriceChanged bool `json:"priceChanged"`
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

// livePrices asks fal what each model costs and remembers the answer, so the
// screen's price check does not call fal on every page load.
type livePrices struct {
	mu     sync.Mutex
	at     time.Time
	ok     bool
	prices map[string]float64 // fal endpoint id -> USD per second
	key    string             // the API key it was fetched with
}

const (
	livePricesFor     = 10 * time.Minute
	livePricesRetry   = 1 * time.Minute // after a failure, do not hammer fal
	livePricesWaitFor = 6 * time.Second
)

// modelEndpoints lists every fal endpoint a model uses.
func modelEndpoints(m studio.VideoModel) []string {
	var out []string
	for _, id := range []string{m.TextModel, m.ImageModel, m.FirstLastModel} {
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func (s *Server) livePrices(ctx context.Context) map[string]float64 {
	key, err := s.q.GetSetting(ctx, "falApiKey")
	key = strings.TrimSpace(key)
	if err != nil || key == "" {
		return nil
	}
	lp := &s.live
	lp.mu.Lock()
	defer lp.mu.Unlock()
	age := time.Since(lp.at)
	if lp.key == key && ((lp.ok && age < livePricesFor) || (!lp.ok && age < livePricesRetry)) {
		return lp.prices
	}

	var ids []string
	for _, m := range studio.Models() {
		ids = append(ids, modelEndpoints(m)...)
	}
	ctx, cancel := context.WithTimeout(ctx, livePricesWaitFor)
	defer cancel()
	prices, err := generate.NewFalAccount(key).ModelPrices(ctx, ids)
	lp.at, lp.key, lp.ok = time.Now(), key, err == nil
	if err != nil {
		lp.prices = nil
	} else {
		lp.prices = prices
	}
	return lp.prices
}

// liveFor is the top live price among a model's endpoints (0 = not known).
func liveFor(m studio.VideoModel, prices map[string]float64) float64 {
	top := 0.0
	for _, id := range modelEndpoints(m) {
		top = math.Max(top, prices[id])
	}
	return top
}

// changed reports whether fal's live price differs from ours by more than
// rounding. Our table holds the price with sound, which is what fal lists.
func changed(m studio.VideoModel, live float64) bool {
	return live > 0 && m.USDPerSecondAudio > 0 && math.Abs(live-m.USDPerSecondAudio) > 0.0005
}

// handleStudioPricing tells the screen what each video model takes and costs,
// so the confirmation before spending shows a number from one place (the model
// registry) instead of a copy in the frontend. It also asks fal what the models
// cost today, so a price rise shows up before a clip is paid for, not after.
func (s *Server) handleStudioPricing(w http.ResponseWriter, r *http.Request) {
	live := s.livePrices(r.Context())
	out := []pricingModel{}
	for _, m := range studio.Models() {
		lv := liveFor(m, live)
		out = append(out, pricingModel{
			Key: string(m.Key), Label: m.Label,
			Seconds: m.Seconds, DefaultSeconds: m.DefaultSeconds,
			USDPerSecond: known(m.USDPerSecond), USDPerSecondAudio: known(m.USDPerSecondAudio),
			ConfirmedSilent: m.ConfirmedSilent, ConfirmedAudio: m.ConfirmedAudio,
			LiveUSDPerSecond: known(lv), PriceChanged: changed(m, lv),
			SupportsText: m.SupportsText(), EndPicture: m.EndPicture,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"models": out})
}
