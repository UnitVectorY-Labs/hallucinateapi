package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/config"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/errutil"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/llm"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/logging"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/prompt"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/swagger"
	"github.com/UnitVectorY-Labs/hallucinateapi/internal/validation"
)

// Server is the HTTP API server
type Server struct {
	cfg            *config.Config
	spec           *openapi.Spec
	operations     []*openapi.Operation
	allPaths       map[string][]string
	requestSchemas map[*openapi.Operation]requestSchemaValidator
	llmClient      llm.Client
	logger         *logging.Logger
	mux            *http.ServeMux
	requestID      uint64
}

// New creates a new server instance
func New(cfg *config.Config, spec *openapi.Spec, llmClient llm.Client) *Server {
	s := &Server{
		cfg:            cfg,
		spec:           spec,
		operations:     validation.GetSupportedOperations(spec),
		allPaths:       validation.AllPaths(spec),
		requestSchemas: make(map[*openapi.Operation]requestSchemaValidator),
		llmClient:      llmClient,
		logger:         logging.New(),
		mux:            http.NewServeMux(),
	}
	s.compileRequestSchemas()
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Swagger UI at /
	s.mux.HandleFunc("/", s.rootHandler)

	// Spec endpoints
	specEndpoint := s.spec.ServeEndpoint()
	s.mux.HandleFunc(specEndpoint, s.specHandler)

	// If the input is YAML, also serve at /openapi.yaml
	// If the input is JSON, also serve at /openapi.json
	// Both endpoints are registered for completeness
	if specEndpoint == "/openapi.json" {
		s.mux.HandleFunc("/openapi.yaml", s.notFoundHandler)
	} else {
		s.mux.HandleFunc("/openapi.json", s.notFoundHandler)
	}

	// Register API routes
	for _, op := range s.operations {
		op := op // capture
		pattern := convertOpenAPIPathToGoPattern(op.Method, op.Path)
		s.mux.HandleFunc(pattern, s.apiHandler(op))
	}
}

// convertOpenAPIPathToGoPattern converts OpenAPI path template to Go 1.22+ ServeMux pattern
func convertOpenAPIPathToGoPattern(method, path string) string {
	// Convert {param} to {param} (Go 1.22+ supports this natively)
	converted := path
	// Go 1.22 ServeMux uses "METHOD /path" pattern format
	return fmt.Sprintf("%s %s", method, converted)
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// Only serve Swagger UI at exactly "/"
	if r.URL.Path != "/" {
		s.handleDynamicRoute(w, r)
		return
	}
	swagger.Handler(s.spec.ServeEndpoint())(w, r)
}

func (s *Server) handleDynamicRoute(w http.ResponseWriter, r *http.Request) {
	// Check if this path exists in our OpenAPI spec
	path := r.URL.Path
	for _, op := range s.spec.Operations {
		if matchPath(op.Path, path) {
			// Path exists, but method might not be supported
			if op.Method != r.Method {
				if r.Method != "GET" && r.Method != "POST" {
					errutil.WriteError(w, http.StatusMethodNotAllowed, errutil.CodeMethodNotSupported,
						fmt.Sprintf("Method %s is not supported", r.Method), nil)
					return
				}
			}
		}
	}
	errutil.WriteError(w, http.StatusNotFound, errutil.CodeNotFound,
		fmt.Sprintf("No operation found for %s %s", r.Method, path), nil)
}

func (s *Server) specHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", s.spec.ServeContentType())
	w.Write(s.spec.RawContent)
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	errutil.WriteError(w, http.StatusNotFound, errutil.CodeNotFound,
		"Not found", nil)
}

