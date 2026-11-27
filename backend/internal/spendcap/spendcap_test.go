package spendcap

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

func f(v float64) *float64 { return &v }

func TestSum(t *testing.T) {
	clips := []sqlc.SpendClipsSinceRow{
		{CostUsd: f(2.4), Seconds: 6, Status: "succeeded", VideoModel: "veo-3.1", VideoAudio: true},   // billed: the bill, not the estimate
		{CostUsd: nil, Seconds: 5, Status: "pending", VideoModel: "kling-v3"},                         // 5 x 0.084
		{CostUsd: nil, Seconds: 5, Status: "failed", VideoModel: "kling-v3"},                          // nothing
		{CostUsd: f(0.1), Seconds: 5, Status: "failed", VideoModel: "kling-v3"},                       // a failed clip fal billed
		{CostUsd: nil, Seconds: 4, Status: "succeeded", VideoModel: "veo-3.1-fast", VideoAudio: true}, // bill not in yet: estimate with sound
	}
	frames := []sqlc.SpendKeyframesSinceRow{{Model: "seedream", Count: 2}}
	want := 2.4 + 0.42 + 0.1 + 4*0.15 + 2*0.03
	if got := Sum(clips, frames); math.Abs(got-want) > 1e-9 {
		t.Fatalf("Sum = %v, want %v", got, want)
	}
}

func TestParseCap(t *testing.T) {
	for in, want := range map[string]float64{"": 0, " ": 0, "abc": 0, "-3": 0, "0": 0, "5": 5, " 7.5 ": 7.5} {
		if got := ParseCap(in); got != want {
			t.Errorf("ParseCap(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestStartOfDayUsesOwnersZone(t *testing.T) {
	bkk, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		t.Skip("no tz database")
	}
	// 20:00 UTC on the 26th is 03:00 on the 27th in Bangkok.
	now := time.Date(2026, 11, 26, 20, 0, 0, 0, time.UTC)
	got := StartOfDay(now, bkk)
	want := time.Date(2026, 11, 27, 0, 0, 0, 0, bkk)
	if !got.Equal(want) {
		t.Fatalf("StartOfDay = %v, want %v", got, want)
	}
}

func TestClipUSDFallsBackToSilent(t *testing.T) {
	m := studio.VideoModel{USDPerSecond: 0.1} // no sound price known
	if got := ClipUSD(m, 4, true); math.Abs(got-0.4) > 1e-9 {
		t.Fatalf("ClipUSD with unknown sound price = %v, want 0.4", got)
	}
}

type fakeStore struct {
	settings map[string]string
	clips    []sqlc.SpendClipsSinceRow
}

func (s fakeStore) GetSetting(_ context.Context, key string) (string, error) {
	v, ok := s.settings[key]
	if !ok {
		return "", pgx.ErrNoRows
	}
	return v, nil
}
func (s fakeStore) SpendClipsSince(context.Context, pgtype.Timestamptz) ([]sqlc.SpendClipsSinceRow, error) {
	return s.clips, nil
}
func (s fakeStore) SpendKeyframesSince(context.Context, pgtype.Timestamptz) ([]sqlc.SpendKeyframesSinceRow, error) {
	return nil, nil
}

func TestCheck(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	spent := []sqlc.SpendClipsSinceRow{{CostUsd: f(2.0), Seconds: 5, Status: "succeeded", VideoModel: "kling-v3"}}
	cases := []struct {
		name     string
		settings map[string]string
		add      float64
		blocked  bool
	}{
		{"no limit set", map[string]string{}, 100, false},
		{"limit empty", map[string]string{KeyCap: ""}, 100, false},
		{"fits", map[string]string{KeyCap: "5"}, 1, false},
		{"exactly the limit", map[string]string{KeyCap: "5"}, 3, false},
		{"over", map[string]string{KeyCap: "5"}, 3.5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			over, err := Check(context.Background(), fakeStore{settings: c.settings, clips: spent}, c.add, now)
			if err != nil {
				t.Fatal(err)
			}
			if (over != nil) != c.blocked {
				t.Fatalf("blocked = %v, want %v (%v)", over != nil, c.blocked, over)
			}
		})
	}
}

type brokenStore struct{ fakeStore }

func (brokenStore) SpendClipsSince(context.Context, pgtype.Timestamptz) ([]sqlc.SpendClipsSinceRow, error) {
	return nil, errors.New("db down")
}

func TestCheckLetsSpendThroughWhenTheGuardBreaks(t *testing.T) {
	st := brokenStore{fakeStore{settings: map[string]string{KeyCap: "1"}}}
	over, err := Check(context.Background(), st, 50, time.Now())
	if over != nil || err == nil {
		t.Fatalf("want no limit error and a returned error, got %v / %v", over, err)
	}
}
