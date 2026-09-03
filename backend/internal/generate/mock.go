package generate

import (
	"context"
	"fmt"
	"strings"
)

// MockGenerator is a placeholder until a real AI provider is wired in
// (Phase 1 item 5). It reflects the prompt back as a numbered "script" so the
// end-to-end flow — trigger, persist, display, list — can be built and tested
// without an API key or network call.
type MockGenerator struct{}

func (MockGenerator) Name() string { return "mock" }

func (MockGenerator) Generate(_ context.Context, prompt string) (Result, error) {
	var b strings.Builder
	b.WriteString("[MOCK OUTPUT — no AI provider configured yet]\n\n")
	for i, line := range strings.Split(strings.TrimSpace(prompt), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, line)
	}
	return Result{Output: b.String(), Provider: "mock", Model: "mock-v0"}, nil
}
