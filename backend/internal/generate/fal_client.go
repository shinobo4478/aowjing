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

// falClient is the part of talking to fal.ai's queue API that every fal model
// shares, whether it makes a video or a picture: submit a request, wait for it,
// read the result. Called with net/http — fal has no Go SDK.
//
// Flow: POST the model endpoint -> get a request with status_url/response_url
// -> poll status until COMPLETED -> GET the response.
type falClient struct {
	apiKey string
	http   *http.Client
}

func newFalClient(apiKey string) *falClient {
	return &falClient{apiKey: apiKey, http: &http.Client{Timeout: 30 * time.Second}}
}

const (
	falQueueBase = "https://queue.fal.run"
	falPollEvery = 5 * time.Second
)

var errNoFalKey = errors.New("fal.ai API key is not set — add it on the Settings screen")

type falSubmitResponse struct {
	RequestID   string `json:"request_id"`
	StatusURL   string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

type falStatusResponse struct {
	Status string `json:"status"` // IN_QUEUE | IN_PROGRESS | COMPLETED
}

func (c *falClient) submit(ctx context.Context, model string, input map[string]any) (falSubmitResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return falSubmitResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		falQueueBase+"/"+model, bytes.NewReader(body))
	if err != nil {
		return falSubmitResponse{}, err
	}
	req.Header.Set("Authorization", "Key "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	var out falSubmitResponse
	if err := c.do(req, &out); err != nil {
		return falSubmitResponse{}, fmt.Errorf("fal submit: %w", err)
	}
	if out.StatusURL == "" || out.ResponseURL == "" {
		return falSubmitResponse{}, errors.New("fal submit: missing status/response url in reply")
	}
	return out, nil
}

// waitForCompletion polls the status URL until the request is COMPLETED, or
// gives up after maxWait.
func (c *falClient) waitForCompletion(ctx context.Context, statusURL string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Key "+c.apiKey)

		var st falStatusResponse
		if err := c.do(req, &st); err != nil {
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
			return errors.New("fal: timed out waiting for the result")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(falPollEvery):
		}
	}
}

// result reads a completed request's response into v.
func (c *falClient) result(ctx context.Context, responseURL string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, responseURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Key "+c.apiKey)
	if err := c.do(req, v); err != nil {
		return fmt.Errorf("fal result: %w", err)
	}
	return nil
}

// do sends req, decodes a 2xx JSON body into v, and turns non-2xx into an
// error carrying a snippet of the body.
func (c *falClient) do(req *http.Request, v any) error {
	resp, err := c.http.Do(req)
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
