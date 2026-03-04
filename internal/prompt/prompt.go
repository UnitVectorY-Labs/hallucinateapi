package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/toon"
)

// SystemPromptTemplate is set from the embedded prompts at the top level
var SystemPromptTemplate string

// BuildSystemPrompt constructs the system prompt for an operation
func BuildSystemPrompt(customPrefix string, op *openapi.Operation) (string, error) {
	var sb strings.Builder

	// 1. Custom prefix if provided
	if customPrefix != "" {
		sb.WriteString(customPrefix)
		sb.WriteString("\n\n")
	}

	// 2. System prompt template
	sb.WriteString(SystemPromptTemplate)

	// 3. Operation context
	ctx, err := openapi.BuildOperationContext(op)
	if err != nil {
		return "", fmt.Errorf("failed to build operation context: %w", err)
	}
	sb.WriteString(ctx)

	return sb.String(), nil
}

// BuildUserPrompt constructs the user prompt from request data
func BuildUserPrompt(op *openapi.Operation, pathParams, queryParams map[string]string, body interface{}, format string) (string, error) {
	payload := make(map[string]interface{})

	// Build params object from defined parameters only
	if len(pathParams) > 0 {
		params := make(map[string]interface{})
		for _, p := range op.Parameters {
			if p.In == "path" {
				if v, ok := pathParams[p.Name]; ok {
					params[p.Name] = v
				}
			}
		}
		if len(params) > 0 {
			payload["pathParameters"] = params
		}
	}

	if len(queryParams) > 0 {
		params := make(map[string]interface{})
		for _, p := range op.Parameters {
			if p.In == "query" {
				if v, ok := queryParams[p.Name]; ok {
					params[p.Name] = v
				}
			}
		}
		if len(params) > 0 {
			payload["queryParameters"] = params
		}
	}

	if body != nil {
		payload["body"] = body
	}

	switch format {
	case "toon":
		return toon.Serialize(payload), nil
	default:
		// JSON format - minified
		data, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal user prompt: %w", err)
		}
		return string(data), nil
	}
}
