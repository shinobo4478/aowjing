package generate

import (
	"context"
	"errors"
	"strings"
)

// TextGenerator is the base implementation: the generated output is the prompt
// text itself. No external API, no cost. It's what a profile uses when it has
// no provider-backed generator configured, and the fallback everything else is
// measured against.
type TextGenerator struct{}

func (TextGenerator) Name() string { return "text" }

func (TextGenerator) Generate(_ context.Context, prompt string) (Result, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return Result{}, errors.New("prompt is empty")
	}
	return Result{Output: prompt, Provider: "text"}, nil
}
