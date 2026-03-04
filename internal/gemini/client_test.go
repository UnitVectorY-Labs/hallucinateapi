package gemini

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-project", "us-central1", "gemini-2.0-flash", 30*time.Second)
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

func TestBuildGenerateContentURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		project  string
		location string
		model    string
		want     string
	}{
		{
			name:     "regional endpoint",
			project:  "test-project",
			location: "us-central1",
			model:    "gemini-2.5-flash",
			want:     "https://us-central1-aiplatform.googleapis.com/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.5-flash:generateContent",
		},
		{
			name:     "global endpoint",
			project:  "test-project",
			location: "global",
			model:    "gemini-2.5-flash",
			want:     "https://aiplatform.googleapis.com/v1/projects/test-project/locations/global/publishers/google/models/gemini-2.5-flash:generateContent",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildGenerateContentURL(tt.project, tt.location, tt.model)
			if got != tt.want {
				t.Fatalf("buildGenerateContentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
