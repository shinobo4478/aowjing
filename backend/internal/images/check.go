// Package images stores pictures the owner uploads from their computer, to
// start an image-to-video clip from. Pictures only (never video), kept in
// Postgres for now. POST /images uploads one, GET /images/{id} serves it back.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"net/http"

	// Blank imports register the JPEG and PNG decoders with image.DecodeConfig
	// (the standard library's plug-in style: importing is what enables them).
	_ "image/jpeg"
	_ "image/png"
)

const (
	// MaxBytes caps one upload. fal allows 50 MB, but the picture also travels
	// to fal base64-encoded (about a third bigger), so keep it modest.
	MaxBytes = 8 << 20

	// fal's limits for a starting image (from its OpenAPI schema).
	minSide   = 300
	minAspect = 0.40 // width / height
	maxAspect = 2.50
)

// Check decides whether data is a picture fal will accept as a starting image
// and returns its content type. Only JPEG and PNG: the standard library can
// read their size, and fal does not list any other format as supported.
// The type is sniffed from the bytes, never taken from the client.
func Check(data []byte) (contentType string, err error) {
	if len(data) == 0 {
		return "", errors.New("The file is empty.")
	}
	if len(data) > MaxBytes {
		return "", fmt.Errorf("The image is too large; the limit is %d MB.", MaxBytes>>20)
	}

	contentType = http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", errors.New("Use a JPG or PNG image.")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("This image file could not be read.")
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return "", fmt.Errorf("The image is %d×%d px; it must be at least %d×%d px.",
			cfg.Width, cfg.Height, minSide, minSide)
	}
	if aspect := float64(cfg.Width) / float64(cfg.Height); aspect < minAspect || aspect > maxAspect {
		return "", errors.New("The image is too narrow or too wide; its width/height ratio must be between 0.4 and 2.5.")
	}
	return contentType, nil
}
