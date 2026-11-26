package generate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// LLMRequest is one text completion: a system prompt (the rules), a prompt (the
// task) and the model to run.
type LLMRequest struct {
	// Model is a fal/OpenRouter model id such as "anthropic/claude-sonnet-5".
	Model       string
	System      string
	Prompt      string
	MaxTokens   int
	Temperature float64
}

// LLMResult is the reply and what it cost.
type LLMResult struct {
	Text             string
	CostUSD          float64 // fal's own figure for this call (usage.cost)
	PromptTokens     int
	CompletionTokens int
}

// LLM writes text. Like the picture and video generators it is a seam: nothing
// outside this package should know which provider answers.
type LLM interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResult, error)
}

const (
	// falLLMEndpoint is fal's OpenRouter router: it takes the same fal API key,
	// bills per token and returns the cost of each call. (fal-ai/any-llm is
	// marked deprecated on its page.)
	falLLMEndpoint = "openrouter/router"
	falLLMMaxWait  = 3 * time.Minute
)

// FalLLM runs a prompt through fal's OpenRouter router.
type FalLLM struct {
	fal *falClient
}

func NewFalLLM(apiKey string) *FalLLM { return &FalLLM{fal: newFalClient(apiKey)} }

type falLLMResult struct {
	Output  string  `json:"output"`
	Partial bool    `json:"partial"`
	Error   *string `json:"error"`
	Usage   struct {
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		Cost             float64 `json:"cost"`
	} `json:"usage"`
}

// llmInput builds the JSON body for openrouter/router. Only the fields that were
// asked for are sent, so fal's own defaults apply to the rest.
func llmInput(req LLMRequest) map[string]any {
	input := map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
		// Plain text out: reasoning would only add tokens (and cost).
		"reasoning": false,
	}
	if req.System != "" {
		input["system_prompt"] = req.System
	}
	if req.MaxTokens > 0 {
		input["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		input["temperature"] = req.Temperature
	}
	return input
}

func (l *FalLLM) Complete(ctx context.Context, req LLMRequest) (LLMResult, error) {
	if l.fal.apiKey == "" {
		return LLMResult{}, errNoFalKey
	}
	if strings.TrimSpace(req.Model) == "" {
		return LLMResult{}, errors.New("no model chosen")
	}
	sub, err := l.fal.submit(ctx, falLLMEndpoint, llmInput(req))
	if err != nil {
		return LLMResult{}, err
	}
	if err := l.fal.waitForCompletion(ctx, sub.StatusURL, falLLMMaxWait); err != nil {
		return LLMResult{}, err
	}
	var res falLLMResult
	if err := l.fal.result(ctx, sub.ResponseURL, &res); err != nil {
		return LLMResult{}, err
	}
	if res.Error != nil && *res.Error != "" {
		return LLMResult{}, fmt.Errorf("the model reported an error: %s", *res.Error)
	}
	if res.Partial {
		return LLMResult{}, errors.New("the model's reply was cut off; try fewer shots")
	}
	if strings.TrimSpace(res.Output) == "" {
		return LLMResult{}, errors.New("the model returned nothing")
	}
	return LLMResult{
		Text:             res.Output,
		CostUSD:          res.Usage.Cost,
		PromptTokens:     res.Usage.PromptTokens,
		CompletionTokens: res.Usage.CompletionTokens,
	}, nil
}
