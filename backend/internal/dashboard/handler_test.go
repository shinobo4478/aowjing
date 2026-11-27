package dashboard

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
)

func day(s string) pgtype.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return pgtype.Date{Time: t, Valid: true}
}

func TestFillDays(t *testing.T) {
	today := day("2026-12-01").Time

	t.Run("always chartDays entries ending today, oldest first", func(t *testing.T) {
		got := fillDays(today, nil)
		if len(got) != chartDays {
			t.Fatalf("len = %d, want %d", len(got), chartDays)
		}
		if got[0].Date != "2026-11-02" || got[chartDays-1].Date != "2026-12-01" {
			t.Errorf("range = %s .. %s", got[0].Date, got[chartDays-1].Date)
		}
	})

	t.Run("billed days land on their date, other days are zero", func(t *testing.T) {
		got := fillDays(today, []sqlc.DashSpendByDayRow{
			{Day: day("2026-12-01"), Cost: 0.84, Clips: 2},
			{Day: day("2026-11-30"), Cost: 0.42, Clips: 1},
		})
		if g := got[chartDays-1]; g.Cost != 0.84 || g.Clips != 2 {
			t.Errorf("today = %+v", g)
		}
		if g := got[chartDays-2]; g.Cost != 0.42 || g.Clips != 1 {
			t.Errorf("yesterday = %+v", g)
		}
		if g := got[chartDays-3]; g.Cost != 0 || g.Clips != 0 {
			t.Errorf("an empty day should be zero, got %+v", g)
		}
	})

	t.Run("a day older than the window is dropped", func(t *testing.T) {
		got := fillDays(today, []sqlc.DashSpendByDayRow{{Day: day("2026-10-01"), Cost: 9, Clips: 9}})
		for _, d := range got {
			if d.Cost != 0 {
				t.Errorf("unexpected spend on %s", d.Date)
			}
		}
	})
}

func TestTzPattern(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"Asia/Bangkok", true},
		{"UTC", true},
		{"America/Argentina/Buenos_Aires", true},
		{"Etc/GMT+7", true},
		{"", false},
		{"'; DROP TABLE generations;--", false},
		{"Asia/Bangkok/../../x/y", false},
	} {
		if got := tzPattern.MatchString(tc.in); got != tc.want {
			t.Errorf("tzPattern(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
