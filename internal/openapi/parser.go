package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

// Operation represents a parsed OpenAPI operation ready for routing
type Operation struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	OperationID string                 `json:"operationId,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBodySchema     `json:"requestBody,omitempty"`
	Response    *ResponseSchema        `json:"response,omitempty"`
	RawSchema   map[string]interface{} `json:"rawResponseSchema,omitempty"`
}

// Parameter represents an OpenAPI parameter
type Parameter struct {
	Name     string                 `json:"name"`
	In       string                 `json:"in"`
	Required bool                   `json:"required"`
	Schema   map[string]interface{} `json:"schema,omitempty"`
}

// RequestBodySchema represents a POST request body
type RequestBodySchema struct {
	Required bool                   `json:"required"`
	Schema   map[string]interface{} `json:"schema,omitempty"`
}

// ResponseSchema represents the 200 response schema
type ResponseSchema struct {
	Schema map[string]interface{} `json:"schema"`
}

// Spec holds the parsed OpenAPI specification
type Spec struct {
	Operations  []*Operation
	RawContent  []byte
	ContentType string // "json" or "yaml"
	Version     string
}

// LoadSpec loads and parses an OpenAPI spec from a file
func LoadSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var contentType string
	switch ext {
	case ".json":
		contentType = "json"
	case ".yaml", ".yml":
		contentType = "yaml"
	default:
		// Try to detect
		if json.Valid(data) {
			contentType = "json"
		} else {
			contentType = "yaml"
		}
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	model, err := doc.BuildV3Model()
	if model == nil {
		return nil, fmt.Errorf("failed to build OpenAPI v3 model: %v", err)
	}
	if err != nil {
		// Log warning but continue - some errors may be non-critical
		errStr := err.Error()
		if strings.Contains(errStr, "cannot") || strings.Contains(errStr, "invalid") {
			return nil, fmt.Errorf("OpenAPI model error: %v", err)
		}
	}

	version := doc.GetVersion()

	// Extract operations
	ops, err := extractOperations(model)
	if err != nil {
		return nil, err
	}

	return &Spec{
		Operations:  ops,
		RawContent:  data,
		ContentType: contentType,
		Version:     version,
	}, nil
}

func extractOperations(model *libopenapi.DocumentModel[v3.Document]) ([]*Operation, error) {
	var ops []*Operation

	if model.Model.Paths == nil {
		return ops, nil
	}

	ctx := context.Background()

	for pair := range orderedmap.Iterate[string, *v3.PathItem](ctx, model.Model.Paths.PathItems) {
		pathStr := pair.Key()
		pathItem := pair.Value()

		methods := map[string]*v3.Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"PATCH":   pathItem.Patch,
			"DELETE":  pathItem.Delete,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
			"TRACE":   pathItem.Trace,
		}

		for method, opRef := range methods {
			if opRef == nil {
				continue
			}

			op := &Operation{
				Method: method,
				Path:   pathStr,
			}

			if opRef.OperationId != "" {
				op.OperationID = opRef.OperationId
			}
			if opRef.Summary != "" {
				op.Summary = opRef.Summary
			}
			if opRef.Description != "" {
				op.Description = opRef.Description
			}

			// Extract parameters
			if opRef.Parameters != nil {
				for _, p := range opRef.Parameters {
					param := Parameter{
						Name:     p.Name,
						In:       p.In,
						Required: falseIfNil(p.Required),
					}
					if p.Schema != nil {
						schema := p.Schema.Schema()
						if schema != nil {
							param.Schema = schemaToMap(schema)
						}
					}
					op.Parameters = append(op.Parameters, param)
				}
			}
			// Also include path-level parameters
			if pathItem.Parameters != nil {
				for _, p := range pathItem.Parameters {
					param := Parameter{
						Name:     p.Name,
						In:       p.In,
						Required: falseIfNil(p.Required),
					}
					if p.Schema != nil {
						schema := p.Schema.Schema()
						if schema != nil {
							param.Schema = schemaToMap(schema)
						}
					}
					op.Parameters = append(op.Parameters, param)
				}
			}

			// Extract request body for POST
			if method == "POST" && opRef.RequestBody != nil {
				rb := opRef.RequestBody
				if rb.Content != nil {
					for ctPair := range orderedmap.Iterate[string, *v3.MediaType](ctx, rb.Content) {
						ct := ctPair.Key()
						mt := ctPair.Value()
						if ct == "application/json" && mt.Schema != nil {
							schema := mt.Schema.Schema()
							if schema != nil {
								op.RequestBody = &RequestBodySchema{
									Required: falseIfNil(rb.Required),
									Schema:   schemaToMap(schema),
								}
							}
						}
					}
				}
			}

			// Extract 200 response schema
			if opRef.Responses != nil && opRef.Responses.Codes != nil {
				for respPair := range orderedmap.Iterate[string, *v3.Response](ctx, opRef.Responses.Codes) {
					code := respPair.Key()
					resp := respPair.Value()
					if code == "200" && resp.Content != nil {
						for ctPair := range orderedmap.Iterate[string, *v3.MediaType](ctx, resp.Content) {
							ct := ctPair.Key()
							mt := ctPair.Value()
							if ct == "application/json" && mt.Schema != nil {
								schema := mt.Schema.Schema()
								if schema != nil {
									schemaMap := schemaToMap(schema)
									op.Response = &ResponseSchema{
										Schema: schemaMap,
									}
									op.RawSchema = schemaMap
								}
							}
						}
					}
				}
			}

			ops = append(ops, op)
		}
	}

	// Sort for deterministic ordering
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})

	return ops, nil
}

func falseIfNil(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// schemaToMap converts a libopenapi schema proxy to a map for JSON Schema
func schemaToMap(schema interface{}) map[string]interface{} {
	// Marshal to JSON and back to get a clean map
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	// Clean up: remove empty/nil fields and libopenapi internal fields
	return cleanSchemaMap(m)
}

// cleanSchemaMap removes empty and internal fields from the schema map
func cleanSchemaMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		// Skip libopenapi internal fields and OpenAPI extensions
		if strings.HasPrefix(k, "x-") || k == "GoLow" || k == "ParentProxy" ||
			k == "IsCircular" {
			continue
		}
		switch val := v.(type) {
		case nil:
			// skip nil values
		case map[string]interface{}:
			if len(val) > 0 {
				cleaned := cleanSchemaMap(val)
				if len(cleaned) > 0 {
					result[k] = cleaned
				}
			}
		case []interface{}:
			if len(val) > 0 {
				cleaned := cleanSchemaSlice(val)
				if len(cleaned) > 0 {
					result[k] = cleaned
				}
			}
		case string:
			if val != "" {
				result[k] = val
			}
		case bool:
			result[k] = val
		case float64:
			result[k] = val
		default:
			result[k] = v
		}
	}
	return result
}

func cleanSchemaSlice(s []interface{}) []interface{} {
	var result []interface{}
	for _, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			cleaned := cleanSchemaMap(val)
			if len(cleaned) > 0 {
				result = append(result, cleaned)
			}
		default:
			result = append(result, v)
		}
	}
	return result
}

// BuildOperationContext builds the operation context for system prompt
func BuildOperationContext(op *Operation) (string, error) {
	ctx := map[string]interface{}{
		"method": op.Method,
		"path":   op.Path,
	}
	if op.OperationID != "" {
		ctx["operationId"] = op.OperationID
	}
	if op.Summary != "" {
		ctx["summary"] = op.Summary
	}
	if op.Description != "" {
		ctx["description"] = op.Description
	}
	if len(op.Parameters) > 0 {
		params := make([]map[string]interface{}, len(op.Parameters))
		for i, p := range op.Parameters {
			pm := map[string]interface{}{
				"name":     p.Name,
				"in":       p.In,
				"required": p.Required,
			}
			if p.Schema != nil {
				pm["schema"] = p.Schema
			}
			params[i] = pm
		}
		ctx["parameters"] = params
	}
	if op.RequestBody != nil {
		ctx["requestBody"] = op.RequestBody
	}
	if op.Response != nil {
		ctx["responseSchema"] = op.Response.Schema
	}
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal operation context: %w", err)
	}
	return string(data), nil
}

// ServeContentType returns the content type header for the served spec
func (s *Spec) ServeContentType() string {
	if s.ContentType == "json" {
		return "application/json"
	}
	return "application/x-yaml"
}

// ServeEndpoint returns the endpoint path for serving the spec
func (s *Spec) ServeEndpoint() string {
	if s.ContentType == "json" {
		return "/openapi.json"
	}
	return "/openapi.yaml"
}

// ConvertToJSON converts the spec to JSON if not already
func (s *Spec) ConvertToJSON() ([]byte, error) {
	if s.ContentType == "json" {
		return s.RawContent, nil
	}
	// Parse YAML and convert to JSON
	var obj interface{}
	if err := yaml.Unmarshal(s.RawContent, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	obj = convertYAMLToJSON(obj)
	return json.Marshal(obj)
}

// convertYAMLToJSON recursively converts YAML-parsed types to JSON-compatible types
func convertYAMLToJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[k] = convertYAMLToJSON(v)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = convertYAMLToJSON(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = convertYAMLToJSON(v)
		}
		return result
	default:
		return v
	}
}
