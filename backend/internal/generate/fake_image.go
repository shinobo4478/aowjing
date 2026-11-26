package generate

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"time"
)

// FakeImageGenerator stands in for FalImageGenerator during local dev without
// a fal.ai key (AI_FAKE_FAL=1): after a short wait it returns plain gradient
// pictures, so the whole pick-a-picture flow can be tried at no cost.
type FakeImageGenerator struct{}

func (FakeImageGenerator) GenerateImages(ctx context.Context, req ImageRequest) ([][]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
	}

	out := make([][]byte, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		data, err := fakePicture(i)
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}

// fakePicture draws a 576x1024 (9:16) gradient whose hue depends on n.
func fakePicture(n int) ([]byte, error) {
	const w, h = 576, 1024
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	base := [][3]uint8{{200, 90, 90}, {90, 160, 200}, {110, 190, 120}, {210, 180, 80}}[n%4]
	for y := 0; y < h; y++ {
		shade := uint8(255 - y*120/h)
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(int(base[0]) * int(shade) / 255),
				G: uint8(int(base[1]) * int(shade) / 255),
				B: uint8(int(base[2]) * int(shade) / 255),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
