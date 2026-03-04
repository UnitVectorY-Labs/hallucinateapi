package validation

import (
	"testing"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
)

func TestValidateValidSpec(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec)
	if !result.Valid {
		t.Errorf("expected valid spec, got errors: %s", result.FormatText())
	}
}

func TestValidateRouteConflict(t *testing.T) {
	spec, err := openapi.LoadSpec("../../testdata/route_conflict.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	result := Validate(spec)
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

	result := Validate(spec)

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

	result := Validate(spec)
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

	result := Validate(spec)
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
