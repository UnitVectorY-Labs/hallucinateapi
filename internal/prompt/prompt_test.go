package prompt

import (
	"testing"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
)

func init() {
	// Set the system prompt template for tests
	SystemPromptTemplate = "You are a test API. Follow instructions.\n\nOPERATION DETAILS:\n"
}

func TestBuildSystemPrompt(t *testing.T) {
	op := &openapi.Operation{
		Method:      "GET",
		Path:        "/api/test",
		OperationID: "testOp",
		Summary:     "Test operation",
		Description: "A test operation",
	}

	result, err := BuildSystemPrompt("Custom prefix", op)
	if err != nil {
		t.Fatalf("failed to build system prompt: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty system prompt")
	}

	// Should contain custom prefix
	if len(result) < len("Custom prefix") {
		t.Error("expected system prompt to contain custom prefix")
	}
}

func TestBuildSystemPromptNoPrefix(t *testing.T) {
	op := &openapi.Operation{
		Method: "GET",
		Path:   "/api/test",
	}

	result, err := BuildSystemPrompt("", op)
	if err != nil {
		t.Fatalf("failed to build system prompt: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty system prompt")
	}
}

func TestBuildUserPromptJSON(t *testing.T) {
	op := &openapi.Operation{
		Method: "GET",
		Path:   "/api/users/{userId}",
		Parameters: []openapi.Parameter{
			{Name: "userId", In: "path", Required: true},
			{Name: "limit", In: "query"},
		},
	}

	pathParams := map[string]string{"userId": "123"}
	queryParams := map[string]string{"limit": "10"}

	result, err := BuildUserPrompt(op, pathParams, queryParams, nil, "json")
	if err != nil {
		t.Fatalf("failed to build user prompt: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty user prompt")
	}

	// Should not contain any raw data not in params
	if len(result) == 0 {
		t.Error("expected non-empty user prompt")
	}
}

func TestBuildUserPromptTOON(t *testing.T) {
	op := &openapi.Operation{
		Method: "GET",
		Path:   "/api/test",
		Parameters: []openapi.Parameter{
			{Name: "q", In: "query"},
		},
	}

	queryParams := map[string]string{"q": "hello"}

	result, err := BuildUserPrompt(op, nil, queryParams, nil, "toon")
	if err != nil {
		t.Fatalf("failed to build user prompt: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty TOON prompt")
	}
}

func TestBuildUserPromptWithBody(t *testing.T) {
	op := &openapi.Operation{
		Method: "POST",
		Path:   "/api/users",
	}

	body := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	}

	result, err := BuildUserPrompt(op, nil, nil, body, "json")
	if err != nil {
		t.Fatalf("failed to build user prompt: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty user prompt with body")
	}
}

func TestBuildUserPromptOnlyIncludesAllowedParams(t *testing.T) {
	op := &openapi.Operation{
		Method: "GET",
		Path:   "/api/test",
		Parameters: []openapi.Parameter{
			{Name: "allowed", In: "query"},
		},
	}

	// Pass extra params that are not defined in the operation
	queryParams := map[string]string{
		"allowed":  "yes",
		"injected": "bad",
	}

	result, err := BuildUserPrompt(op, nil, queryParams, nil, "json")
	if err != nil {
		t.Fatalf("failed to build user prompt: %v", err)
	}

	// The "injected" param should not appear
	if contains(result, "injected") {
		t.Error("user prompt should not contain non-allowed params")
	}
	if contains(result, "bad") {
		t.Error("user prompt should not contain non-allowed param values")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
