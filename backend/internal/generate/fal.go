package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FalVideoGenerator runs a prompt through fal.ai's queue API and returns the
// URL of the generated video. Called with net/http — fal has no Go SDK.
//
// Flow: POST the model endpoint -> get a request with status_url/response_url
// -> poll status until COMPLETED -> GET the response for the video URL.
type FalVideoGenerator struct {
	apiKey string
	model  string // fal model id, e.g. "fal-ai/kling-video/v3/standard/text-to-video"
	http   *http.Client
}

func NewFalVideoGenerator(apiKey, model string) *FalVideoGenerator {
	return &FalVideoGenerator{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *FalVideoGenerator) Name() string { return "fal" }

const (
	falQueueBase = "https://queue.fal.run"
	falPollEvery = 5 * time.Second
	falMaxWait   = 12 * time.Minute
)

type falSubmitResponse struct {
	RequestID   string `json:"request_id"`
	StatusURL   string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

type falStatusResponse struct {
	Status string `json:"status"` // IN_QUEUE | IN_PROGRESS | COMPLETED
}

type falResult struct {
	Video struct {
		URL string `json:"url"`
	} `json:"video"`
}

func (g *FalVideoGenerator) Generate(ctx context.Context, prompt string) (Result, error) {
	if g.apiKey == "" {
		return Result{}, errors.New("fal.ai API key is not set — add it on the Settings screen")
	}

	sub, err := g.submit(ctx, prompt)
	if err != nil {
		return Result{}, err
	}

	if err := g.waitForCompletion(ctx, sub.StatusURL); err != nil {
		return Result{}, err
	}

	url, err := g.fetchVideoURL(ctx, sub.ResponseURL)
	if err != nil {
		return Result{}, err
	}
	return Result{Output: url, Kind: "video", Provider: "fal", Model: g.model}, nil
}

func (g *FalVideoGenerator) submit(ctx context.Context, prompt string) (falSubmitResponse, error) {
	body, _ := json.Marshal(map[string]any{"prompt": prompt})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		falQueueBase+"/"+g.model, bytes.NewReader(body))
	if err != nil {
		return falSubmitResponse{}, err
	}
	req.Header.Set("Authorization", "Key "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	var out falSubmitResponse
	if err := g.do(req, &out); err != nil {
		return falSubmitResponse{}, fmt.Errorf("fal submit: %w", err)
	}
	if out.StatusURL == "" || out.ResponseURL == "" {
		return falSubmitResponse{}, errors.New("fal submit: missing status/response url in reply")
	}
	return out, nil
}

func (g *FalVideoGenerator) waitForCompletion(ctx context.Context, statusURL string) error {
	deadline := time.Now().Add(falMaxWait)
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Key "+g.apiKey)

		var st falStatusResponse
		if err := g.do(req, &st); err != nil {
			return fmt.Errorf("fal status: %w", err)
		}
		switch st.Status {
		case "COMPLETED":
			return nil
		case "IN_QUEUE", "IN_PROGRESS":
			// keep waiting
		default:
			return fmt.Errorf("fal status: unexpected %q", st.Status)
		}

		if time.Now().After(deadline) {
			return errors.New("fal: timed out waiting for the video")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(falPollEvery):
		}
	}
}

func (g *FalVideoGenerator) fetchVideoURL(ctx context.Context, responseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, responseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+g.apiKey)

	var res falResult
	if err := g.do(req, &res); err != nil {
		return "", fmt.Errorf("fal result: %w", err)
	}
	if res.Video.URL == "" {
		return "", errors.New("fal result: no video url in reply")
	}
	return res.Video.URL, nil
}

// do sends req, decodes a 2xx JSON body into v, and turns non-2xx into an
// error carrying a snippet of the body.
func (g *FalVideoGenerator) do(req *http.Request, v any) error {
	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return fmt.Errorf("http %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
