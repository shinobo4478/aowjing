package videos

import (
	"context"
	"net/http"
	"time"

	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/scriptwriter"
)

// reviewTimeout covers the model's reply (fal's queue wait is up to 3 minutes).
const reviewTimeout = 4 * time.Minute

// reviewVideo asks the script helper to review the whole video as a strict reviewer
// (a picture that already shows the action, an icon that fights a shot's picture, a
// narration that starts with a loanword, ...). It SAVES AND CHANGES NOTHING: the
// answer is a list of findings; the screen offers "ให้ AI แก้ช็อตนี้" for the ones a
// rewrite of the prompts can fix. The only cost is the model's reply, returned so the
// screen can show it (not counted in the daily limit, like a draft).
func (h *Handler) reviewVideo(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	profile, err := h.q.GetProfile(r.Context(), v.ProfileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return
	}
	shots, err := h.q.ListShots(r.Context(), v.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the shots.")
		return
	}
	req := scriptwriter.ReviewRequest{
		Brief: scriptwriter.Brief{
			ProfileName: profile.Name, Niche: profile.Niche, Description: profile.Description,
			VideoStyle: profile.VideoStyle,
		},
	}
	ids := make(map[int]string, len(shots))
	for i, s := range shots {
		req.Shots = append(req.Shots, scriptwriter.ReviewShot{
			Position: i + 1, PicturePrompt: s.PicturePrompt, EndPicturePrompt: s.EndPicturePrompt,
			MotionPrompt: s.Prompt, NarrationTh: s.NarrationTh, Seconds: int(s.Seconds),
		})
		ids[i+1] = pgconv.UUIDString(s.ID)
	}
	if errs := req.Validate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	if req.Brief.Cast, err = h.castNotes(r.Context(), profile.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the cast.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), reviewTimeout)
	defer cancel()
	rev, err := h.drafter.Review(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The review failed: "+err.Error())
		return
	}
	type findingDTO struct {
		ShotID    string `json:"shotId"`
		Shot      int    `json:"shot"`
		Kind      string `json:"kind"`
		MessageTh string `json:"messageTh"`
		// FixTh is the instruction to hand to "revise"; "" when a rewrite of the
		// prompts cannot fix the problem (the narration kinds).
		FixTh string `json:"fixTh"`
	}
	out := make([]findingDTO, 0, len(rev.Findings))
	for _, f := range rev.Findings {
		out = append(out, findingDTO{ShotID: ids[f.Shot], Shot: f.Shot, Kind: f.Kind, MessageTh: f.MessageTh, FixTh: f.FixTh})
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": out, "costUsd": rev.CostUSD, "model": rev.Model})
}
