// Package generate defines the AI text-generation seam for the "Generate"
// action. Phase 1 ships a single implementation (MockGenerator); a real
// provider drops in behind the same interface once one is chosen.
package generate

import "context"

// Result is what a generator returns on success.
type Result struct {
	Output   string
	Provider string
	Model    string
}

// Generator turns a prompt into generated text.
type Generator interface {
	// Name identifies the provider, recorded on every generation (including
	// failures).
	Name() string
	Generate(ctx context.Context, prompt string) (Result, error)
}
