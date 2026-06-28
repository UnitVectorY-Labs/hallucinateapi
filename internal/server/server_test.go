package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/config"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/errutil"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/llm"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/prompt"
)

func init() {
	// Set the system prompt template for tests
	prompt.SystemPromptTemplate = "You are a test API.\n\nOPERATION DETAILS:\n"
}

// mockLLMClient implements llm.Client for testing
type mockLLMClient struct {
	response      string
	responses     []string
	err           error
	calls         int
	systemPrompts []string
	userPrompts   []string
	schemas       []any
}

func (m *mockLLMClient) Generate(_ context.Context, systemPrompt, userPrompt string, responseSchema any) (*llm.GenerateResult, error) {
	m.calls++
	m.systemPrompts = append(m.systemPrompts, systemPrompt)
	m.userPrompts = append(m.userPrompts, userPrompt)
	m.schemas = append(m.schemas, responseSchema)
	if m.err != nil {
		return nil, m.err
	}
	content := m.response
	if len(m.responses) >= m.calls {
		content = m.responses[m.calls-1]
	}
	return &llm.GenerateResult{
		Content:      content,
		PromptTokens: 10,
		OutputTokens: 5,
		TotalTokens:  15,
		Latency:      100 * time.Millisecond,
	}, nil
}

func newTestServer(t *testing.T, specPath string, mockClient *mockLLMClient) *Server {
	t.Helper()
	spec, err := openapi.LoadSpec(specPath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	cfg := &config.Config{
		Provider:        "gemini",
		ListenAddr:      ":0",
		PromptFormat:    "json",
		MaxRequestBytes: 10240,
		TimeoutSeconds:  10,
	}

	return New(cfg, spec, mockClient)
}

func newTestServerWithMode(t *testing.T, specPath string, mode config.Mode, mockClient *mockLLMClient) *Server {
	t.Helper()
	spec, err := openapi.LoadSpec(specPath)
	if err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	cfg := &config.Config{
		Provider:        "gemini",
		ListenAddr:      ":0",
		PromptFormat:    "json",
		MaxRequestBytes: 10240,
		TimeoutSeconds:  10,
		Mode:            mode,
	}

	return New(cfg, spec, mockClient)
}

func TestSwaggerUIServed(t *testing.T) {
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", &mockLLMClient{response: `{"message":"hello"}`})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected text/html content type, got %q", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("expected swagger-ui in response body")
	}
}

func TestSpecEndpointServed(t *testing.T) {
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", &mockLLMClient{response: `{"message":"hello"}`})

	req := httptest.NewRequest("GET", "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi") {
		t.Error("expected OpenAPI content in spec endpoint response")
	}
}

