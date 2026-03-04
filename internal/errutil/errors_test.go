package errutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, CodeRequestValidationFailed, "test error", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %q", w.Header().Get("Content-Type"))
	}

	var apiErr APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if apiErr.Error.Code != CodeRequestValidationFailed {
		t.Errorf("expected error code %q, got %q", CodeRequestValidationFailed, apiErr.Error.Code)
	}

	if apiErr.Error.Message != "test error" {
		t.Errorf("expected message 'test error', got %q", apiErr.Error.Message)
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]string{"field": "name"}
	WriteError(w, http.StatusBadRequest, CodeRequestValidationFailed, "validation error", details)

	var apiErr APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if apiErr.Error.Details == nil {
		t.Error("expected details in error response")
	}
}
