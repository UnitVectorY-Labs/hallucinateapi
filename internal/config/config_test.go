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
			name: "valid config",
			cfg: Config{
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
			name: "missing openapi path",
			cfg: Config{
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing gcp project",
			cfg: Config{
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing gcp location",
			cfg: Config{
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			cfg: Config{
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
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.5-flash",
				PromptFormat: "json",
			},
			wantErr: false,
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
		name    string
		profile string
		want    string
	}{
		{
			name:    "default profile when empty",
			profile: "",
			want:    "GEMINI_202602",
		},
		{
			name:    "explicit MINIMAL_202602 override",
			profile: "MINIMAL_202602",
			want:    "MINIMAL_202602",
		},
		{
			name:    "explicit GEMINI_202602",
			profile: "GEMINI_202602",
			want:    "GEMINI_202602",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{SchemaProfile: tt.profile}
			got := string(cfg.ResolvedSchemaProfile())
			if got != tt.want {
				t.Errorf("ResolvedSchemaProfile() = %q, want %q", got, tt.want)
			}
		})
	}
}
