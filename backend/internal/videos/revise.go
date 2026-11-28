package videos

import (
	"context"
	"net/http"
	"time"

	"github.com/shinobo4478/aowjing/backend/internal/scriptwriter"
)

// reviseTimeout covers the model's reply (fal's queue wait is up to 3 minutes).
const reviseTimeout = 4 * time.Minute

// reviseShot asks the script helper to rewrite ONE shot's three prompts (picture,
// end picture, motion) from a sentence of the owner's, with the studio's rules, the
// shot's narration and its neighbours as context. It SAVES NOTHING: the screen puts
// the suggestion into the shot's boxes and the owner saves it. The only cost is the
// model's reply, returned so the screen can show it (like a draft or a translation,
// it is not counted in the daily limit).
func (h *Handler) reviseShot(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	shot, ok := h.loadShot(w, r, v)
	if !ok {
		return
	}
	var body struct {
		Instruction string `json:"instruction"`
	}
	if !decode(w, r, &body) {
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
	req := scriptwriter.ReviseRequest{
		Instruction: body.Instruction,
		Scene:       v.Scene,
		Brief: scriptwriter.Brief{
			ProfileName: profile.Name, Niche: profile.Niche, Description: profile.Description,
			VideoStyle: profile.VideoStyle,
		},
		Shot: scriptwriter.ShotText{
			PicturePrompt: shot.PicturePrompt, EndPicturePrompt: shot.EndPicturePrompt,
			MotionPrompt: shot.Prompt, NarrationTh: shot.NarrationTh,
		},
	}
	for i, s := range shots {
		if s.ID != shot.ID {
			continue
		}
		if i > 0 {
			req.Previous = shots[i-1].PicturePrompt
		}
		if i < len(shots)-1 {
			req.Next, req.HasNext = shots[i+1].PicturePrompt, true
		}
	}
	if errs := req.Validate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return
	}
	if req.Brief.Cast, err = h.castNotes(r.Context(), profile.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the cast.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), reviseTimeout)
	defer cancel()
	rev, err := h.drafter.Revise(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The revision failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"picturePrompt": rev.PicturePrompt, "picturePromptTh": rev.PicturePromptTh,
		"endPicturePrompt": rev.EndPicturePrompt, "endPicturePromptTh": rev.EndPicturePromptTh,
		"prompt": rev.MotionPrompt, "promptTh": rev.MotionPromptTh,
		"costUsd": rev.CostUSD, "model": rev.Model,
	})
}
