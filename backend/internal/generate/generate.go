// Package generate defines the Generator seam for the "Generate" action.
//
// One interface, swappable implementations — nothing outside this package
// should reference a concrete provider. TextGenerator is the base/fallback
// (prompt text only, no external cost); provider-backed generators
// (e.g. fal.ai video) implement the same interface.
package generate

import (
	"context"
	"slices"
)

// Result is what a generator returns on success.
type Result struct {
	// Output is the generated text, or — when Kind is "video" — a URL.
	Output string
	// Kind is "text" or "video".
	Kind     string
	Provider string
	Model    string
}

// Request is what a generator is asked to make. It is a struct, not loose
// arguments, so a new input (a shot list, a voice) is a new field rather than a
// new signature on every implementation.
type Request struct {
	Prompt string
	// ImageURL, when set, is a public image the video starts from
	// (image-to-video). Generators that cannot use one ignore it.
	ImageURL string
	// Seconds is the clip length for an image-to-video clip (fal takes 3 to
	// 15). 0 means the default, 5.
	Seconds int
}

// Generator turns a request into a generation.
type Generator interface {
	// Name identifies the provider, recorded on every generation (including
	// failures).
	Name() string
	Generate(ctx context.Context, req Request) (Result, error)
}

// Providers lists the provider keys a Profile may select. "fal" is accepted as
// a forward-looking setting — its generator lands in Phase 2 item 1c; until
// then a profile set to "fal" still generates with TextGenerator.
func Providers() []string { return []string{"text", "fal"} }

// ValidProvider reports whether s is a known provider key.
func ValidProvider(s string) bool { return slices.Contains(Providers(), s) }
