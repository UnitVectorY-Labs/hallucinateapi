package main

import (
	"testing"
)

func TestVersionDefault(t *testing.T) {
	// Version should have a default value
	if Version == "" {
		t.Error("expected non-empty default version")
	}
}
