// Package dashboard serves the numbers on the home screen: what has been
// spent, what is running or broken, how far the videos are along, what has been
// published and which products are not pinned yet. It only reads; every figure
// comes from tables the other packages already keep.
package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler { return &Handler{q: q} }

// Routes returns the router to mount at /dashboard.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.get)
	r.Get("/running", h.running)
	return r
}

// chartDays is how many days the spend chart covers, ending today.
const chartDays = 30

// tzPattern is the shape of an IANA zone name ("Asia/Bangkok", "UTC",
// "America/Argentina/Buenos_Aires"). Postgres does the real check; this only
// keeps odd strings out of the query.
var tzPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+){0,2}$`)

// Response is the whole dashboard in one reply: one request on page load, and
// the numbers are consistent with each other.
type Response struct {
	Timezone  string    `json:"timezone"`
	Spend     spend     `json:"spend"`
	Active    active    `json:"active"`
	Pipeline  pipeline  `json:"pipeline"`
	Published published `json:"published"`
	Products  products  `json:"products"`
}

type spend struct {
	TodayCost     float64 `json:"todayCost"`
	TodayBilled   int64   `json:"todayBilled"`   // clips today whose bill is known
	TodayUnbilled int64   `json:"todayUnbilled"` // fal clips today with no bill yet (fal can trail a clip by minutes)
	MonthCost     float64 `json:"monthCost"`
	MonthBilled   int64   `json:"monthBilled"`   // clips this month whose bill is known
	MonthUnbilled int64   `json:"monthUnbilled"` // fal clips this month with no bill yet
	TotalCost     float64 `json:"totalCost"`
	TotalBilled   int64   `json:"totalBilled"`
	TotalUnbilled int64   `json:"totalUnbilled"` // includes clips made before bills were tracked
	// AvgPerClip is MonthCost / MonthBilled; null when nothing is billed yet.
	AvgPerClip *float64       `json:"avgPerClip"`
	Days       []spendDay     `json:"days"` // chartDays entries, oldest first, empty days included
	ByProfile  []profileSpend `json:"byProfile"`
}

type spendDay struct {
	Date  string  `json:"date"` // YYYY-MM-DD in the viewer's zone
	Cost  float64 `json:"cost"`
	Clips int64   `json:"clips"`
}

type profileSpend struct {
	ProfileID string  `json:"profileId"`
	Name      string  `json:"name"`
	Cost      float64 `json:"cost"`
	Clips     int64   `json:"clips"`
}

type active struct {
	Pending []pendingClip `json:"pending"`
	Failed  []failedClip  `json:"failed"`
}

type pendingClip struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Seconds     int    `json:"seconds"`
	FromShot    bool   `json:"fromShot"`
	CreatedAt   string `json:"createdAt"`
}

type failedClip struct {
	ID          string   `json:"id"`
	ProfileID   string   `json:"profileId"`
	ProfileName string   `json:"profileName"`
	Error       string   `json:"error"`
	CostUSD     *float64 `json:"costUsd"` // fal can bill a clip that then failed
	CreatedAt   string   `json:"createdAt"`
}

type pipeline struct {
	Videos           int64 `json:"videos"`
	Shots            int64 `json:"shots"`
	ShotsApproved    int64 `json:"shotsApproved"`
	ApprovedNotMade  int64 `json:"approvedNotMade"`
	ShotsMade        int64 `json:"shotsMade"`
	UnpublishedClips int64 `json:"unpublishedClips"`
}

type published struct {
	Last7Days  int64    `json:"last7Days"`
	Last30Days int64    `json:"last30Days"`
	Total      int64    `json:"total"`
	Recent     []upload `json:"recent"`
}

type upload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Privacy     string `json:"privacy"`
	URL         string `json:"url"`
	ChannelName string `json:"channelName"`
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	CreatedAt   string `json:"createdAt"`
}

type products struct {
	Total    int64             `json:"total"`
	Pinned   int64             `json:"pinned"`
	Unpinned []unpinnedProduct `json:"unpinned"`
}

type unpinnedProduct struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tz := r.URL.Query().Get("tz")
	if !tzPattern.MatchString(tz) {
		tz = "UTC"
	}
	resp, err := h.build(r.Context(), tz)
	if err != nil && tz != "UTC" {
		// Postgres did not know the zone name: show UTC rather than nothing.
		tz = "UTC"
		resp, err = h.build(r.Context(), tz)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load the dashboard."})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) build(ctx context.Context, tz string) (Response, error) {
	resp := Response{Timezone: tz}

	today, err := h.q.DashToday(ctx, tz)
	if err != nil {
		return resp, err
	}

	s, err := h.q.DashSpend(ctx, tz)
	if err != nil {
		return resp, err
	}
	resp.Spend = spend{
		TodayCost: s.TodayCost, TodayBilled: s.TodayBilled, TodayUnbilled: s.TodayUnbilled,
		MonthCost: s.MonthCost, MonthBilled: s.MonthBilled, MonthUnbilled: s.MonthUnbilled,
		TotalCost: s.TotalCost, TotalBilled: s.TotalBilled, TotalUnbilled: s.TotalUnbilled,
	}
	if s.MonthBilled > 0 {
		avg := s.MonthCost / float64(s.MonthBilled)
		resp.Spend.AvgPerClip = &avg
	}

	days, err := h.q.DashSpendByDay(ctx, tz)
	if err != nil {
		return resp, err
	}
	resp.Spend.Days = fillDays(today.Time, days)

	byProfile, err := h.q.DashSpendByProfile(ctx, tz)
	if err != nil {
		return resp, err
	}
	resp.Spend.ByProfile = make([]profileSpend, len(byProfile))
	for i, p := range byProfile {
		resp.Spend.ByProfile[i] = profileSpend{
			ProfileID: pgconv.UUIDString(p.ID), Name: p.Name, Cost: p.Cost, Clips: p.Clips,
		}
	}

	pending, err := h.q.DashPending(ctx)
	if err != nil {
		return resp, err
	}
	resp.Active.Pending = make([]pendingClip, len(pending))
	for i, g := range pending {
		resp.Active.Pending[i] = pendingClip{
			ID: pgconv.UUIDString(g.ID), ProfileID: pgconv.UUIDString(g.ProfileID),
			ProfileName: g.ProfileName, Seconds: int(g.Seconds), FromShot: g.ShotID.Valid,
			CreatedAt: pgconv.TimeString(g.CreatedAt),
		}
	}
	failed, err := h.q.DashRecentFailed(ctx)
	if err != nil {
		return resp, err
	}
	resp.Active.Failed = make([]failedClip, len(failed))
	for i, g := range failed {
		resp.Active.Failed[i] = failedClip{
			ID: pgconv.UUIDString(g.ID), ProfileID: pgconv.UUIDString(g.ProfileID),
			ProfileName: g.ProfileName, Error: g.Error, CostUSD: g.CostUsd,
			CreatedAt: pgconv.TimeString(g.CreatedAt),
		}
	}

	pl, err := h.q.DashPipeline(ctx)
	if err != nil {
		return resp, err
	}
	resp.Pipeline = pipeline{
		Videos: pl.Videos, Shots: pl.Shots, ShotsApproved: pl.ShotsApproved,
		ApprovedNotMade: pl.ApprovedNotMade, ShotsMade: pl.ShotsMade, UnpublishedClips: pl.UnpublishedClips,
	}

	pc, err := h.q.DashPublishedCounts(ctx)
	if err != nil {
		return resp, err
	}
	resp.Published = published{Last7Days: pc.Last7Days, Last30Days: pc.Last30Days, Total: pc.Total}
	recent, err := h.q.DashRecentUploads(ctx)
	if err != nil {
		return resp, err
	}
	resp.Published.Recent = make([]upload, len(recent))
	for i, u := range recent {
		resp.Published.Recent[i] = upload{
			ID: pgconv.UUIDString(u.ID), Title: u.Title, Privacy: u.Privacy,
			URL:         "https://www.youtube.com/watch?v=" + u.YoutubeVideoID,
			ChannelName: u.ChannelName, ProfileID: pgconv.UUIDString(u.ProfileID),
			ProfileName: u.ProfileName, CreatedAt: pgconv.TimeString(u.CreatedAt),
		}
	}

	pr, err := h.q.DashProducts(ctx)
	if err != nil {
		return resp, err
	}
	resp.Products = products{Total: pr.Total, Pinned: pr.Pinned}
	unpinned, err := h.q.DashUnpinnedProducts(ctx)
	if err != nil {
		return resp, err
	}
	resp.Products.Unpinned = make([]unpinnedProduct, len(unpinned))
	for i, p := range unpinned {
		resp.Products.Unpinned[i] = unpinnedProduct{
			ID: pgconv.UUIDString(p.ID), Name: p.Name, CreatedAt: pgconv.TimeString(p.CreatedAt),
		}
	}
	return resp, nil
}

// fillDays returns chartDays entries ending on `today`, one per calendar day,
// with zeros for days nothing was billed, so the chart has no gaps to guess at.
func fillDays(today time.Time, rows []sqlc.DashSpendByDayRow) []spendDay {
	byDate := make(map[string]sqlc.DashSpendByDayRow, len(rows))
	for _, r := range rows {
		byDate[r.Day.Time.Format("2006-01-02")] = r
	}
	out := make([]spendDay, chartDays)
	for i := range out {
		d := today.AddDate(0, 0, i-(chartDays-1)).Format("2006-01-02")
		out[i] = spendDay{Date: d}
		if r, ok := byDate[d]; ok {
			out[i].Cost, out[i].Clips = r.Cost, r.Clips
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
