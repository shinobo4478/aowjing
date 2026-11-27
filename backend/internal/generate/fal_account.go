package generate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FalCredits is the account's prepaid credit balance.
type FalCredits struct {
	Balance  float64
	Currency string
}

// FalAccount reads account facts from fal's Platform API (api.fal.ai/v1), which
// is a different API from the model queue. The credit balance needs a key on
// the BILLING preset (or READONLY/FULL); an ordinary API-preset key is not
// documented to reach it.
type FalAccount struct {
	fal     *falClient
	baseURL string
}

func NewFalAccount(apiKey string) *FalAccount {
	c := newFalClient(apiKey)
	c.http.Timeout = 10 * time.Second
	return &FalAccount{fal: c, baseURL: "https://api.fal.ai/v1"}
}

// Credits returns the current balance (GET /account/billing?expand=credits).
func (a *FalAccount) Credits(ctx context.Context) (FalCredits, error) {
	if a.fal.apiKey == "" {
		return FalCredits{}, errors.New("no billing key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		a.baseURL+"/account/billing?expand=credits", nil)
	if err != nil {
		return FalCredits{}, err
	}
	req.Header.Set("Authorization", "Key "+a.fal.apiKey)

	var out struct {
		Credits *struct {
			CurrentBalance float64 `json:"current_balance"`
			Currency       string  `json:"currency"`
		} `json:"credits"`
	}
	if err := a.fal.do(req, &out); err != nil {
		return FalCredits{}, fmt.Errorf("fal billing: %w", err)
	}
	if out.Credits == nil {
		return FalCredits{}, errors.New("fal billing: no credits in the reply")
	}
	return FalCredits{Balance: out.Credits.CurrentBalance, Currency: out.Credits.Currency}, nil
}

// BillingCosts returns what fal actually charged, in USD, for each of the given
// request ids it has a billing event for (GET /models/billing-events). Ids it
// does not know yet are simply absent: billing can lag the job by a while, so
// the caller asks again later. At most 50 ids per call (fal's limit); `since`
// bounds the search (fal defaults to the last 24 h and caps a range at 90 days).
func (a *FalAccount) BillingCosts(ctx context.Context, requestIDs []string, since time.Time) (map[string]float64, error) {
	if a.fal.apiKey == "" {
		return nil, errors.New("no billing key")
	}
	costs := map[string]float64{}
	for len(requestIDs) > 0 {
		n := min(len(requestIDs), 50)
		batch := requestIDs[:n]
		requestIDs = requestIDs[n:]

		cursor := ""
		for {
			q := url.Values{}
			q.Set("request_id", strings.Join(batch, ","))
			q.Set("start", since.UTC().Format(time.RFC3339))
			q.Set("limit", "100")
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet,
				a.baseURL+"/models/billing-events?"+q.Encode(), nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Key "+a.fal.apiKey)

			var out struct {
				HasMore       bool   `json:"has_more"`
				NextCursor    string `json:"next_cursor"`
				BillingEvents []struct {
					RequestID string  `json:"request_id"`
					CostTotal float64 `json:"cost_total"`
				} `json:"billing_events"`
			}
			if err := a.fal.do(req, &out); err != nil {
				return nil, fmt.Errorf("fal billing events: %w", err)
			}
			// A request can have more than one event; its cost is their sum.
			for _, ev := range out.BillingEvents {
				costs[ev.RequestID] += ev.CostTotal
			}
			if !out.HasMore || out.NextCursor == "" {
				break
			}
			cursor = out.NextCursor
		}
	}
	return costs, nil
}

// ModelPrices returns fal's current list price per SECOND for each of the given
// model endpoints that are priced per second (GET /models/pricing; any API key
// works, no billing permission needed). An endpoint missing from the reply, or
// priced in another unit, is absent.
//
// What the number means: fal answers ONE price per endpoint, with no split by
// option. Checked on 2026-11-27: Veo 3.1 answered 0.40, which is its price with
// sound (silent is 0.20); Kling answered 0.14 (silent is 0.084). So treat it as
// the top of the range for that endpoint: a ceiling, not the price of a silent
// clip.
func (a *FalAccount) ModelPrices(ctx context.Context, endpointIDs []string) (map[string]float64, error) {
	if a.fal.apiKey == "" {
		return nil, errors.New("no api key")
	}
	prices := map[string]float64{}
	for len(endpointIDs) > 0 {
		n := min(len(endpointIDs), 50) // fal's limit per request
		batch := endpointIDs[:n]
		endpointIDs = endpointIDs[n:]

		q := url.Values{}
		q.Set("endpoint_id", strings.Join(batch, ","))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/models/pricing?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Key "+a.fal.apiKey)

		var out struct {
			Prices []struct {
				EndpointID string  `json:"endpoint_id"`
				UnitPrice  float64 `json:"unit_price"`
				Unit       string  `json:"unit"`
				Currency   string  `json:"currency"`
			} `json:"prices"`
		}
		if err := a.fal.do(req, &out); err != nil {
			return nil, fmt.Errorf("fal pricing: %w", err)
		}
		for _, p := range out.Prices {
			if p.Unit == "seconds" && p.Currency == "USD" && p.UnitPrice > 0 {
				prices[p.EndpointID] = p.UnitPrice
			}
		}
	}
	return prices, nil
}
