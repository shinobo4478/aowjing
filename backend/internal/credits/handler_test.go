package credits

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shinobo4478/aowjing/backend/internal/generate"
)

// newTestHandler builds a Handler whose fal call is a counter and whose clock
// the test moves by hand. It has no database: balance() is what is under test.
func newTestHandler(fetch func(key string) (generate.FalCredits, error)) (*Handler, *time.Time, *int) {
	clock := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	calls := 0
	h := &Handler{
		fetch: func(_ context.Context, key string) (generate.FalCredits, error) {
			calls++
			return fetch(key)
		},
		now: func() time.Time { return clock },
	}
	return h, &clock, &calls
}

func TestBalanceCaches(t *testing.T) {
	h, clock, calls := newTestHandler(func(string) (generate.FalCredits, error) {
		return generate.FalCredits{Balance: 24.5, Currency: "USD"}, nil
	})
	ctx := context.Background()

	got := h.balance(ctx, "k1", false)
	if !got.Configured || got.Balance == nil || *got.Balance != 24.5 || got.Currency != "USD" {
		t.Fatalf("first call = %+v", got)
	}

	*clock = clock.Add(10 * time.Second)
	h.balance(ctx, "k1", false)
	if *calls != 1 {
		t.Errorf("a call 10 s later should be served from the cache, fal was asked %d times", *calls)
	}

	h.balance(ctx, "k1", true) // refresh, but only 10 s old: allowed
	if *calls != 2 {
		t.Errorf("refresh after 10 s should ask fal again, calls = %d", *calls)
	}

	*clock = clock.Add(2 * time.Second)
	h.balance(ctx, "k1", true) // refresh again 2 s later: too soon
	if *calls != 2 {
		t.Errorf("refresh 2 s after the last ask should not hit fal, calls = %d", *calls)
	}

	*clock = clock.Add(time.Minute)
	h.balance(ctx, "k1", false)
	if *calls != 3 {
		t.Errorf("an expired cache should ask fal, calls = %d", *calls)
	}
}

func TestBalanceNewKeyBypassesCache(t *testing.T) {
	h, _, calls := newTestHandler(func(string) (generate.FalCredits, error) {
		return generate.FalCredits{Balance: 1, Currency: "USD"}, nil
	})
	ctx := context.Background()
	h.balance(ctx, "old", false)
	h.balance(ctx, "new", false)
	if *calls != 2 {
		t.Errorf("a changed key must not reuse the old answer, calls = %d", *calls)
	}
}

func TestBalanceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"forbidden", errors.New("fal billing: http 403: nope"), "This key cannot read billing. Use a key with the BILLING preset."},
		{"unauthorized", errors.New("fal billing: http 401: x"), "fal rejected the billing key."},
		{"limited", errors.New("fal billing: http 429: x"), "fal is limiting requests; try again shortly."},
		{"other", errors.New("dial tcp: timeout"), "Could not read the balance from fal."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _ := newTestHandler(func(string) (generate.FalCredits, error) {
				return generate.FalCredits{}, tt.err
			})
			got := h.balance(context.Background(), "k", false)
			if !got.Configured || got.Balance != nil || got.Error != tt.want {
				t.Errorf("got %+v, want error %q", got, tt.want)
			}
		})
	}
}
