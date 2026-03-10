package llm

import (
	"context"
	"time"
)

// GenerateResult holds the result of an LLM generation call
type GenerateResult struct {
	Content      string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	Latency      time.Duration
}

// Client is the common interface for LLM providers
type Client interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, responseSchema interface{}) (*GenerateResult, error)
}
