// Package spendcap is the daily spending limit: a number the owner sets in
// Settings (dailyCapUsd), checked before anything that costs fal credits is
// queued. It guards against a double click, a runaway "generate all", or a
// price that rose without anyone noticing.
//
// What counts as "spent today" is the real bill when fal has shown it, and the
// model's list price for anything it has not (a clip still rendering, or one
// whose bill trails by a few minutes) -- so the limit holds even while bills
// are late, instead of counting those clips as $0.
package spendcap

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// Settings keys (see internal/settings). dailyCapTz is the owner's IANA time
// zone, saved from the browser together with the limit, so "today" starts at
// the owner's midnight and not the server's.
const (
	KeyCap = "dailyCapUsd"
	KeyTz  = "dailyCapTz"
)

// LimitError says the next spend would pass the daily limit. Handlers answer it
// with 402 and its message, which the screen shows as it is.
type LimitError struct {
	Cap, Spent, Add float64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf(
		"Daily spending limit reached: %s of %s used today, and this would add about %s. Raise the limit in Settings to continue.",
		usd(e.Spent), usd(e.Cap), usd(e.Add))
}

func usd(v float64) string { return "$" + strconv.FormatFloat(v, 'f', 2, 64) }

// Store is what the check reads. *sqlc.Queries satisfies it.
type Store interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SpendClipsSince(ctx context.Context, since pgtype.Timestamptz) ([]sqlc.SpendClipsSinceRow, error)
	SpendKeyframesSince(ctx context.Context, since pgtype.Timestamptz) ([]sqlc.SpendKeyframesSinceRow, error)
}

// ClipUSD is the price counted for one clip before its bill is known: the
// model's list price for that length, with or without sound. If the sound price
// is not known it falls back to the silent one rather than counting $0.
func ClipUSD(m studio.VideoModel, seconds int, audio bool) float64 {
	if audio {
		if p := m.ClipUSD(seconds, true); p > 0 {
			return p
		}
	}
	return m.ClipUSD(seconds, false)
}

// Check returns a *LimitError when `add` more dollars today would pass the
// owner's limit, and nil when there is no limit set or it still fits. Any
// failure to read the numbers lets the spend through: a broken guard must not
// stop the owner from working, and the error is returned for the caller to log.
func Check(ctx context.Context, s Store, add float64, now time.Time) (*LimitError, error) {
	capUSD, tz, err := load(ctx, s)
	if err != nil {
		return nil, err
	}
	if capUSD <= 0 {
		return nil, nil
	}
	spent, err := SpentToday(ctx, s, tz, now)
	if err != nil {
		return nil, err
	}
	// Half a cent of slack: prices are rounded to fractions of a cent.
	if spent+add > capUSD+0.005 {
		return &LimitError{Cap: capUSD, Spent: spent, Add: add}, nil
	}
	return nil, nil
}

// load reads the limit (0 = none) and the zone.
func load(ctx context.Context, s Store) (float64, *time.Location, error) {
	raw, err := s.GetSetting(ctx, KeyCap)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.UTC, nil
	}
	if err != nil {
		return 0, nil, err
	}
	capUSD := ParseCap(raw)
	if capUSD <= 0 {
		return 0, time.UTC, nil
	}
	loc := time.UTC
	if name, err := s.GetSetting(ctx, KeyTz); err == nil {
		if l, err := time.LoadLocation(strings.TrimSpace(name)); err == nil && strings.TrimSpace(name) != "" {
			loc = l
		}
	}
	return capUSD, loc, nil
}

// ParseCap reads the setting; "" or anything that is not a positive number
// means no limit.
func ParseCap(raw string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// StartOfDay is the owner's midnight that began the day `now` falls in.
func StartOfDay(now time.Time, loc *time.Location) time.Time {
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// SpentToday adds up today's fal spend: bills where known, estimates elsewhere.
func SpentToday(ctx context.Context, s Store, loc *time.Location, now time.Time) (float64, error) {
	since := pgtype.Timestamptz{Time: StartOfDay(now, loc), Valid: true}
	clips, err := s.SpendClipsSince(ctx, since)
	if err != nil {
		return 0, err
	}
	frames, err := s.SpendKeyframesSince(ctx, since)
	if err != nil {
		return 0, err
	}
	return Sum(clips, frames), nil
}

// Sum is the arithmetic of SpentToday, apart from the database so it can be
// tested. A failed clip with no bill counts as nothing; a failed clip fal did
// bill counts as its bill.
func Sum(clips []sqlc.SpendClipsSinceRow, frames []sqlc.SpendKeyframesSinceRow) float64 {
	total := 0.0
	for _, c := range clips {
		switch {
		case c.CostUsd != nil:
			total += *c.CostUsd
		case c.Status == "failed":
			// nothing was made, and fal has shown no charge
		default:
			total += ClipUSD(studio.ModelFor(c.VideoModel), int(c.Seconds), c.VideoAudio)
		}
	}
	for _, f := range frames {
		if m, ok := studio.KeyframeModelFor(f.Model); ok {
			total += m.CostUSD(int(f.Count))
		}
	}
	return total
}
