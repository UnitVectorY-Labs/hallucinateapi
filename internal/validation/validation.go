package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	jsp "github.com/UnitVectorY-Labs/jsonschemaprofiles"

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

// Validate runs all validations on the parsed spec using the given schema profile
// for response schema validation via the jsonschemaprofiles library.
func Validate(spec *openapi.Spec, profile jsp.ProfileID) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Doc route conflicts
	checkRouteConflicts(spec, result)

	// Supported operations
	checkOperations(spec, result)

	// Request schema availability
	checkRequestSchemas(spec, result)

	// Response schema compatibility (uses jsonschemaprofiles)
	checkResponseSchemas(spec, result, profile)

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

// checkResponseSchemas validates response schema compatibility using the
// jsonschemaprofiles library. See https://jsonschemaprofiles.unitvectorylabs.com/schemas/gemini
// for details on schema limitations.
func checkResponseSchemas(spec *openapi.Spec, result *ValidationResult, profile jsp.ProfileID) {
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
			continue
		}

		// Validate the response schema against the selected profile using
		// the jsonschemaprofiles library.
		schemaBytes, err := json.Marshal(op.Response.Schema)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: fmt.Sprintf("Failed to serialize response schema: %v", err),
				Path:    op.Path,
				Method:  op.Method,
			})
			continue
		}

		report, err := jsp.ValidateSchema(profile, schemaBytes, nil)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "OPENAPI_INVALID",
				Message: fmt.Sprintf("Schema profile validation error: %v", err),
				Path:    op.Path,
				Method:  op.Method,
			})
			continue
		}

		if !report.Valid {
			for _, f := range report.Findings {
				if f.Severity == jsp.SeverityError {
					result.Valid = false
					msg := f.Message
					if f.Path != "" {
						msg = fmt.Sprintf("%s (at %s)", f.Message, f.Path)
					}
					result.Errors = append(result.Errors, ValidationError{
						Code:    "SCHEMA_PROFILE_VIOLATION",
						Message: msg,
						Path:    op.Path,
						Method:  op.Method,
					})
				}
			}
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
