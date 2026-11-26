package profiles

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/images"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// maxReferenceBytes is tighter than a general upload (images.MaxBytes): up to
// four of these travel to the image model together, as base64 in one request.
const maxReferenceBytes = 4 << 20

type referenceDTO struct {
	ImageID  string `json:"imageId"` // GET /images/{id} serves the picture
	Position int16  `json:"position"`
}

// listReferences returns the profile's reference pictures in order.
func (h *Handler) listReferences(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	rows, err := h.q.ListProfileReferences(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the reference pictures.")
		return
	}
	out := make([]referenceDTO, len(rows))
	for i, row := range rows {
		out[i] = referenceDTO{ImageID: pgconv.UUIDString(row.ImageID), Position: row.Position}
	}
	writeJSON(w, http.StatusOK, map[string]any{"references": out})
}

// addReference takes a multipart form with one file field named "file" and
// keeps it as the next reference picture.
func (h *Handler) addReference(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	existing, err := h.q.ListProfileReferences(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add the reference picture.")
		return
	}
	if len(existing) >= studio.MaxReferences {
		writeFileError(w, "A profile can keep up to "+strconv.Itoa(studio.MaxReferences)+" reference pictures. Remove one first.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxReferenceBytes+1<<10)
	file, _, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeFileError(w, referenceTooLarge())
			return
		}
		writeFileError(w, "Choose an image file to upload.")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxReferenceBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Could not read the upload.")
		return
	}
	if len(data) > maxReferenceBytes {
		writeFileError(w, referenceTooLarge())
		return
	}
	contentType, err := images.Check(data)
	if err != nil {
		writeFileError(w, err.Error())
		return
	}

	img, err := h.q.CreateImage(r.Context(), sqlc.CreateImageParams{
		ContentType: contentType,
		SizeBytes:   int32(len(data)),
		Data:        data,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save the image.")
		return
	}
	position, err := h.q.AddProfileReference(r.Context(), sqlc.AddProfileReferenceParams{
		ProfileID: profileID,
		ImageID:   img.ID,
	})
	if err != nil {
		_ = h.q.DeleteImageIfUnused(r.Context(), img.ID) // do not leave it behind
		if errors.Is(err, pgx.ErrNoRows) {
			writeFileError(w, "A profile can keep up to "+strconv.Itoa(studio.MaxReferences)+" reference pictures. Remove one first.")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to add the reference picture.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"reference": referenceDTO{ImageID: pgconv.UUIDString(img.ID), Position: position},
	})
}

// removeReference drops a reference picture (and the stored image, as nothing
// else uses it).
func (h *Handler) removeReference(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	imageID, err := pgconv.ParseUUID(chi.URLParam(r, "imageId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid image id.")
		return
	}
	n, err := h.q.RemoveProfileReference(r.Context(), sqlc.RemoveProfileReferenceParams{
		ProfileID: profileID,
		ImageID:   imageID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove the reference picture.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Reference picture not found.")
		return
	}
	_ = h.q.DeleteImageIfUnused(r.Context(), imageID)
	w.WriteHeader(http.StatusNoContent)
}

// profileFromPath reads {id}, checks the profile exists, and writes the error
// response itself when it does not.
func (h *Handler) profileFromPath(w http.ResponseWriter, r *http.Request) (id pgtype.UUID, ok bool) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile id.")
		return id, false
	}
	exists, err := h.q.ProfileExists(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile.")
		return id, false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "Profile not found.")
		return id, false
	}
	return id, true
}

func referenceTooLarge() string {
	return "The picture is too large; a reference picture can be at most " + strconv.Itoa(maxReferenceBytes>>20) + " MB."
}

// writeFileError is a 422 for the "file" field, the shape other handlers use
// for validation problems.
func writeFileError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"errors": map[string]string{"file": msg},
	})
}
