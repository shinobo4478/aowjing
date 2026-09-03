package generate

import (
	"context"
	"testing"
)

func TestTextGenerator(t *testing.T) {
	g := TextGenerator{}

	if g.Name() != "text" {
		t.Fatalf("Name() = %q, want %q", g.Name(), "text")
	}

	tests := []struct {
		name       string
		prompt     string
		wantOutput string
		wantErr    bool
	}{
		{"passthrough", "Write a 60s script.", "Write a 60s script.", false},
		{"trims surrounding whitespace", "  hello \n", "hello", false},
		{"keeps interior newlines", "line one\nline two", "line one\nline two", false},
		{"empty is an error", "   ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := g.Generate(context.Background(), tt.prompt)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got output %q", res.Output)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", res.Output, tt.wantOutput)
			}
			if res.Provider != "text" {
				t.Errorf("Provider = %q, want %q", res.Provider, "text")
			}
			if res.Model != "" {
				t.Errorf("Model = %q, want empty", res.Model)
			}
		})
	}
}
