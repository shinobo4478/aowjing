package videos

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
)

// maxSceneRunes is the most a scene (one place, a sentence or two) may hold. The
// scene goes in front of a shot's picture prompt, and a picture request accepts 1500
// characters in all.
const maxSceneRunes = 600

// setScene replaces the video's scene: the place every shot happens in, kept once
// here and put in front of each shot's picture prompt when its picture is made. The
// English is what the picture model receives, so it may hold no Thai letters (the
// same rule as a shot's English prompts); the Thai is the owner's own twin of it.
// Both may be empty (no scene).
func (h *Handler) setScene(w http.ResponseWriter, r *http.Request) {
	v, ok := h.loadVideo(w, r)
	if !ok {
		return
	}
	var body struct {
		Scene   string `json:"scene"`
		SceneTh string `json:"sceneTh"`
	}
	if !decode(w, r, &body) {
		return
	}
	scene, sceneTh := strings.TrimSpace(body.Scene), strings.TrimSpace(body.SceneTh)
	if utf8.RuneCountInString(scene) > maxSceneRunes || utf8.RuneCountInString(sceneTh) > maxSceneRunes {
		writeFieldError(w, "scene", "The scene must be 600 characters or fewer.")
		return
	}
	for _, c := range scene {
		if unicode.In(c, unicode.Thai) {
			writeFieldError(w, "scene", "The English scene may not contain Thai letters: translate it first.")
			return
		}
	}
	updated, err := h.q.UpdateVideoScene(r.Context(), sqlc.UpdateVideoSceneParams{ID: v.ID, Scene: scene, SceneTh: sceneTh})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save the scene.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"video": toVideoDTO(updated, 0, 0)})
}
