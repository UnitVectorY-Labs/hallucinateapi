package llm

import (
	"context"
	"testing"
	"time"
)

// mockClient is a simple mock for testing
type mockClient struct{}

func (m *mockClient) Generate(_ context.Context, _, _ string, _ interface{}) (*GenerateResult, error) {
	return &GenerateResult{
		Content:      `{"test": true}`,
		PromptTokens: 10,
		OutputTokens: 5,
		TotalTokens:  15,
		Latency:      100 * time.Millisecond,
	}, nil
}

func TestClientInterface(t *testing.T) {
	var client Client = &mockClient{}
	result, err := client.Generate(context.Background(), "system", "user", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != `{"test": true}` {
		t.Errorf("expected content '{\"test\": true}', got %q", result.Content)
	}
	if result.PromptTokens != 10 {
		t.Errorf("expected PromptTokens 10, got %d", result.PromptTokens)
	}
	if result.OutputTokens != 5 {
		t.Errorf("expected OutputTokens 5, got %d", result.OutputTokens)
	}
	if result.TotalTokens != 15 {
		t.Errorf("expected TotalTokens 15, got %d", result.TotalTokens)
	}
}
