package images

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

// Routes returns the router to mount at /images.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.upload)
	r.Get("/{id}", h.get)
	return r
}

// upload takes a multipart form with one file field named "file".
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	// Stop reading early on an oversized body instead of buffering all of it.
	// The extra KB covers the multipart framing around the file.
	r.Body = http.MaxBytesReader(w, r.Body, MaxBytes+1<<10)
	file, _, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeFileError(w, "The image is too large; the limit is "+strconv.Itoa(MaxBytes>>20)+" MB.")
			return
		}
		writeFileError(w, "Choose an image file to upload.")
		return
	}
	defer file.Close()

	// Read one byte past the limit so Check can tell "too big" from "just fits".
	data, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Could not read the upload.")
		return
	}
	contentType, err := Check(data)
	if err != nil {
		writeFileError(w, err.Error())
		return
	}

	row, err := h.q.CreateImage(r.Context(), sqlc.CreateImageParams{
		ContentType: contentType,
		SizeBytes:   int32(len(data)),
		Data:        data,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save the image.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"image": map[string]any{
		"id":          pgconv.UUIDString(row.ID),
		"contentType": row.ContentType,
		"sizeBytes":   row.SizeBytes,
	}})
}

// get serves the stored bytes. The id is unguessable and an image never
// changes, so browsers may cache it.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgconv.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid image id.")
		return
	}
	row, err := h.q.GetImage(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Image not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load the image.")
		return
	}
	w.Header().Set("Content-Type", row.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(row.Data)))
	w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(row.Data)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// writeFileError is a 422 for the "file" field, the shape other handlers use
// for validation problems.
func writeFileError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"errors": map[string]string{"file": msg},
	})
}
