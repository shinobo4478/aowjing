package generate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFalAccountCredits(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotQuery = r.Header.Get("Authorization"), r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"username":"me","credits":{"current_balance":24.5,"currency":"USD"}}`))
	}))
	defer srv.Close()

	a := NewFalAccount("secret")
	a.baseURL = srv.URL
	got, err := a.Credits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 24.5 || got.Currency != "USD" {
		t.Errorf("got %+v", got)
	}
	if gotAuth != "Key secret" || gotPath != "/account/billing" || gotQuery != "expand=credits" {
		t.Errorf("request = %q %q %q", gotAuth, gotPath, gotQuery)
	}
}

func TestFalAccountCreditsErrors(t *testing.T) {
	t.Run("forbidden carries the status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"type":"authorization_error"}}`))
		}))
		defer srv.Close()
		a := NewFalAccount("k")
		a.baseURL = srv.URL
		if _, err := a.Credits(context.Background()); err == nil || !strings.Contains(err.Error(), "http 403") {
			t.Errorf("err = %v, want it to mention http 403", err)
		}
	})
	t.Run("reply without credits", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"username":"me"}`))
		}))
		defer srv.Close()
		a := NewFalAccount("k")
		a.baseURL = srv.URL
		if _, err := a.Credits(context.Background()); err == nil {
			t.Error("want an error when the reply has no credits")
		}
	})
	t.Run("no key", func(t *testing.T) {
		if _, err := NewFalAccount("").Credits(context.Background()); err == nil {
			t.Error("want an error without a key")
		}
	})
}

func TestFalAccountBillingCosts(t *testing.T) {
	var gotPath, gotIDs, gotStart string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotPath, gotIDs, gotStart = r.URL.Path, r.URL.Query().Get("request_id"), r.URL.Query().Get("start")
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			// First page: req-a (two events that add up) and a cursor onward.
			_, _ = w.Write([]byte(`{"has_more":true,"next_cursor":"c2","billing_events":[
				{"request_id":"req-a","cost_total":0.3},{"request_id":"req-a","cost_total":0.12}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"has_more":false,"billing_events":[{"request_id":"req-b","cost_total":0.336}]}`))
	}))
	defer srv.Close()

	a := NewFalAccount("secret")
	a.baseURL = srv.URL
	since := time.Date(2026, 11, 26, 8, 0, 0, 0, time.UTC)
	got, err := a.BillingCosts(context.Background(), []string{"req-a", "req-b", "req-missing"}, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["req-a"] < 0.4199 || got["req-a"] > 0.4201 || got["req-b"] != 0.336 {
		t.Errorf("got %v", got)
	}
	if _, ok := got["req-missing"]; ok {
		t.Error("an id with no billing event must be absent, not zero")
	}
	if gotPath != "/models/billing-events" || gotIDs != "req-a,req-b,req-missing" || gotStart != "2026-11-26T08:00:00Z" {
		t.Errorf("request = %q %q %q", gotPath, gotIDs, gotStart)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (a second page)", calls)
	}
}

func TestFalAccountBillingCostsBatches(t *testing.T) {
	var sizes []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sizes = append(sizes, len(strings.Split(r.URL.Query().Get("request_id"), ",")))
		_, _ = w.Write([]byte(`{"has_more":false,"billing_events":[]}`))
	}))
	defer srv.Close()
	a := NewFalAccount("k")
	a.baseURL = srv.URL
	ids := make([]string, 120)
	for i := range ids {
		ids[i] = fmt.Sprintf("r%d", i)
	}
	if _, err := a.BillingCosts(context.Background(), ids, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 3 || sizes[0] != 50 || sizes[1] != 50 || sizes[2] != 20 {
		t.Errorf("batch sizes = %v, want [50 50 20] (fal accepts at most 50 ids)", sizes)
	}
}

func TestFalAccountModelPrices(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotQuery = r.Header.Get("Authorization"), r.URL.Path, r.URL.Query().Get("endpoint_id")
		_, _ = w.Write([]byte(`{"prices":[
			{"endpoint_id":"fal-ai/veo3.1","unit_price":0.4,"unit":"seconds","currency":"USD"},
			{"endpoint_id":"fal-ai/flux/dev","unit_price":0.025,"unit":"images","currency":"USD"},
			{"endpoint_id":"fal-ai/odd","unit_price":0.1,"unit":"seconds","currency":"EUR"}
		],"next_cursor":null,"has_more":false}`))
	}))
	defer srv.Close()

	a := NewFalAccount("secret")
	a.baseURL = srv.URL
	got, err := a.ModelPrices(context.Background(), []string{"fal-ai/veo3.1", "fal-ai/flux/dev", "fal-ai/odd"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["fal-ai/veo3.1"] != 0.4 {
		t.Errorf("got %v, want only the per-second USD price of veo3.1", got)
	}
	if gotAuth != "Key secret" || gotPath != "/models/pricing" || gotQuery != "fal-ai/veo3.1,fal-ai/flux/dev,fal-ai/odd" {
		t.Errorf("request = %q %q %q", gotAuth, gotPath, gotQuery)
	}
	if _, err := NewFalAccount("").ModelPrices(context.Background(), []string{"x"}); err == nil {
		t.Error("want an error without a key")
	}
}
