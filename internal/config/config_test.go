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
				Model:           "gemini-2.0-flash",
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
				Model:        "gemini-2.0-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing gcp project",
			cfg: Config{
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPLocation:  "us-central1",
				Model:        "gemini-2.0-flash",
				PromptFormat: "json",
			},
			wantErr: true,
		},
		{
			name: "missing gcp location",
			cfg: Config{
				OpenAPIPath:  "/path/to/spec.yaml",
				GCPProject:   "my-project",
				Model:        "gemini-2.0-flash",
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
				Model:        "gemini-2.0-flash",
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
				Model:        "gemini-2.0-flash",
				PromptFormat: "toon",
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
