package images

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func encode(t *testing.T, w, h int, format string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&buf, img)
	case "jpeg":
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantType string
		wantErr  string // substring of the error; "" = no error
	}{
		{"png ok", encode(t, 400, 400, "png"), "image/png", ""},
		{"jpeg ok", encode(t, 600, 800, "jpeg"), "image/jpeg", ""},
		{"smallest allowed", encode(t, 300, 300, "png"), "image/png", ""},
		{"widest allowed (2.5)", encode(t, 750, 300, "png"), "image/png", ""},
		{"too small", encode(t, 299, 400, "png"), "", "at least 300"},
		{"too wide", encode(t, 1000, 300, "png"), "", "ratio"},
		{"too tall", encode(t, 300, 1000, "png"), "", "ratio"},
		{"empty", nil, "", "empty"},
		{"not an image", []byte("hello, this is just text"), "", "JPG or PNG"},
		{"gif is not accepted", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), "", "JPG or PNG"},
		{"png header but truncated", []byte("\x89PNG\r\n\x1a\n\x00\x00"), "", "could not be read"},
		{"over the size cap", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, MaxBytes)...), "", "too large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Check(tt.data)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.wantType {
					t.Errorf("type = %q, want %q", got, tt.wantType)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q, got type %q", tt.wantErr, got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
