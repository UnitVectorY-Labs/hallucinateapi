package errutil

import (
	"encoding/json"
	"net/http"
)

// Error codes as stable constants
const (
	CodeOpenAPIInvalid           = "OPENAPI_INVALID"
	CodeRouteConflict            = "ROUTE_CONFLICT"
	CodeMethodNotSupported       = "METHOD_NOT_SUPPORTED"
	CodeRequestValidationFailed  = "REQUEST_VALIDATION_FAILED"
	CodeContentTypeUnsupported   = "CONTENT_TYPE_UNSUPPORTED"
	CodeGeminiError              = "GEMINI_ERROR"
	CodeResponseSchemaMismatch   = "RESPONSE_SCHEMA_MISMATCH"
	CodeInternalError            = "INTERNAL_ERROR"
	CodeNotFound                 = "NOT_FOUND"
	CodeRequestTooLarge          = "REQUEST_TOO_LARGE"
)

// APIError represents a structured error response
type APIError struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// WriteError writes a JSON error response
func WriteError(w http.ResponseWriter, statusCode int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := APIError{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	json.NewEncoder(w).Encode(resp)
}