func (s *Server) apiHandler(op *openapi.Operation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.requestID++
		requestID := fmt.Sprintf("req-%d-%d", time.Now().UnixMilli(), s.requestID)

		s.logger.Info("request received", map[string]interface{}{
			"requestId":   requestID,
			"method":      op.Method,
			"path":        op.Path,
			"operationId": op.OperationID,
		})

		// Check method
		if r.Method != op.Method {
			// Check if this path has the requested method
			methods, exists := s.allPaths[op.Path]
			if exists {
				found := false
				for _, m := range methods {
					if m == r.Method {
						found = true
						break
					}
				}
				if !found {
					errutil.WriteError(w, http.StatusMethodNotAllowed, errutil.CodeMethodNotSupported,
						fmt.Sprintf("Method %s not allowed for %s", r.Method, op.Path), nil)
					return
				}
			}
			return
		}

		// Enforce max request bytes
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxRequestBytes)

		var pathParams map[string]string
		var queryParams map[string]string
		var body interface{}

		// Extract path params
		pathParams = extractPathParams(op, r)

		// Extract and validate query params
		queryParams, err := extractQueryParams(op, r)
		if err != nil {
			errutil.WriteError(w, http.StatusBadRequest, errutil.CodeRequestValidationFailed,
				err.Error(), nil)
			return
		}

		// Handle POST body
		if op.Method == "POST" {
			ct := r.Header.Get("Content-Type")
			if !isJSONContentType(ct) {
				errutil.WriteError(w, http.StatusBadRequest, errutil.CodeContentTypeUnsupported,
					"Content-Type must be application/json", nil)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				if err.Error() == "http: request body too large" {
					errutil.WriteError(w, http.StatusBadRequest, errutil.CodeRequestTooLarge,
						"Request body exceeds maximum size", nil)
					return
				}
				errutil.WriteError(w, http.StatusBadRequest, errutil.CodeRequestValidationFailed,
					"Failed to read request body", nil)
				return
			}

			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				errutil.WriteError(w, http.StatusBadRequest, errutil.CodeRequestValidationFailed,
					"Invalid JSON in request body", nil)
				return
			}

			if err := s.validateRequestBody(op, body); err != nil {
				errutil.WriteError(w, http.StatusBadRequest, errutil.CodeRequestValidationFailed,
					err.Error(), nil)
				return
			}
		}

		// Build prompts
		systemPrompt, err := prompt.BuildSystemPrompt(s.cfg.SystemPrefix, op)
		if err != nil {
			s.logger.Error("failed to build system prompt", map[string]interface{}{
				"requestId": requestID,
				"error":     err.Error(),
			})
			errutil.WriteError(w, http.StatusInternalServerError, errutil.CodeInternalError,
				"Failed to build prompt", nil)
			return
		}

		userPrompt, err := prompt.BuildUserPrompt(op, pathParams, queryParams, body, s.cfg.PromptFormat)
		if err != nil {
			s.logger.Error("failed to build user prompt", map[string]interface{}{
				"requestId": requestID,
				"error":     err.Error(),
			})
			errutil.WriteError(w, http.StatusInternalServerError, errutil.CodeInternalError,
				"Failed to build prompt", nil)
			return
		}

		// Call LLM
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.TimeoutSeconds)*time.Second)
		defer cancel()

		result, err := s.llmClient.Generate(ctx, systemPrompt, userPrompt, op.RawSchema)
		if err != nil {
			s.logger.Error("LLM API error", map[string]interface{}{
				"requestId": requestID,
				"error":     err.Error(),
			})
			errutil.WriteError(w, http.StatusBadGateway, errutil.CodeLLMError,
				"Failed to generate response", nil)
			return
		}

		s.logger.Info("LLM response received", map[string]interface{}{
			"requestId":      requestID,
			"latencyMs":      result.Latency.Milliseconds(),
			"promptTokens":   result.PromptTokens,
			"responseTokens": result.OutputTokens,
			"totalTokens":    result.TotalTokens,
		})

		// Validate the response is valid JSON
		var responseJSON interface{}
		if err := json.Unmarshal([]byte(result.Content), &responseJSON); err != nil {
			s.logger.Error("response schema mismatch", map[string]interface{}{
				"requestId": requestID,
				"error":     "LLM response is not valid JSON",
			})
			errutil.WriteError(w, http.StatusBadGateway, errutil.CodeResponseSchemaMismatch,
				"Model response does not match expected schema", nil)
			return
		}

		s.logger.Info("response validated", map[string]interface{}{
			"requestId": requestID,
			"status":    "valid",
		})

		// Return the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result.Content))
	}
}

// extractPathParams extracts path parameters from the request
func extractPathParams(op *openapi.Operation, r *http.Request) map[string]string {
	params := make(map[string]string)
	for _, p := range op.Parameters {
		if p.In == "path" {
			val := r.PathValue(p.Name)
			if val != "" {
				params[p.Name] = val
			}
		}
	}
	return params
}

// extractQueryParams extracts and validates query parameters
func extractQueryParams(op *openapi.Operation, r *http.Request) (map[string]string, error) {
	params := make(map[string]string)

	// Build a set of allowed query params
	allowed := make(map[string]bool)
	for _, p := range op.Parameters {
		if p.In == "query" {
			allowed[p.Name] = true
		}
	}

	// Reject unknown query params
	for key := range r.URL.Query() {
		if !allowed[key] {
			return nil, fmt.Errorf("unknown query parameter: %s", key)
		}
	}

	// Extract defined params
	for _, p := range op.Parameters {
		if p.In == "query" {
			val := r.URL.Query().Get(p.Name)
			if val != "" {
				params[p.Name] = val
			} else if p.Required {
				return nil, fmt.Errorf("required query parameter %q is missing", p.Name)
			}
		}
	}

	return params, nil
}

// matchPath checks if a request path matches an OpenAPI path template
func matchPath(template, path string) bool {
	templateParts := strings.Split(template, "/")
	pathParts := strings.Split(path, "/")

	if len(templateParts) != len(pathParts) {
		return false
	}

	for i, tp := range templateParts {
		if strings.HasPrefix(tp, "{") && strings.HasSuffix(tp, "}") {
			continue // path parameter matches anything
		}
		if tp != pathParts[i] {
			return false
		}
	}
	return true
}

// isJSONContentType checks if the content type is application/json
func isJSONContentType(ct string) bool {
	ct = strings.TrimSpace(strings.ToLower(ct))
	return ct == "application/json" || strings.HasPrefix(ct, "application/json;")
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	s.logger.Info("starting server", map[string]interface{}{
		"addr":       s.cfg.ListenAddr,
		"operations": len(s.operations),
	})
	return http.ListenAndServe(s.cfg.ListenAddr, s.mux)
}
