package generate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
