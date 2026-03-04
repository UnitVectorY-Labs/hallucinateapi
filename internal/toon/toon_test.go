package toon

import (
	"strings"
	"testing"
)

func TestSerializeSimpleObject(t *testing.T) {
	input := map[string]interface{}{
		"name":  "Alice",
		"age":   float64(30),
		"email": "alice@example.com",
	}

	result := Serialize(input)
	if result == "" {
		t.Error("expected non-empty output")
	}

	// Check deterministic ordering (alphabetical)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), result)
	}
	if !strings.HasPrefix(lines[0], "age:") {
		t.Errorf("expected first line to start with 'age:', got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "email:") {
		t.Errorf("expected second line to start with 'email:', got %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "name:") {
		t.Errorf("expected third line to start with 'name:', got %q", lines[2])
	}
}

func TestSerializeNestedObject(t *testing.T) {
	input := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Bob",
		},
	}

	result := Serialize(input)
	if !strings.Contains(result, "user:") {
		t.Error("expected 'user:' in output")
	}
	if !strings.Contains(result, "  name: Bob") {
		t.Errorf("expected indented name, got %q", result)
	}
}

func TestSerializeArray(t *testing.T) {
	input := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}

	result := Serialize(input)
	if !strings.Contains(result, "[0]: a") {
		t.Errorf("expected array item [0], got %q", result)
	}
}

func TestSerializeNull(t *testing.T) {
	input := map[string]interface{}{
		"value": nil,
	}

	result := Serialize(input)
	if !strings.Contains(result, "null") {
		t.Errorf("expected 'null' in output, got %q", result)
	}
}

func TestSerializeBool(t *testing.T) {
	input := map[string]interface{}{
		"active": true,
		"banned": false,
	}

	result := Serialize(input)
	if !strings.Contains(result, "true") {
		t.Errorf("expected 'true' in output, got %q", result)
	}
	if !strings.Contains(result, "false") {
		t.Errorf("expected 'false' in output, got %q", result)
	}
}

func TestSerializeEmpty(t *testing.T) {
	input := map[string]interface{}{}
	result := Serialize(input)
	if result != "" {
		t.Errorf("expected empty output for empty map, got %q", result)
	}
}
