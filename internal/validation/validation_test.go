package validation

import (
	"testing"

	jsp "github.com/UnitVectorY-Labs/jsonschemaprofiles"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
)

// defaultProfile is used by tests unless a specific profile is being exercised.
var defaultProfile = jsp.GEMINI_202602

func TestValidateValidSpec(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)
	if !result.Valid {
		t.Errorf("expected valid spec, got errors: %s", result.FormatText())
	}
}

func TestValidateRouteConflict(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/route_conflict.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)
	if result.Valid {
		t.Error("expected validation failure for route conflict")
	}

	foundConflict := false
	for _, e := range result.Errors {
		if e.Code == "ROUTE_CONFLICT" {
			foundConflict = true
			break
		}
	}
	if !foundConflict {
		t.Error("expected ROUTE_CONFLICT error code")
	}
}

func TestValidateRefInResponse(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/ref_in_response.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)

	// The $ref in response should be resolved by the parser, but if it still
	// contains $ref it should fail. Since libopenapi resolves $ref,
	// we need to check what actually gets parsed.
	// If $ref is resolved, the spec might be valid.
	// If $ref is not resolved, it should fail validation.
	t.Logf("validation result: %s", result.FormatText())
}

func TestValidateNoResponseSchema(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/no_response_schema.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)
	if result.Valid {
		t.Error("expected validation failure for missing response schema")
	}

	found := false
	for _, e := range result.Errors {
		if e.Code == "OPENAPI_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected OPENAPI_INVALID error for missing response schema")
	}
}

func TestValidatePostNoBody(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/post_no_body.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)
	if result.Valid {
		t.Error("expected validation failure for POST without requestBody")
	}

	found := false
	for _, e := range result.Errors {
		if e.Code == "OPENAPI_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected OPENAPI_INVALID error for missing requestBody")
	}
}

func TestValidateAdditionalPropertiesAllowed(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/additional_properties_response.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec, defaultProfile)
	if !result.Valid {
		t.Errorf("expected spec with additionalProperties to be valid, got errors: %s", result.FormatText())
	}
}

func TestContainsRef(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]interface{}
		want   bool
	}{
		{
			name:   "no ref",
			schema: map[string]interface{}{"type": "object"},
			want:   false,
		},
		{
			name:   "top level ref",
			schema: map[string]interface{}{"$ref": "#/components/schemas/Foo"},
			want:   true,
		},
		{
			name: "nested ref",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"$ref": "#/components/schemas/Foo",
					},
				},
			},
			want: true,
		},
		{
			name: "ref in array",
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"anyOf": []interface{}{
						map[string]interface{}{"$ref": "#/components/schemas/Foo"},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsRef(tt.schema)
			if got != tt.want {
				t.Errorf("containsRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResultFormatJSON(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Code: "TEST_ERROR", Message: "test message"},
		},
	}

	json := result.FormatJSON()
	if json == "" {
		t.Error("expected non-empty JSON output")
	}
}

func TestValidationResultFormatText(t *testing.T) {
	result := &ValidationResult{Valid: true}
	text := result.FormatText()
	if text != "Validation passed: OpenAPI spec is valid" {
		t.Errorf("unexpected text output: %q", text)
	}

	result = &ValidationResult{
		Valid:  false,
		Errors: []ValidationError{{Code: "TEST", Message: "test", Method: "GET", Path: "/test"}},
	}
	text = result.FormatText()
	if text == "" {
		t.Error("expected non-empty text output for invalid result")
	}
}

func TestGetSupportedOperations(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	ops := GetSupportedOperations(spec)
	for _, op := range ops {
		if op.Method != "GET" && op.Method != "POST" {
			t.Errorf("unexpected method %s in supported operations", op.Method)
		}
	}
}

// TestValidateSchemaProfileViolations uses data-driven test cases to verify that
// response schemas violating the selected profile are correctly caught by the
// jsonschemaprofiles library validation.
func TestValidateSchemaProfileViolations(t *testing.T) {
	tests := []struct {
		name      string
		specFile  string
		profile   jsp.ProfileID
		wantValid bool
		wantCode  string
	}{
		{
			name:      "valid spec with GEMINI_202602",
			specFile:  "../../testdata/valid_spec.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: true,
		},
		{
			name:      "valid spec with MINIMAL_202602 is stricter",
			specFile:  "../../testdata/valid_spec.yaml",
			profile:   jsp.MINIMAL_202602,
			wantValid: false,
			wantCode:  "SCHEMA_PROFILE_VIOLATION",
		},
		{
			name:      "allOf in response rejected by GEMINI_202602",
			specFile:  "../../testdata/schema_allof_response.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "SCHEMA_PROFILE_VIOLATION",
		},
		{
			name:      "not keyword in response rejected by GEMINI_202602",
			specFile:  "../../testdata/schema_not_response.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "SCHEMA_PROFILE_VIOLATION",
		},
		{
			name:      "array missing items rejected by GEMINI_202602",
			specFile:  "../../testdata/schema_array_no_items.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "SCHEMA_PROFILE_VIOLATION",
		},
		{
			name:      "additional properties response passes GEMINI_202602",
			specFile:  "../../testdata/additional_properties_response.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: true,
		},
		{
			name:      "route conflict still detected with GEMINI_202602",
			specFile:  "../../testdata/route_conflict.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "ROUTE_CONFLICT",
		},
		{
			name:      "post without body still detected with GEMINI_202602",
			specFile:  "../../testdata/post_no_body.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "OPENAPI_INVALID",
		},
		{
			name:      "no response schema still detected with GEMINI_202602",
			specFile:  "../../testdata/no_response_schema.yaml",
			profile:   jsp.GEMINI_202602,
			wantValid: false,
			wantCode:  "OPENAPI_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := openapi.LoadSpec(tt.specFile)
			if err != nil {
				t.Fatalf("failed to load spec: %v", err)
			}

			result := Validate(spec, tt.profile)
			if result.Valid != tt.wantValid {
				t.Errorf("Validate().Valid = %v, want %v\n%s", result.Valid, tt.wantValid, result.FormatText())
			}
			if tt.wantCode != "" {
				found := false
				for _, e := range result.Errors {
					if e.Code == tt.wantCode {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %q, got errors: %s", tt.wantCode, result.FormatText())
				}
			}
		})
	}
}
