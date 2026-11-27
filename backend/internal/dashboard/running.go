package dashboard

import (
	"net/http"
	"strconv"

	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// runningItem is one thing being made: a clip or a set of pictures.
type runningItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // "clip" or "pictures"
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	// What it is, in words: "Video "The bag drop" - shot 2" or "3 pictures".
	Label     string `json:"label"`
	Seconds   int    `json:"seconds,omitempty"` // clips only
	Model     string `json:"model,omitempty"`   // clips only: the video model's label
	CreatedAt string `json:"createdAt"`
}

// running lists what is being made right now (GET /dashboard/running), for the
// menu bar's "rendering" indicator. It is its own small request so the bar can
// ask every few seconds without building the whole dashboard.
func (h *Handler) running(w http.ResponseWriter, r *http.Request) {
	clips, err := h.q.RunningClips(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load what is rendering."})
		return
	}
	frames, err := h.q.RunningKeyframes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load what is rendering."})
		return
	}
	items := make([]runningItem, 0, len(clips)+len(frames))
	for _, c := range clips {
		label := "Clip"
		if c.ShotPosition > 0 {
			label = "Video “" + c.VideoTitle + "”, shot " + strconv.Itoa(int(c.ShotPosition))
		}
		items = append(items, runningItem{
			ID: pgconv.UUIDString(c.ID), Kind: "clip",
			ProfileID: pgconv.UUIDString(c.ProfileID), ProfileName: c.ProfileName,
			Label: label, Seconds: int(c.Seconds), Model: studio.ModelFor(c.VideoModel).Label,
			CreatedAt: pgconv.TimeString(c.CreatedAt),
		})
	}
	for _, k := range frames {
		label := strconv.Itoa(int(k.Count)) + " pictures"
		if k.Count == 1 {
			label = "1 picture"
		}
		items = append(items, runningItem{
			ID: pgconv.UUIDString(k.ID), Kind: "pictures",
			ProfileID: pgconv.UUIDString(k.ProfileID), ProfileName: k.ProfileName,
			Label: label, CreatedAt: pgconv.TimeString(k.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
