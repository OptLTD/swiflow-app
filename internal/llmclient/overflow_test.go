package llmclient

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsContextOverflow(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errors.New("network down"), false},
		{"code", &APIError{StatusCode: 400, Body: `{"error":{"code":"context_length_exceeded"}}`}, true},
		{"max length", &APIError{StatusCode: 400, Body: "maximum context length is 128000 tokens"}, true},
		{"prompt too long", errors.New("prompt is too long"), true},
		{"too many tokens", &APIError{StatusCode: 400, Body: "too many tokens in request"}, true},
		{"auth", &APIError{StatusCode: 401, Body: "invalid api key"}, false},
		{"rate limit", &APIError{StatusCode: 429, Body: "rate limited"}, false},
		{"wrapped", fmt.Errorf("chat: %w", &APIError{StatusCode: 400, Body: "Prompt too long for model"}), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsContextOverflow(tc.err); got != tc.want {
				t.Fatalf("IsContextOverflow(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRetryableLLMErrorSkipsOverflow(t *testing.T) {
	err := &APIError{StatusCode: 400, Body: "context_length_exceeded"}
	if retryableLLMError(err) {
		t.Fatal("context overflow must not be treated as retryable network error")
	}
}
