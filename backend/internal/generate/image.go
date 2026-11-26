package generate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ImageRequest asks for keyframe pictures: the starting image of a clip.
type ImageRequest struct {
	// Model is the image model's key from internal/studio ("nano-banana",
	// "seedream"). It decides which request fields fal needs; ModelID is the fal
	// model to call (the studio registry picks the edit model when References
	// is not empty).
	Model   string
	ModelID string

	Prompt string
	// References are pictures to keep the look of (data URIs or public URLs).
	References []string
	// Count is how many candidates to make.
	Count int
	// Aspect is the picture's shape, e.g. "9:16".
	Aspect string
}

// ImageGenerator makes pictures. Unlike Generator it returns the picture bytes,
// not a URL: a fal link expires (7 days at least, per fal's FAQ), and a picture
// the owner has picked must not vanish before the clip is made.
type ImageGenerator interface {
	GenerateImages(ctx context.Context, req ImageRequest) ([][]byte, error)
}

// maxPictureBytes stops a runaway download; the images package enforces its own
// (smaller) limit when the picture is stored.
const maxPictureBytes = 16 << 20

const falImageMaxWait = 5 * time.Minute

// FalImageGenerator makes pictures with an image model on fal.ai.
type FalImageGenerator struct {
	fal *falClient
}

func NewFalImageGenerator(apiKey string) *FalImageGenerator {
	return &FalImageGenerator{fal: newFalClient(apiKey)}
}

type falImageResult struct {
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
}

func (g *FalImageGenerator) GenerateImages(ctx context.Context, req ImageRequest) ([][]byte, error) {
	if g.fal.apiKey == "" {
		return nil, errNoFalKey
	}
	input, err := imageInput(req)
	if err != nil {
		return nil, err
	}

	sub, err := g.fal.submit(ctx, req.ModelID, input)
	if err != nil {
		return nil, err
	}
	if err := g.fal.waitForCompletion(ctx, sub.StatusURL, falImageMaxWait); err != nil {
		return nil, err
	}
	var res falImageResult
	if err := g.fal.result(ctx, sub.ResponseURL, &res); err != nil {
		return nil, err
	}
	if len(res.Images) == 0 {
		return nil, errors.New("fal result: no pictures in reply")
	}

	out := make([][]byte, 0, len(res.Images))
	for _, im := range res.Images {
		data, err := g.download(ctx, im.URL)
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}

// download fetches one picture from fal's CDN (a public URL, no key needed).
func (g *FalImageGenerator) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.fal.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download picture: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download picture: http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPictureBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download picture: %w", err)
	}
	if len(data) > maxPictureBytes {
		return nil, errors.New("download picture: the picture is too large")
	}
	return data, nil
}

// seedreamSizes gives Seedream's pixel size for each aspect: it takes a width
// and height (not a ratio), with 960x960 to 4096x4096 total pixels.
var seedreamSizes = map[string][2]int{
	"9:16": {1152, 2048},
	"16:9": {2048, 1152},
	"1:1":  {2048, 2048},
}

// imageInput builds the JSON body for a model. The two models name their
// fields differently (checked against fal's OpenAPI schemas, 2026-11-26):
//
//   - nano-banana: aspect_ratio ("9:16"), output_format, image_urls
//   - seedream:    image_size {width,height} (no output_format), image_urls
//
// Both take num_images and, for the edit model, image_urls.
func imageInput(req ImageRequest) (map[string]any, error) {
	if req.Prompt == "" {
		return nil, errors.New("picture prompt is empty")
	}
	if req.Count < 1 {
		return nil, errors.New("picture count must be at least 1")
	}
	input := map[string]any{
		"prompt":     req.Prompt,
		"num_images": req.Count,
	}

	switch req.Model {
	case "nano-banana":
		input["aspect_ratio"] = req.Aspect
		// jpeg keeps the files small (the default is png).
		input["output_format"] = "jpeg"
	case "seedream":
		size, ok := seedreamSizes[req.Aspect]
		if !ok {
			return nil, fmt.Errorf("seedream has no size for aspect %q", req.Aspect)
		}
		input["image_size"] = map[string]any{"width": size[0], "height": size[1]}
	default:
		return nil, fmt.Errorf("unknown picture model %q", req.Model)
	}

	if len(req.References) > 0 {
		input["image_urls"] = req.References
	}
	return input, nil
}
