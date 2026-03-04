package gemini

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-project", "us-central1", "gemini-2.0-flash", 30)
	if client.project != "test-project" {
		t.Errorf("expected project 'test-project', got %q", client.project)
	}
	if client.location != "us-central1" {
		t.Errorf("expected location 'us-central1', got %q", client.location)
	}
	if client.model != "gemini-2.0-flash" {
		t.Errorf("expected model 'gemini-2.0-flash', got %q", client.model)
	}
}

func TestClientImplementsInterface(t *testing.T) {
	var _ GeminiClientInterface = (*Client)(nil)
}
