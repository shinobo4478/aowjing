package generate

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// FalVideoSpec says which fal models a video generator calls and how their
// inputs are shaped. It is plain data so this package does not depend on the
// studio package that owns the model registry; the worker fills it from there.
type FalVideoSpec struct {
	// Wire is "kling" or "veo": the family whose input shape to use.
	Wire string
	// TextModel makes a clip from a prompt alone ("" = a picture is required);
	// ImageModel starts from a picture; FirstLastModel is a separate endpoint for
	// a clip that also has an end picture ("" = the model takes it as a field on
	// ImageModel, as Kling does, or not at all when EndPicture is false).
	TextModel      string
	ImageModel     string
	FirstLastModel string
	EndPicture     bool
	// DefaultSeconds is the length used when a request does not say.
	DefaultSeconds int
	// Audio asks fal for sound. Off by default: Thai narration is added
	// separately and the models' sound costs more.
	Audio bool
}

// FalVideoGenerator runs a prompt through fal.ai's queue API and returns the
// URL of the generated video. The queue plumbing is shared with the picture
// generator in falClient.
type FalVideoGenerator struct {
	fal  *falClient
	spec FalVideoSpec
}

func NewFalVideoGenerator(apiKey string, spec FalVideoSpec) *FalVideoGenerator {
	if spec.DefaultSeconds == 0 {
		spec.DefaultSeconds = defaultClipSeconds
	}
	return &FalVideoGenerator{fal: newFalClient(apiKey), spec: spec}
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

	model, input, err := g.plan(req)
	if err != nil {
		return Result{}, err
	}
	sub, err := g.fal.submit(ctx, model, input)
	if err != nil {
		return Result{}, err
	}

	if err := g.fal.waitForCompletion(ctx, sub.StatusURL, falVideoMaxWait); err != nil {
		return Result{RequestID: sub.RequestID}, err
	}

	var res falVideoResult
	if err := g.fal.result(ctx, sub.ResponseURL, &res); err != nil {
		return Result{RequestID: sub.RequestID}, err
	}
	if res.Video.URL == "" {
		return Result{RequestID: sub.RequestID}, errors.New("fal result: no video url in reply")
	}
	return Result{Output: res.Video.URL, Kind: "video", Provider: "fal", Model: model, RequestID: sub.RequestID}, nil
}

// plan picks the model and builds its JSON input, in the shape the model's
// family wants. It returns an error when the model has no endpoint for what was
// asked (a text-only clip, or an end picture, on a model that cannot).
func (g *FalVideoGenerator) plan(req Request) (model string, input map[string]any, err error) {
	if g.spec.Wire == "veo" {
		return g.planVeo(req)
	}
	return g.planKling(req)
}

// planKling: with an image it uses the image-to-video model, whose image field
// is start_image_url (not image_url) and whose length is a string of seconds
// ("5"). The end picture is a field on the same model (end_image_url).
func (g *FalVideoGenerator) planKling(req Request) (string, map[string]any, error) {
	if req.ImageURL == "" {
		return g.spec.TextModel, map[string]any{"prompt": req.Prompt}, nil
	}
	seconds := req.Seconds
	if seconds == 0 {
		seconds = g.spec.DefaultSeconds
	}
	input := map[string]any{
		"prompt":          req.Prompt,
		"start_image_url": req.ImageURL,
		"duration":        strconv.Itoa(seconds), // a string on fal, not a number
		"generate_audio":  g.spec.Audio,
	}
	if req.EndImageURL != "" {
		input["end_image_url"] = req.EndImageURL
	}
	return g.spec.ImageModel, input, nil
}

// planVeo: Veo's inputs differ from Kling's. The picture is image_url; the
// length is "4s", "6s" or "8s"; the shape of the clip is set with aspect_ratio
// (we make 9:16 clips) and resolution (720p, the cheapest). An end picture is
// not a field: it needs the first-last-frame endpoint, which takes both pictures
// as first_frame_url and last_frame_url. Read from fal's model pages, 2026-11-27.
func (g *FalVideoGenerator) planVeo(req Request) (string, map[string]any, error) {
	seconds := req.Seconds
	if seconds == 0 {
		seconds = g.spec.DefaultSeconds
	}
	common := map[string]any{
		"prompt":         req.Prompt,
		"aspect_ratio":   "9:16",
		"duration":       fmt.Sprintf("%ds", seconds),
		"resolution":     "720p",
		"generate_audio": g.spec.Audio,
	}
	if req.ImageURL == "" {
		if g.spec.TextModel == "" {
			return "", nil, errors.New("this video model needs a starting picture; give the clip one")
		}
		return g.spec.TextModel, common, nil
	}
	if req.EndImageURL != "" {
		if g.spec.FirstLastModel == "" {
			return "", nil, errors.New("this video model cannot take an end picture")
		}
		common["first_frame_url"] = req.ImageURL
		common["last_frame_url"] = req.EndImageURL
		return g.spec.FirstLastModel, common, nil
	}
	common["image_url"] = req.ImageURL
	return g.spec.ImageModel, common, nil
}