func TestAPIEndpointGETSuccess(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"message":"hello world"}`,
	}
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", mockClient)

	req := httptest.NewRequest("GET", "/api/hello", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["message"] != "hello world" {
		t.Errorf("expected message 'hello world', got %v", resp["message"])
	}
}

func TestAPIEndpointGETTwoPassReturnsSelectedStatus(t *testing.T) {
	mockClient := &mockLLMClient{
		responses: []string{
			`{"statusCode":"404"}`,
			`{"error":"user not found"}`,
		},
	}
	srv := newTestServerWithMode(t, "../../testdata/multi_response.yaml", config.ModeTwoPass, mockClient)

	req := httptest.NewRequest("GET", "/users/does-not-exist", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", w.Code, w.Body.String())
	}
	if mockClient.calls != 2 {
		t.Fatalf("expected two LLM calls, got %d", mockClient.calls)
	}
}

func TestAPIEndpointGETTwoPassSelectionSchemaUsesEnum(t *testing.T) {
	mockClient := &mockLLMClient{
		responses: []string{
			`{"statusCode":"200"}`,
			`{"id":"123"}`,
		},
	}
	srv := newTestServerWithMode(t, "../../testdata/multi_response.yaml", config.ModeTwoPass, mockClient)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if len(mockClient.schemas) < 1 {
		t.Fatal("expected at least one schema capture")
	}

	selectionSchema, ok := mockClient.schemas[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first schema to be a map, got %T", mockClient.schemas[0])
	}
	properties, ok := selectionSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in selection schema")
	}
	statusProp, ok := properties["statusCode"].(map[string]any)
	if !ok {
		t.Fatal("expected statusCode property in selection schema")
	}
	enumValues, ok := statusProp["enum"].([]any)
	if !ok {
		t.Fatal("expected enum in statusCode schema")
	}
	if len(enumValues) != 2 {
		t.Fatalf("expected 2 enum values, got %d", len(enumValues))
	}
}

func TestAPIEndpointGETTwoPassSecondPromptIncludesSelectionContext(t *testing.T) {
	mockClient := &mockLLMClient{
		responses: []string{
			`{"statusCode":"404"}`,
			`{"error":"missing"}`,
		},
	}
	srv := newTestServerWithMode(t, "../../testdata/multi_response.yaml", config.ModeTwoPass, mockClient)

	req := httptest.NewRequest("GET", "/users/does-not-exist", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if len(mockClient.userPrompts) < 2 {
		t.Fatalf("expected two prompts, got %d", len(mockClient.userPrompts))
	}
	secondPrompt := mockClient.userPrompts[1]
	if !strings.Contains(secondPrompt, "selectedResponseType") {
		t.Fatalf("expected second prompt to include selectedResponseType context, got %q", secondPrompt)
	}
	if !strings.Contains(secondPrompt, `"statusCode":"404"`) {
		t.Fatalf("expected second prompt to include selected status code, got %q", secondPrompt)
	}
}

func TestAPIEndpointPOSTSuccess(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"id":"1","name":"Alice","email":"alice@test.com"}`,
	}
	srv := newTestServer(t, "../../testdata/valid_spec.yaml", mockClient)

	body := `{"name":"Alice","email":"alice@test.com"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestPOSTMissingContentType(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"id":"1"}`,
	}
	srv := newTestServer(t, "../../testdata/valid_spec.yaml", mockClient)

	body := `{"name":"Alice"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	// No Content-Type header
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var apiErr errutil.APIError
	json.Unmarshal(w.Body.Bytes(), &apiErr)
	if apiErr.Error.Code != errutil.CodeContentTypeUnsupported {
		t.Errorf("expected error code %q, got %q", errutil.CodeContentTypeUnsupported, apiErr.Error.Code)
	}
}

func TestPOSTInvalidJSON(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"id":"1"}`,
	}
	srv := newTestServer(t, "../../testdata/valid_spec.yaml", mockClient)

	req := httptest.NewRequest("POST", "/api/users", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPOSTRejectsUnknownRequestBodyProperties(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"sentiment":"NEUTRAL","confidence":0.5}`,
	}
	srv := newTestServer(t, "../../example/sentiment.yaml", mockClient)

	body := `{"message":"that was not terrible","test":2}`
	req := httptest.NewRequest("POST", "/sentiment", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}

	if mockClient.calls != 0 {
		t.Fatalf("expected LLM not to be called, got %d calls", mockClient.calls)
	}

	var apiErr errutil.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if apiErr.Error.Code != errutil.CodeRequestValidationFailed {
		t.Fatalf("expected error code %q, got %q", errutil.CodeRequestValidationFailed, apiErr.Error.Code)
	}

	if !strings.Contains(apiErr.Error.Message, "Additional property test is not allowed") {
		t.Fatalf("expected additionalProperties validation error, got %q", apiErr.Error.Message)
	}
}

func TestPOSTRejectsMissingRequiredRequestBodyProperty(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"sentiment":"NEUTRAL","confidence":0.5}`,
	}
	srv := newTestServer(t, "../../example/sentiment.yaml", mockClient)

	body := `{}`
	req := httptest.NewRequest("POST", "/sentiment", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}

	if mockClient.calls != 0 {
		t.Fatalf("expected LLM not to be called, got %d calls", mockClient.calls)
	}

	var apiErr errutil.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if apiErr.Error.Code != errutil.CodeRequestValidationFailed {
		t.Fatalf("expected error code %q, got %q", errutil.CodeRequestValidationFailed, apiErr.Error.Code)
	}

	if !strings.Contains(apiErr.Error.Message, "message is required") {
		t.Fatalf("expected required-property validation error, got %q", apiErr.Error.Message)
	}
}

func TestUnknownQueryParamsRejected(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"id":"1","name":"Alice","email":"a@b.com"}`,
	}
	srv := newTestServer(t, "../../testdata/valid_spec.yaml", mockClient)

	req := httptest.NewRequest("GET", "/api/users/123?limit=10&unknown=bad", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown query params, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestLLMErrorReturns502(t *testing.T) {
	mockClient := &mockLLMClient{
		err: fmt.Errorf("LLM API error"),
	}
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", mockClient)

	req := httptest.NewRequest("GET", "/api/hello", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestLLMInvalidJSONResponse(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `not json at all`,
	}
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", mockClient)

	req := httptest.NewRequest("GET", "/api/hello", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for invalid JSON response, got %d", w.Code)
	}
}

func TestContentTypeWithCharset(t *testing.T) {
	mockClient := &mockLLMClient{
		response: `{"id":"1","name":"Alice","email":"a@b.com"}`,
	}
	srv := newTestServer(t, "../../testdata/valid_spec.yaml", mockClient)

	body := `{"name":"Alice","email":"alice@test.com"}`
	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with charset, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestNotFoundReturns404(t *testing.T) {
	srv := newTestServer(t, "../../testdata/minimal_get.yaml", &mockLLMClient{response: `{"message":"hello"}`})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		template string
		path     string
		want     bool
	}{
		{"/api/users/{userId}", "/api/users/123", true},
		{"/api/users/{userId}", "/api/users/abc", true},
		{"/api/users/{userId}", "/api/users", false},
		{"/api/users", "/api/users", true},
		{"/api/users", "/api/other", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.template, tt.path), func(t *testing.T) {
			got := matchPath(tt.template, tt.path)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.template, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"Application/JSON", true},
		{"text/html", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			got := isJSONContentType(tt.ct)
			if got != tt.want {
				t.Errorf("isJSONContentType(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}
