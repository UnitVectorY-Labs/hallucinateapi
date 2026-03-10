package main

import (
	"embed"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/config"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/gemini"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/llm"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/logging"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openaicompat"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/prompt"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/server"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/validation"
)

//go:embed prompts/system.txt
var systemPromptFS embed.FS

// Version is the application version, injected at build time via ldflags
var Version = "dev"

func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	// Load embedded system prompt
	data, err := systemPromptFS.ReadFile("prompts/system.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load system prompt: %v\n", err)
		os.Exit(1)
	}
	prompt.SystemPromptTemplate = string(data)

	logger := logging.New()

	rootCmd := &cobra.Command{
		Use:          "hallucinate",
		Short:        "HallucinateAPI - OpenAPI-driven LLM API gateway",
		Version:      Version,
		SilenceUsage: true,
	}

	serveCmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the HTTP server",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(logger)
		},
	}

	validateCmd := &cobra.Command{
		Use:          "validate",
		Short:        "Validate configuration and OpenAPI spec",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(logger)
		},
	}

	// Bind flags on root command (persistent flags are inherited by subcommands)
	config.BindFlags(rootCmd)
	config.BindEnvVars()

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(validateCmd)

	// Default to serve if no subcommand provided
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runServe(logger)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadAndValidate(logger *logging.Logger) (*config.Config, *openapi.Spec, *validation.ValidationResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("config validation failed: %w", err)
	}

	spec, err := openapi.LoadSpec(cfg.OpenAPIPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	logger.Info("OpenAPI spec loaded", map[string]any{
		"version":    spec.Version,
		"format":     spec.ContentType,
		"operations": len(spec.Operations),
	})

	result := validation.Validate(spec, cfg.ResolvedSchemaProfile())
	return cfg, spec, result, nil
}

func runServe(logger *logging.Logger) error {
	cfg, spec, result, err := loadAndValidate(logger)
	if err != nil {
		logger.Error("startup failed", map[string]any{"error": err.Error()})
		return err
	}

	if !result.Valid {
		logger.Error("validation failed", map[string]any{
			"errors": len(result.Errors),
		})
		fmt.Fprintln(os.Stderr, result.FormatText())
		return fmt.Errorf("validation failed with %d errors", len(result.Errors))
	}

	logger.Info("validation passed", nil)

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second

	// Create LLM client based on provider
	var llmClient llm.Client
	switch cfg.Provider {
	case "gemini":
		llmClient = gemini.NewClient(
			cfg.GCPProject,
			cfg.GCPLocation,
			cfg.Model,
			cfg.URL,
			cfg.APIKey,
			timeout,
		)
	case "openai":
		llmClient = openaicompat.NewClient(
			cfg.Model,
			cfg.URL,
			cfg.APIKey,
			cfg.StrictSchema,
			timeout,
		)
	}

	logger.Info("LLM provider configured", map[string]any{
		"provider": cfg.Provider,
		"model":    cfg.Model,
	})

	// Create and start server
	srv := server.New(cfg, spec, llmClient)

	return srv.ListenAndServe()
}

func runValidate(logger *logging.Logger) error {
	_, _, result, err := loadAndValidate(logger)
	if err != nil {
		logger.Error("validation failed", map[string]any{"error": err.Error()})
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	// Output both JSON and text
	fmt.Println(result.FormatJSON())
	fmt.Println()
	fmt.Println(result.FormatText())

	if !result.Valid {
		return fmt.Errorf("validation failed")
	}

	return nil
}
