package generate

import (
	"context"
	"encoding/json"
	"testing"
)

func TestLLMInput(t *testing.T) {
	tests := []struct {
		name string
		req  LLMRequest
		want map[string]any
	}{
		{
			name: "everything given",
			req:  LLMRequest{Model: "anthropic/x", System: "rules", Prompt: "task", MaxTokens: 900, Temperature: 0.7},
			want: map[string]any{
				"model": "anthropic/x", "prompt": "task", "system_prompt": "rules",
				"max_tokens": 900, "temperature": 0.7, "reasoning": false,
			},
		},
		{
			name: "only what was asked for is sent",
			req:  LLMRequest{Model: "m", Prompt: "p"},
			want: map[string]any{"model": "m", "prompt": "p", "reasoning": false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := json.Marshal(llmInput(tt.req))
			want, _ := json.Marshal(tt.want)
			if string(got) != string(want) {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}

func TestFalLLMNeedsKeyAndModel(t *testing.T) {
	if _, err := NewFalLLM("").Complete(context.Background(), LLMRequest{Model: "m", Prompt: "p"}); err == nil {
		t.Error("want an error without a key")
	}
	if _, err := NewFalLLM("k").Complete(context.Background(), LLMRequest{Prompt: "p"}); err == nil {
		t.Error("want an error without a model")
	}
}
