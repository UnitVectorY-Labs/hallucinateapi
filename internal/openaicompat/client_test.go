package openaicompat

import (
	"testing"
	"time"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/llm"
)

func TestNewClient(t *testing.T) {
	client := NewClient("gpt-4o", "", "test-key", false, 30*time.Second)
	if client.model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", client.model)
	}
	if client.url != defaultOpenAIURL {
		t.Errorf("expected default URL %q, got %q", defaultOpenAIURL, client.url)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", client.apiKey)
	}
	if client.strictSchema {
		t.Error("expected strictSchema false")
	}
}

func TestNewClientCustomURL(t *testing.T) {
	customURL := "http://localhost:11434/v1/chat/completions"
	client := NewClient("llama3", customURL, "", false, 0)
	if client.url != customURL {
		t.Errorf("expected URL %q, got %q", customURL, client.url)
	}
	if client.apiKey != "" {
		t.Errorf("expected empty apiKey, got %q", client.apiKey)
	}
}

func TestNewClientStrictSchema(t *testing.T) {
	client := NewClient("gpt-4o", "", "test-key", true, 30*time.Second)
	if !client.strictSchema {
		t.Error("expected strictSchema true")
	}
}

func TestClientImplementsInterface(t *testing.T) {
	var _ llm.Client = (*Client)(nil)
}
