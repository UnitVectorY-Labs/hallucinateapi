package openapi

import (
	"testing"
)

func TestLoadSpecYAML(t *testing.T) {
	spec, err := LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load YAML spec: %v", err)
	}

	if spec.ContentType != "yaml" {
		t.Errorf("expected content type 'yaml', got %q", spec.ContentType)
	}

	if spec.Version != "3.0.3" {
		t.Errorf("expected version '3.0.3', got %q", spec.Version)
	}

	if len(spec.Operations) == 0 {
		t.Fatal("expected operations, got none")
	}

	// Check we found the GET and POST operations
	var foundGet, foundPost bool
	for _, op := range spec.Operations {
		if op.Method == "GET" && op.Path == "/api/users/{userId}" {
			foundGet = true
			if op.OperationID != "getUserById" {
				t.Errorf("expected operationId 'getUserById', got %q", op.OperationID)
			}
			if len(op.Parameters) == 0 {
				t.Error("expected parameters for GET operation")
			}
		}
		if op.Method == "POST" && op.Path == "/api/users" {
			foundPost = true
			if op.OperationID != "createUser" {
				t.Errorf("expected operationId 'createUser', got %q", op.OperationID)
			}
			if op.RequestBody == nil {
				t.Error("expected requestBody for POST operation")
			}
		}
	}

	if !foundGet {
		t.Error("GET /api/users/{userId} operation not found")
	}
	if !foundPost {
		t.Error("POST /api/users operation not found")
	}
}

func TestLoadSpecJSON(t *testing.T) {
	spec, err := LoadSpec("../../testdata/valid_spec.json")
	if err != nil {
		t.Fatalf("failed to load JSON spec: %v", err)
	}

	if spec.ContentType != "json" {
		t.Errorf("expected content type 'json', got %q", spec.ContentType)
	}

	if len(spec.Operations) == 0 {
		t.Fatal("expected operations, got none")
	}
}

func TestLoadSpecNonExistent(t *testing.T) {
	_, err := LoadSpec("../../testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadSpecResponseSchema(t *testing.T) {
	spec, err := LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	for _, op := range spec.Operations {
		if op.Method == "GET" || op.Method == "POST" {
			if op.Response == nil {
				t.Errorf("%s %s: expected response schema", op.Method, op.Path)
			}
			if op.RawSchema == nil {
				t.Errorf("%s %s: expected raw schema", op.Method, op.Path)
			}
		}
	}
}

func TestLoadSpecParameters(t *testing.T) {
	spec, err := LoadSpec("../../testdata/valid_spec.yaml")
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	for _, op := range spec.Operations {
		if op.Method == "GET" && op.Path == "/api/users/{userId}" {
			// Should have userId (path) and limit (query)
			if len(op.Parameters) < 2 {
				t.Fatalf("expected at least 2 parameters, got %d", len(op.Parameters))
			}

			var foundPath, foundQuery bool
			for _, p := range op.Parameters {
				if p.Name == "userId" && p.In == "path" {
					foundPath = true
					if !p.Required {
						t.Error("path parameter userId should be required")
					}
				}
				if p.Name == "limit" && p.In == "query" {
					foundQuery = true
				}
			}

			if !foundPath {
				t.Error("userId path parameter not found")
			}
			if !foundQuery {
				t.Error("limit query parameter not found")
			}
		}
	}
}

func TestServeEndpoint(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"json", "/openapi.json"},
		{"yaml", "/openapi.yaml"},
	}

	for _, tt := range tests {
		s := &Spec{ContentType: tt.contentType}
		if got := s.ServeEndpoint(); got != tt.expected {
			t.Errorf("ContentType=%s: expected %q, got %q", tt.contentType, tt.expected, got)
		}
	}
}

func TestBuildOperationContext(t *testing.T) {
	op := &Operation{
		Method:      "GET",
		Path:        "/api/test",
		OperationID: "testOp",
		Summary:     "Test operation",
		Description: "A test operation",
	}

	ctx, err := BuildOperationContext(op)
	if err != nil {
		t.Fatalf("failed to build operation context: %v", err)
	}

	if ctx == "" {
		t.Error("expected non-empty context")
	}
}
