package products

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

// maxPictureBytes is tighter than a general upload (images.MaxBytes): several
// pictures travel to the image model together, as base64 in one request.
const maxPictureBytes = 4 << 20

const maxLabelRunes = 60

type pictureDTO struct {
	ImageID string `json:"imageId"` // GET /images/{id} serves the picture
	Label   string `json:"label"`
}

// pickableDTO is a product with pictures, as the picture dialog lists it.
type pickableDTO struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Pictures []pictureDTO `json:"pictures"`
}

// pickable lists every product that has pictures, with them, for the ticks in
// the picture dialog.
func (h *Handler) pickable(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListPickableProductPictures(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the product pictures.")
		return
	}
	out := []pickableDTO{}
	for _, row := range rows { // ordered by product, then position
		id := pgconv.UUIDString(row.ProductID)
		if n := len(out); n == 0 || out[n-1].ID != id {
			out = append(out, pickableDTO{ID: id, Name: row.Name, Pictures: []pictureDTO{}})
		}
		last := &out[len(out)-1]
		last.Pictures = append(last.Pictures, pictureDTO{ImageID: pgconv.UUIDString(row.ImageID), Label: row.Label})
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": out})
}

// listPictures returns one product's pictures in order.
func (h *Handler) listPictures(w http.ResponseWriter, r *http.Request) {
	id, ok := h.productFromPath(w, r)
	if !ok {
		return
	}
	rows, err := h.q.ListProductPictures(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list the pictures.")
		return
	}
	out := make([]pictureDTO, len(rows))
	for i, row := range rows {
		out[i] = pictureDTO{ImageID: pgconv.UUIDString(row.ImageID), Label: row.Label}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pictures": out, "max": studio.MaxProductPictures})
}

// addPicture takes a multipart form with a "file" and an optional "label" and
// keeps it as the product's next picture.
func (h *Handler) addPicture(w http.ResponseWriter, r *http.Request) {
	id, ok := h.productFromPath(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPictureBytes+1<<10+4<<10)
	file, _, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeFileError(w, pictureTooLarge())
			return
		}
		writeFileError(w, "Choose an image file to upload.")
		return
	}
	defer file.Close()
	label := strings.TrimSpace(r.FormValue("label"))
	if utf8.RuneCountInString(label) > maxLabelRunes {
		writeFieldError(w, "label", "The label must be "+strconv.Itoa(maxLabelRunes)+" characters or fewer.")
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxPictureBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Could not read the upload.")
		return
	}
	if len(data) > maxPictureBytes {
		writeFileError(w, pictureTooLarge())
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
	if _, err := h.q.AddProductPicture(r.Context(), sqlc.AddProductPictureParams{ProductID: id, ImageID: img.ID, Label: label}); err != nil {
		_ = h.q.DeleteImageIfUnused(r.Context(), img.ID) // do not leave it behind
		if errors.Is(err, pgx.ErrNoRows) {
			writeFileError(w, "A product can have up to "+strconv.Itoa(studio.MaxProductPictures)+" pictures. Remove one first.")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to add the picture.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"picture": pictureDTO{ImageID: pgconv.UUIDString(img.ID), Label: label}})
}

// updatePictureLabel changes what a picture says it shows.
func (h *Handler) updatePictureLabel(w http.ResponseWriter, r *http.Request) {
	id, imageID, ok := h.pictureFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		Label string `json:"label"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}
	label := strings.TrimSpace(body.Label)
	if utf8.RuneCountInString(label) > maxLabelRunes {
		writeFieldError(w, "label", "The label must be "+strconv.Itoa(maxLabelRunes)+" characters or fewer.")
		return
	}
	n, err := h.q.UpdateProductPictureLabel(r.Context(), sqlc.UpdateProductPictureLabelParams{ProductID: id, ImageID: imageID, Label: label})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save the label.")
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "Picture not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"picture": pictureDTO{ImageID: pgconv.UUIDString(imageID), Label: label}})
}

// removePicture drops a picture from a product (and the stored image, as
// nothing else uses it).
func (h *Handler) removePicture(w http.ResponseWriter, r *http.Request) {
	id, imageID, ok := h.pictureFromPath(w, r)
	if !ok {
		return
	}
	n, err := h.q.RemoveProductPicture(r.Context(), sqlc.RemoveProductPictureParams{ProductID: id, ImageID: imageID})
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

// productFromPath reads {id} and checks the product exists; it writes the error
// response itself when not.
func (h *Handler) productFromPath(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id.")
		return id, false
	}
	if _, err := h.q.GetProduct(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Product not found.")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to load the product.")
		}
		return id, false
	}
	return id, true
}

// pictureFromPath reads {id} and {imageId}.
func (h *Handler) pictureFromPath(w http.ResponseWriter, r *http.Request) (id, imageID pgtype.UUID, ok bool) {
	id, ok = h.productFromPath(w, r)
	if !ok {
		return
	}
	imageID, err := pgconv.ParseUUID(chi.URLParam(r, "imageId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid image id.")
		return id, imageID, false
	}
	return id, imageID, true
}

func pictureTooLarge() string {
	return "The picture is too large; a product picture can be at most " + strconv.Itoa(maxPictureBytes>>20) + " MB."
}

// writeFileError is a 422 for the "file" field.
func writeFileError(w http.ResponseWriter, msg string) { writeFieldError(w, "file", msg) }

func writeFieldError(w http.ResponseWriter, field, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": map[string]string{field: msg}})
}
