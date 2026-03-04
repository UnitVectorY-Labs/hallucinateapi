package swagger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	handler := Handler("/openapi.yaml")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("expected swagger-ui in response body")
	}

	if !strings.Contains(body, "/openapi.yaml") {
		t.Error("expected spec endpoint URL in response body")
	}
}

func TestHandlerJSONSpec(t *testing.T) {
	handler := Handler("/openapi.json")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "/openapi.json") {
		t.Error("expected JSON spec endpoint URL in response body")
	}
}
