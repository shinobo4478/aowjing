package generate

import (
	"context"
	"errors"
	"strconv"
	"time"
)

// FalVideoGenerator runs a prompt through fal.ai's queue API and returns the
// URL of the generated video. The queue plumbing is shared with the picture
// generator in falClient.
type FalVideoGenerator struct {
	fal *falClient
	// textModel makes a video from a prompt alone; imageModel starts from an
	// image. e.g. "fal-ai/kling-video/v3/standard/text-to-video" and
	// ".../image-to-video".
	textModel  string
	imageModel string
}

func NewFalVideoGenerator(apiKey, textModel, imageModel string) *FalVideoGenerator {
	return &FalVideoGenerator{
		fal:        newFalClient(apiKey),
		textModel:  textModel,
		imageModel: imageModel,
	}
}

func (g *FalVideoGenerator) Name() string { return "fal" }

const (
	// falVideoMaxWait is how long to wait for a clip. The first real Kling clip
	// took about 8 minutes.
	falVideoMaxWait = 12 * time.Minute

	// defaultClipSeconds is the length of an image-to-video clip when the
	// request does not say. fal takes 3 to 15, as a string.
	defaultClipSeconds = 5
)

type falVideoResult struct {
	Video struct {
		URL string `json:"url"`
	} `json:"video"`
}

func (g *FalVideoGenerator) Generate(ctx context.Context, req Request) (Result, error) {
	if g.fal.apiKey == "" {
		return Result{}, errNoFalKey
	}

	model, input := g.plan(req)
	sub, err := g.fal.submit(ctx, model, input)
	if err != nil {
		return Result{}, err
	}

	if err := g.fal.waitForCompletion(ctx, sub.StatusURL, falVideoMaxWait); err != nil {
		return Result{}, err
	}

	var res falVideoResult
	if err := g.fal.result(ctx, sub.ResponseURL, &res); err != nil {
		return Result{}, err
	}
	if res.Video.URL == "" {
		return Result{}, errors.New("fal result: no video url in reply")
	}
	return Result{Output: res.Video.URL, Kind: "video", Provider: "fal", Model: model}, nil
}

// plan picks the model and builds its JSON input. With an image it uses the
// image-to-video model, whose image field is start_image_url (not image_url).
// Those clips are silent: generate_audio defaults to true on fal and costs
// more, and Thai narration is added separately (CLAUDE.md, Phase 4).
func (g *FalVideoGenerator) plan(req Request) (model string, input map[string]any) {
	if req.ImageURL == "" {
		return g.textModel, map[string]any{"prompt": req.Prompt}
	}
	seconds := req.Seconds
	if seconds == 0 {
		seconds = defaultClipSeconds
	}
	input = map[string]any{
		"prompt":          req.Prompt,
		"start_image_url": req.ImageURL,
		"duration":        strconv.Itoa(seconds), // a string on fal, not a number
		"generate_audio":  false,
	}
	if req.EndImageURL != "" {
		input["end_image_url"] = req.EndImageURL
	}
	return g.imageModel, input
}
