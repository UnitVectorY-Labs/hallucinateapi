package config

import (
	"fmt"

	jsp "github.com/UnitVectorY-Labs/jsonschemaprofiles"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Provider        string `mapstructure:"provider"`
	OpenAPIPath     string `mapstructure:"openapi-path"`
	GCPProject      string `mapstructure:"gcp-project"`
	GCPLocation     string `mapstructure:"gcp-location"`
	Model           string `mapstructure:"model"`
	URL             string `mapstructure:"url"`
	APIKey          string `mapstructure:"api-key"`
	StrictSchema    bool   `mapstructure:"strict-schema"`
	ListenAddr      string `mapstructure:"listen-addr"`
	SystemPrefix    string `mapstructure:"base-system-prefix"`
	PromptFormat    string `mapstructure:"prompt-format"`
	MaxRequestBytes int64  `mapstructure:"max-request-bytes"`
	TimeoutSeconds  int    `mapstructure:"timeout-seconds"`
	SchemaProfile   string `mapstructure:"schema-profile"`
	Insecure        bool   `mapstructure:"insecure"`
}

// BindFlags registers CLI flags and maps env vars on a persistent flag set
func BindFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.String("provider", "", "LLM provider: gemini or openai (required)")
	f.String("openapi-path", "", "Path to OpenAPI specification file")
	f.String("gcp-project", "", "GCP project ID (gemini only)")
	f.String("gcp-location", "", "Vertex AI location (e.g. us-central1) (gemini only)")
	f.String("model", "", "Model name (e.g. gemini-2.5-flash, gpt-4o)")
	f.String("url", "", "Override default API URL")
	f.String("api-key", "", "API key for bearer auth")
	f.Bool("strict-schema", false, "Enable strict mode for JSON schema (openai only)")
	f.String("listen-addr", ":8080", "Address to listen on")
	f.String("base-system-prefix", "", "System prompt prefix for all operations")
	f.String("prompt-format", "json", "Prompt format: json or toon")
	f.Int64("max-request-bytes", 10240, "Maximum request body size in bytes")
	f.Int("timeout-seconds", 300, "Outbound LLM API call timeout in seconds")
	f.String("schema-profile", "", "Schema profile override for response schema validation (e.g. GEMINI_202602, OPENAI_202602, MINIMAL_202602)")
	f.Bool("insecure", false, "Skip TLS certificate verification for outbound LLM calls")

	viper.BindPFlag("provider", f.Lookup("provider"))
	viper.BindPFlag("openapi-path", f.Lookup("openapi-path"))
	viper.BindPFlag("gcp-project", f.Lookup("gcp-project"))
	viper.BindPFlag("gcp-location", f.Lookup("gcp-location"))
	viper.BindPFlag("model", f.Lookup("model"))
	viper.BindPFlag("url", f.Lookup("url"))
	viper.BindPFlag("api-key", f.Lookup("api-key"))
	viper.BindPFlag("strict-schema", f.Lookup("strict-schema"))
	viper.BindPFlag("listen-addr", f.Lookup("listen-addr"))
	viper.BindPFlag("base-system-prefix", f.Lookup("base-system-prefix"))
	viper.BindPFlag("prompt-format", f.Lookup("prompt-format"))
	viper.BindPFlag("max-request-bytes", f.Lookup("max-request-bytes"))
	viper.BindPFlag("timeout-seconds", f.Lookup("timeout-seconds"))
	viper.BindPFlag("schema-profile", f.Lookup("schema-profile"))
	viper.BindPFlag("insecure", f.Lookup("insecure"))
}

// BindEnvVars maps environment variables to config keys
func BindEnvVars() {
	viper.BindEnv("provider", "HALLUCINATE_PROVIDER")
	viper.BindEnv("openapi-path", "HALLUCINATE_OPENAPI_PATH")
	viper.BindEnv("gcp-project", "GOOGLE_CLOUD_PROJECT", "HALLUCINATE_GCP_PROJECT")
	viper.BindEnv("gcp-location", "HALLUCINATE_GCP_LOCATION")
	viper.BindEnv("model", "HALLUCINATE_MODEL")
	viper.BindEnv("url", "HALLUCINATE_URL")
	viper.BindEnv("api-key", "OPENAI_API_KEY", "HALLUCINATE_API_KEY")
	viper.BindEnv("strict-schema", "HALLUCINATE_STRICT_SCHEMA")
	viper.BindEnv("listen-addr", "HALLUCINATE_LISTEN_ADDR")
	viper.BindEnv("base-system-prefix", "HALLUCINATE_SYSTEM_PREFIX")
	viper.BindEnv("prompt-format", "HALLUCINATE_PROMPT_FORMAT")
	viper.BindEnv("max-request-bytes", "HALLUCINATE_MAX_REQUEST_BYTES")
	viper.BindEnv("timeout-seconds", "HALLUCINATE_TIMEOUT_SECONDS")
	viper.BindEnv("schema-profile", "HALLUCINATE_SCHEMA_PROFILE")
	viper.BindEnv("insecure", "HALLUCINATE_INSECURE")
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

// Validate checks that required fields are present and provider-specific rules are met
func (c *Config) Validate() error {
	if c.Provider != "gemini" && c.Provider != "openai" {
		return fmt.Errorf("provider must be 'gemini' or 'openai', got %q", c.Provider)
	}
	if c.OpenAPIPath == "" {
		return fmt.Errorf("openapi-path is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.PromptFormat != "json" && c.PromptFormat != "toon" {
		return fmt.Errorf("prompt-format must be 'json' or 'toon', got %q", c.PromptFormat)
	}

	// Provider-specific validation
	switch c.Provider {
	case "gemini":
		if c.StrictSchema {
			return fmt.Errorf("--strict-schema is not supported with the gemini provider")
		}
		if c.URL == "" {
			if c.GCPProject == "" {
				return fmt.Errorf("gcp-project is required for the gemini provider (unless --url is provided)")
			}
			if c.GCPLocation == "" {
				return fmt.Errorf("gcp-location is required for the gemini provider (unless --url is provided)")
			}
		}
	case "openai":
		if c.GCPProject != "" {
			return fmt.Errorf("--gcp-project is not supported with the openai provider")
		}
		if c.GCPLocation != "" {
			return fmt.Errorf("--gcp-location is not supported with the openai provider")
		}
	}

	if c.SchemaProfile != "" {
		if _, err := jsp.GetProfileInfo(jsp.ProfileID(c.SchemaProfile)); err != nil {
			return fmt.Errorf("schema-profile is invalid: %w", err)
		}
	}
	return nil
}

// ResolvedSchemaProfile returns the effective schema profile ID.
// If SchemaProfile is set it is used as the override; otherwise the default
// profile is selected based on the provider: OPENAI_202602 for openai,
// GEMINI_202602 for gemini.
func (c *Config) ResolvedSchemaProfile() jsp.ProfileID {
	if c.SchemaProfile != "" {
		return jsp.ProfileID(c.SchemaProfile)
	}
	if c.Provider == "openai" {
		return jsp.OPENAI_202602
	}
	// Default to Gemini profile (provider is validated to be "gemini" or "openai")
	return jsp.GEMINI_202602
}
