package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	OpenAPIPath     string `mapstructure:"openapi-path"`
	GCPProject      string `mapstructure:"gcp-project"`
	GCPLocation     string `mapstructure:"gcp-location"`
	Model           string `mapstructure:"model"`
	ListenAddr      string `mapstructure:"listen-addr"`
	SystemPrefix    string `mapstructure:"base-system-prefix"`
	PromptFormat    string `mapstructure:"prompt-format"`
	MaxRequestBytes int64  `mapstructure:"max-request-bytes"`
	TimeoutSeconds  int    `mapstructure:"timeout-seconds"`
}

// BindFlags registers CLI flags and maps env vars on a persistent flag set
func BindFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.String("openapi-path", "", "Path to OpenAPI specification file")
	f.String("gcp-project", "", "GCP project ID")
	f.String("gcp-location", "", "Vertex AI location (e.g. us-central1)")
	f.String("model", "", "Gemini model name (e.g. gemini-2.5-flash)")
	f.String("listen-addr", ":8080", "Address to listen on")
	f.String("base-system-prefix", "", "System prompt prefix for all operations")
	f.String("prompt-format", "json", "Prompt format: json or toon")
	f.Int64("max-request-bytes", 10240, "Maximum request body size in bytes")
	f.Int("timeout-seconds", 300, "Outbound Gemini call timeout in seconds")

	viper.BindPFlag("openapi-path", f.Lookup("openapi-path"))
	viper.BindPFlag("gcp-project", f.Lookup("gcp-project"))
	viper.BindPFlag("gcp-location", f.Lookup("gcp-location"))
	viper.BindPFlag("model", f.Lookup("model"))
	viper.BindPFlag("listen-addr", f.Lookup("listen-addr"))
	viper.BindPFlag("base-system-prefix", f.Lookup("base-system-prefix"))
	viper.BindPFlag("prompt-format", f.Lookup("prompt-format"))
	viper.BindPFlag("max-request-bytes", f.Lookup("max-request-bytes"))
	viper.BindPFlag("timeout-seconds", f.Lookup("timeout-seconds"))
}

// BindEnvVars maps environment variables to config keys
func BindEnvVars() {
	viper.BindEnv("openapi-path", "HALLUCINATE_OPENAPI_PATH")
	viper.BindEnv("gcp-project", "GOOGLE_CLOUD_PROJECT", "HALLUCINATE_GCP_PROJECT")
	viper.BindEnv("gcp-location", "HALLUCINATE_GCP_LOCATION")
	viper.BindEnv("model", "HALLUCINATE_MODEL")
	viper.BindEnv("listen-addr", "HALLUCINATE_LISTEN_ADDR")
	viper.BindEnv("base-system-prefix", "HALLUCINATE_SYSTEM_PREFIX")
	viper.BindEnv("prompt-format", "HALLUCINATE_PROMPT_FORMAT")
	viper.BindEnv("max-request-bytes", "HALLUCINATE_MAX_REQUEST_BYTES")
	viper.BindEnv("timeout-seconds", "HALLUCINATE_TIMEOUT_SECONDS")
}

// Load loads the config from viper
func Load() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for fields viper doesn't handle via unmarshal
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.PromptFormat == "" {
		cfg.PromptFormat = "json"
	}
	if cfg.MaxRequestBytes == 0 {
		cfg.MaxRequestBytes = 10240
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 300
	}

	return cfg, nil
}

// Validate checks that required fields are present
func (c *Config) Validate() error {
	if c.OpenAPIPath == "" {
		return fmt.Errorf("openapi-path is required")
	}
	if c.GCPProject == "" {
		return fmt.Errorf("gcp-project is required")
	}
	if c.GCPLocation == "" {
		return fmt.Errorf("gcp-location is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.PromptFormat != "json" && c.PromptFormat != "toon" {
		return fmt.Errorf("prompt-format must be 'json' or 'toon', got %q", c.PromptFormat)
	}
	return nil
}
