package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
)

// ValidationError represents a single validation failure
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Method  string `json:"method,omitempty"`
}

// ValidationResult holds all validation results
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Validate runs all validations on the parsed spec
func Validate(spec *openapi.Spec) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// 5.2 Doc route conflicts
	checkRouteConflicts(spec, result)

	// 5.3 Supported operations
	checkOperations(spec, result)

	// 5.4 Request schema availability
	checkRequestSchemas(spec, result)

	// 5.5 Response schema compatibility
	checkResponseSchemas(spec, result)

	return result
}

// checkRouteConflicts ensures OpenAPI doesn't define operations at reserved paths
func checkRouteConflicts(spec *openapi.Spec, result *ValidationResult) {
	reserved := map[string]bool{
		"/":             true,
		"/openapi.json": true,
		"/openapi.yaml": true,
	}

	for _, op := range spec.Operations {
		if reserved[op.Path] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "ROUTE_CONFLICT",
				Message: fmt.Sprintf("OpenAPI spec defines operation at reserved path %s", op.Path),
				Path:    op.Path,
				Method:  op.Method,
			})
		}
	}
}

// checkOperations validates that operations use supported methods
func checkOperations(spec *openapi.Spec, result *ValidationResult) {
	for _, op := range spec.Operations {
		if op.Method != "GET" && op.Method != "POST" {
			// Non-GET/POST methods are noted but allowed (will return 405 at runtime)
			// We don't fail validation for these, they are just flagged
		}
	}
}

// checkRequestSchemas validates request schema availability
func checkRequestSchemas(spec *openapi.Spec, result *ValidationResult) {
	for _, op := range spec.Operations {
		if op.Method != "GET" && op.Method != "POST" {
			continue
		}

		if op.Method == "POST" {
			if op.RequestBody == nil || op.RequestBody.Schema == nil {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Code:    "OPENAPI_INVALID",
					Message: fmt.Sprintf("POST operation must define application/json requestBody schema"),
					Path:    op.Path,
					Method:  op.Method,
				})
			}
		}
	}
}

// checkResponseSchemas validates response schema compatibility with Gemini
func checkResponseSchemas(spec *openapi.Spec, result *ValidationResult) {
	for _, op := range spec.Operations {
		if op.Method != "GET" && op.Method != "POST" {
			continue
		}

		if op.Response == nil || op.Response.Schema == nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: "Operation must define HTTP 200 response with application/json schema",
				Path:    op.Path,
				Method:  op.Method,
			})
			continue
		}

		// The server sends only the response schema fragment to Gemini, so
		// component references must already be resolved into an inline schema.
		if containsRef(op.Response.Schema) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: "Response schema must be fully inline; hallucinateapi sends the response schema to Gemini as a standalone fragment",
				Path:    op.Path,
				Method:  op.Method,
			})
		}

		// Check for unsupported JSON Schema keywords
		checkUnsupportedKeywords(op.Response.Schema, op.Path, op.Method, result)

		// Check schema depth
		depth := schemaDepth(op.Response.Schema)
		if depth > 10 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: fmt.Sprintf("Response schema depth %d exceeds maximum of 10", depth),
				Path:    op.Path,
				Method:  op.Method,
			})
		}
	}
}

// containsRef checks if a schema map contains $ref anywhere
func containsRef(schema map[string]interface{}) bool {
	for k, v := range schema {
		if k == "$ref" {
			return true
		}
		switch val := v.(type) {
		case map[string]interface{}:
			if containsRef(val) {
				return true
			}
		case []interface{}:
			for _, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					if containsRef(m) {
						return true
					}
				}
			}
		}
	}
	return false
}

var restrictedKeywords = map[string]string{
	"oneOf": `Response schema contains keyword "oneOf"; Gemini treats oneOf the same as anyOf, and hallucinateapi does not preserve oneOf semantics`,
	"allOf": `Response schema contains keyword "allOf" unsupported by Gemini structured outputs`,
	"not":   `Response schema contains keyword "not" unsupported by Gemini structured outputs`,
}

// checkUnsupportedKeywords checks keywords that hallucinateapi intentionally
// rejects for response-schema generation.
func checkUnsupportedKeywords(schema map[string]interface{}, path, method string, result *ValidationResult) {
	for k, v := range schema {
		if strings.HasPrefix(k, "x-") {
			continue
		}
		if msg, ok := restrictedKeywords[k]; ok {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: msg,
				Path:    path,
				Method:  method,
			})
		}

		walkNestedSchemas(k, v, func(child map[string]interface{}) {
			checkUnsupportedKeywords(child, path, method, result)
		})
	}
}

// schemaDepth calculates the nesting depth of a schema
func schemaDepth(schema map[string]interface{}) int {
	maxDepth := 1
	for k, v := range schema {
		walkNestedSchemas(k, v, func(child map[string]interface{}) {
			d := schemaDepth(child) + 1
			if d > maxDepth {
				maxDepth = d
			}
		})
	}
	return maxDepth
}

func walkNestedSchemas(keyword string, value interface{}, visit func(map[string]interface{})) {
	switch keyword {
	case "properties", "$defs":
		if m, ok := value.(map[string]interface{}); ok {
			for _, rawChild := range m {
				if child, ok := rawChild.(map[string]interface{}); ok {
					visit(child)
				}
			}
		}
	case "items", "additionalProperties":
		if child, ok := value.(map[string]interface{}); ok {
			visit(child)
		}
	case "anyOf", "oneOf", "allOf", "prefixItems":
		if items, ok := value.([]interface{}); ok {
			for _, rawChild := range items {
				if child, ok := rawChild.(map[string]interface{}); ok {
					visit(child)
				}
			}
		}
	}
}

// FormatJSON returns validation result as JSON
func (r *ValidationResult) FormatJSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// FormatText returns validation result as human-readable text
func (r *ValidationResult) FormatText() string {
	if r.Valid {
		return "Validation passed: OpenAPI spec is valid"
	}
	var sb strings.Builder
	sb.WriteString("Validation failed:\n")
	for i, e := range r.Errors {
		sb.WriteString(fmt.Sprintf("  %d. [%s]", i+1, e.Code))
		if e.Method != "" && e.Path != "" {
			sb.WriteString(fmt.Sprintf(" %s %s:", e.Method, e.Path))
		}
		sb.WriteString(fmt.Sprintf(" %s\n", e.Message))
	}
	return sb.String()
}

// GetSupportedOperations returns only GET and POST operations
func GetSupportedOperations(spec *openapi.Spec) []*openapi.Operation {
	var supported []*openapi.Operation
	for _, op := range spec.Operations {
		if op.Method == "GET" || op.Method == "POST" {
			supported = append(supported, op)
		}
	}
	return supported
}

// AllPaths returns all unique paths with their defined methods
func AllPaths(spec *openapi.Spec) map[string][]string {
	paths := make(map[string][]string)
	for _, op := range spec.Operations {
		paths[op.Path] = append(paths[op.Path], op.Method)
	}
	return paths
}
