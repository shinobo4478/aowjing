package profiles

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/images"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// maxReferenceBytes is tighter than a general upload (images.MaxBytes): several
// of these travel to the image model together, as base64 in one request.
const maxReferenceBytes = 4 << 20

const (
	maxCastNameRunes = 60
	maxCastNoteRunes = 120
)

// castDTO is one cast member with its pictures in order. ImageIDs are served by
// GET /images/{id}.
type castDTO struct {
	ID       string   `json:"id"`
	Role     string   `json:"role"`
	Name     string   `json:"name"`
	Note     string   `json:"note"`
	ImageIDs []string `json:"imageIds"`
}

func castToDTO(m sqlc.CastMember, imageIDs []string) castDTO {
	if imageIDs == nil {
		imageIDs = []string{}
	}
	return castDTO{ID: pgconv.UUIDString(m.ID), Role: m.Role, Name: m.Name, Note: m.Note, ImageIDs: imageIDs}
}

type castInput struct {
	Role string `json:"role"`
	Name string `json:"name"`
	Note string `json:"note"`
}

// validate trims the input and returns the field errors, empty when fine.
func (in *castInput) validate() map[string]string {
	in.Name = strings.TrimSpace(in.Name)
	in.Note = strings.TrimSpace(in.Note)
	errs := map[string]string{}
	if !studio.ValidCastRole(in.Role) {
		errs["role"] = "Choose hero, costar, prop or place."
	}
	if in.Name == "" {
		errs["name"] = "Give it a name."
	} else if utf8.RuneCountInString(in.Name) > maxCastNameRunes {
		errs["name"] = "The name must be " + strconv.Itoa(maxCastNameRunes) + " characters or fewer."
	}
	if utf8.RuneCountInString(in.Note) > maxCastNoteRunes {
		errs["note"] = "The note must be " + strconv.Itoa(maxCastNoteRunes) + " characters or fewer."
	}
	return errs
}

func decodeCastInput(w http.ResponseWriter, r *http.Request) (castInput, bool) {
	var in castInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return in, false
	}
	if errs := in.validate(); len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return in, false
	}
	return in, true
}

// listCast returns the profile's cast in order, each with its pictures.
func (h *Handler) listCast(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	members, err := h.q.ListCastMembers(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the cast.")
		return
	}
	pics, err := h.q.ListCastPictures(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the cast pictures.")
		return
	}
	byCast := map[string][]string{}
	for _, p := range pics {
		k := pgconv.UUIDString(p.CastID)
		byCast[k] = append(byCast[k], pgconv.UUIDString(p.ImageID))
	}
	out := make([]castDTO, len(members))
	for i, m := range members {
		out[i] = castToDTO(m, byCast[pgconv.UUIDString(m.ID)])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cast":              out,
		"maxMembers":        studio.MaxCastMembers,
		"maxMemberPictures": studio.MaxMemberPictures,
	})
}

// createCast adds a member with no pictures yet.
func (h *Handler) createCast(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	in, ok := decodeCastInput(w, r)
	if !ok {
		return
	}
	n, err := h.q.CountCastMembers(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add to the cast.")
		return
	}
	if n >= studio.MaxCastMembers {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string]string{"name": "A profile can keep up to " + strconv.Itoa(studio.MaxCastMembers) + " cast members. Remove one first."},
		})
		return
	}
	m, err := h.q.CreateCastMember(r.Context(), sqlc.CreateCastMemberParams{
		ProfileID: profileID, Role: in.Role, Name: in.Name, Note: in.Note,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add to the cast.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"member": castToDTO(m, nil)})
}

// updateCast changes a member's role, name and note (pictures have their own
// routes).
func (h *Handler) updateCast(w http.ResponseWriter, r *http.Request) {
	profileID, castID, ok := h.castFromPath(w, r)
	if !ok {
		return
	}
	in, ok := decodeCastInput(w, r)
	if !ok {
		return
	}
	m, err := h.q.UpdateCastMember(r.Context(), sqlc.UpdateCastMemberParams{
		ID: castID, ProfileID: profileID, Role: in.Role, Name: in.Name, Note: in.Note,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Cast member not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update the cast member.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"member": castToDTO(m, nil)})
}

// deleteCast removes a member and every picture of it that nothing else uses.
func (h *Handler) deleteCast(w http.ResponseWriter, r *http.Request) {
	profileID, castID, ok := h.castFromPath(w, r)
	if !ok {
		return
	}
	// Collected first: the cascade drops the links, and then nothing says which
	// pictures to clean up.
	imageIDs, err := h.q.ListCastMemberImageIDs(r.Context(), castID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove the cast member.")
		return
	}
	n, err := h.q.DeleteCastMember(r.Context(), sqlc.DeleteCastMemberParams{ID: castID, ProfileID: profileID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove the cast member.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Cast member not found.")
		return
	}
	for _, id := range imageIDs {
		_ = h.q.DeleteImageIfUnused(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// addCastImage takes a multipart form with one file field named "file" and
// keeps it as the member's next picture.
func (h *Handler) addCastImage(w http.ResponseWriter, r *http.Request) {
	_, castID, ok := h.castFromPath(w, r)
	if !ok {
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
	if _, err := h.q.AddCastImage(r.Context(), sqlc.AddCastImageParams{CastID: castID, ImageID: img.ID}); err != nil {
		_ = h.q.DeleteImageIfUnused(r.Context(), img.ID) // do not leave it behind
		if errors.Is(err, pgx.ErrNoRows) {
			writeFileError(w, "A cast member can have up to "+strconv.Itoa(studio.MaxMemberPictures)+" pictures. Remove one first.")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to add the picture.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"imageId": pgconv.UUIDString(img.ID)})
}

// removeCastImage drops a picture from a member (and the stored image, as
// nothing else uses it).
func (h *Handler) removeCastImage(w http.ResponseWriter, r *http.Request) {
	_, castID, ok := h.castFromPath(w, r)
	if !ok {
		return
	}
	imageID, err := pgconv.ParseUUID(chi.URLParam(r, "imageId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid image id.")
		return
	}
	n, err := h.q.RemoveCastImage(r.Context(), sqlc.RemoveCastImageParams{CastID: castID, ImageID: imageID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove the picture.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Picture not found.")
		return
	}
	_ = h.q.DeleteImageIfUnused(r.Context(), imageID)
	w.WriteHeader(http.StatusNoContent)
}

// castFromPath reads {id} and {castId} and checks the member belongs to that
// profile; it writes the error response itself when not.
func (h *Handler) castFromPath(w http.ResponseWriter, r *http.Request) (profileID, castID pgtype.UUID, ok bool) {
	profileID, ok = h.profileFromPath(w, r)
	if !ok {
		return
	}
	castID, err := pgconv.ParseUUID(chi.URLParam(r, "castId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid cast member id.")
		return profileID, castID, false
	}
	if _, err := h.q.GetCastMember(r.Context(), sqlc.GetCastMemberParams{ID: castID, ProfileID: profileID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Cast member not found.")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to load the cast member.")
		}
		return profileID, castID, false
	}
	return profileID, castID, true
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
	return "The picture is too large; a cast picture can be at most " + strconv.Itoa(maxReferenceBytes>>20) + " MB."
}

// writeFileError is a 422 for the "file" field, the shape other handlers use
// for validation problems.
func writeFileError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"errors": map[string]string{"file": msg},
	})
}
