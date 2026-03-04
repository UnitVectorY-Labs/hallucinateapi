package logging

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLogLevels(t *testing.T) {
	l := New()
	// These should not panic
	l.Info("test info", nil)
	l.Warn("test warn", nil)
	l.Error("test error", nil)
	l.Info("test with fields", map[string]interface{}{"key": "value"})
}
