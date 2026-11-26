package generate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
