package generate

import (
	"context"
	"strings"
	"testing"
)

func TestMockGenerator(t *testing.T) {
	g := MockGenerator{}

	if g.Name() != "mock" {
		t.Fatalf("Name() = %q, want %q", g.Name(), "mock")
	}

	tests := []struct {
		name          string
		prompt        string
		wantContains  []string
		wantLineCount int // numbered lines "N. "
	}{
		{
			name:          "single line",
			prompt:        "Write a script.",
			wantContains:  []string{"MOCK OUTPUT", "1. Write a script."},
			wantLineCount: 1,
		},
		{
			name:          "multi line with blank",
			prompt:        "Intro\n\nBody line",
			wantContains:  []string{"1. Intro", "3. Body line"},
			wantLineCount: 2,
		},
		{
			name:          "surrounding whitespace trimmed",
			prompt:        "   hello   ",
			wantContains:  []string{"1. hello"},
			wantLineCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := g.Generate(context.Background(), tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Provider != "mock" || res.Model != "mock-v0" {
				t.Errorf("provider/model = %q/%q", res.Provider, res.Model)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(res.Output, want) {
					t.Errorf("output missing %q\n---\n%s", want, res.Output)
				}
			}
			got := 0
			for i := 1; i <= 9; i++ {
				if strings.Contains(res.Output, string(rune('0'+i))+". ") {
					got++
				}
			}
			if got != tt.wantLineCount {
				t.Errorf("numbered line count = %d, want %d", got, tt.wantLineCount)
			}
		})
	}
}
