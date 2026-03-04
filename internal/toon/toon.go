package toon

import (
	"fmt"
	"sort"
	"strings"
)

// Serialize converts a Go value to TOON text notation format.
// TOON (Text Object Oriented Notation) uses indentation-based key-value pairs.
// This is a deterministic serialization: keys are sorted alphabetically.
func Serialize(v interface{}) string {
	var b strings.Builder
	writeTOON(&b, v, 0)
	return b.String()
}

func writeTOON(b *strings.Builder, v interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := val[k]
			switch cv := child.(type) {
			case map[string]interface{}:
				fmt.Fprintf(b, "%s%s:\n", prefix, k)
				writeTOON(b, cv, indent+1)
			case []interface{}:
				fmt.Fprintf(b, "%s%s:\n", prefix, k)
				writeTOON(b, cv, indent+1)
			default:
				fmt.Fprintf(b, "%s%s: %v\n", prefix, k, formatValue(child))
			}
		}
	case []interface{}:
		for i, item := range val {
			switch cv := item.(type) {
			case map[string]interface{}:
				fmt.Fprintf(b, "%s[%d]:\n", prefix, i)
				writeTOON(b, cv, indent+1)
			case []interface{}:
				fmt.Fprintf(b, "%s[%d]:\n", prefix, i)
				writeTOON(b, cv, indent+1)
			default:
				fmt.Fprintf(b, "%s[%d]: %v\n", prefix, i, formatValue(item))
			}
		}
	default:
		fmt.Fprintf(b, "%s%v\n", prefix, formatValue(v))
	}
}

func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
