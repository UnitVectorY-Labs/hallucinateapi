package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid gemini config",
			cfg: Config{
				Provider:        "gemini",
				OpenAPIPath:     "/path/to/spec.yaml",
				GCPProject:      "my-project",
				GCPLocation:     "us-central1",
				Model:           "gemini-2.5-flash",
				PromptFormat:    "json",
				ListenAddr:      ":8080",
				MaxRequestBytes: 10240,
				TimeoutSeconds:  300,
			},
			wantErr: false,
		},
		{
			name: "valid openai config",
			cfg: Config{
				Provider:        "openai",
				OpenAPIPath:     "/path/to/spec.yaml",
				Model:           "gpt-4o",
				PromptFormat:    "json",
				ListenAddr:      ":8080",
				MaxRequestBytes: 10240,
				TimeoutSeconds:  300,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			cfg: Config{
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			cfg: Config{
				Provider:     "anthropic",
				OpenAPIPath:  "/path/to/spec.yaml",
				Model:        "claude-3",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing openapi path",
			cfg: Config{
				Provider:     "gemini",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "gemini missing gcp project without url",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "gemini missing gcp location without url",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "gemini with url allows missing project and location",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				Model:        "gemini-2.5-flash",
				URL:          "https://custom-endpoint.example.com/v1",
				PromptFormat: "json",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "invalid prompt format",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "xml",
			},
			wantErr: true,
		},
		{
			name: "toon prompt format",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "toon",
			},
			wantErr: false,
		},
		{
			name: "valid schema profile override",
			cfg: Config{
				Provider:      "gemini",
				OpenAPIPath:   "/path/to/spec.yaml",
				GCPProject:    "my-project",
				GCPLocation:   "us-central1",
				Model:         "gemini-2.5-flash",
				PromptFormat:  "json",
				SchemaProfile: "MINIMAL_202602",
			},
			wantErr: false,
		},
		{
			name: "invalid schema profile",
			cfg: Config{
				Provider:      "gemini",
				OpenAPIPath:   "/path/to/spec.yaml",
				GCPProject:    "my-project",
				GCPLocation:   "us-central1",
				Model:         "gemini-2.5-flash",
				PromptFormat:  "json",
				SchemaProfile: "DOES_NOT_EXIST",
			},
			wantErr: true,
		},
		{
			name: "empty schema profile uses default",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: false,
		},
		{
			name: "strict-schema rejected for gemini",
			cfg: Config{
				Provider:     "gemini",
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
				StrictSchema: true,
			},
			wantErr: true,
		},
		{
			name: "strict-schema allowed for openai",
			cfg: Config{
				Provider:     "openai",
				OpenAPIPath:  "/path/to/spec.yaml",
				Model:        "gpt-4o",
				PromptFormat: "json",
				StrictSchema: true,
			},
			wantErr: false,
		},
		{
			name: "openai rejects gcp-project",
			cfg: Config{
				Provider:     "openai",
				OpenAPIPath:  "/path/to/spec.yaml",
				Model:        "gpt-4o",
				PromptFormat: "json",
				GCPProject:   "my-project",
			},
			wantErr: true,
		},
		{
			name: "openai rejects gcp-location",
			cfg: Config{
				Provider:     "openai",
				OpenAPIPath:  "/path/to/spec.yaml",
				Model:        "gpt-4o",
				PromptFormat: "json",
				GCPLocation:  "us-central1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolvedSchemaProfile(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		profile  string
		want     string
	}{
		{
			name:     "gemini default profile when empty",
			provider: "gemini",
			profile:  "",
			want:     "GEMINI_202602",
		},
		{
			name:     "openai default profile when empty",
			provider: "openai",
			profile:  "",
			want:     "OPENAI_202602",
		},
		{
			name:     "explicit MINIMAL_202602 override",
			provider: "gemini",
			profile:  "MINIMAL_202602",
			want:     "MINIMAL_202602",
		},
		{
			name:     "explicit GEMINI_202602",
			provider: "gemini",
			profile:  "GEMINI_202602",
			want:     "GEMINI_202602",
		},
		{
			name:     "openai with explicit OPENAI_202602",
			provider: "openai",
			profile:  "OPENAI_202602",
			want:     "OPENAI_202602",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Provider: tt.provider, SchemaProfile: tt.profile}
			got := string(cfg.ResolvedSchemaProfile())
			if got != tt.want {
				t.Errorf("ResolvedSchemaProfile() = %q, want %q", got, tt.want)
			}
		})
	}
}
