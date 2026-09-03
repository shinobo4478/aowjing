// Package generate defines the Generator seam for the "Generate" action.
//
// One interface, swappable implementations — nothing outside this package
// should reference a concrete provider. TextGenerator is the base/fallback
// (prompt text only, no external cost); provider-backed generators
// (e.g. fal.ai video) implement the same interface.
package generate

import "context"

// Result is what a generator returns on success.
type Result struct {
	Output   string
	Provider string
	Model    string
}

// Generator turns a prompt into a generation.
type Generator interface {
	// Name identifies the provider, recorded on every generation (including
	// failures).
	Name() string
	Generate(ctx context.Context, prompt string) (Result, error)
}
