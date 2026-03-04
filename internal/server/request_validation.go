package server

import (
	"fmt"
	"strings"

	"github.com/UnitVectorY-Labs/hallucinateapi/internal/openapi"
	"github.com/xeipuuv/gojsonschema"
)

type requestSchemaValidator struct {
	schema *gojsonschema.Schema
	err    error
}

func (s *Server) compileRequestSchemas() {
	for _, op := range s.operations {
		if op.Method != "POST" || op.RequestBody == nil || op.RequestBody.Schema == nil {
			continue
		}

		schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(op.RequestBody.Schema))
		s.requestSchemas[op] = requestSchemaValidator{
			schema: schema,
			err:    err,
		}
	}
}

func (s *Server) validateRequestBody(op *openapi.Operation, body interface{}) error {
	if op.RequestBody == nil || op.RequestBody.Schema == nil {
		return nil
	}

	validator, ok := s.requestSchemas[op]
	if !ok {
		return nil
	}
	if validator.err != nil {
		return fmt.Errorf("request schema could not be compiled: %w", validator.err)
	}

	result, err := validator.schema.Validate(gojsonschema.NewGoLoader(body))
	if err != nil {
		return fmt.Errorf("request body validation failed: %w", err)
	}
	if result.Valid() {
		return nil
	}

	return fmt.Errorf("request body does not match schema: %s", formatRequestSchemaErrors(result.Errors()))
}

func formatRequestSchemaErrors(errors []gojsonschema.ResultError) string {
	messages := make([]string, 0, len(errors))
	for _, err := range errors {
		messages = append(messages, err.String())
	}
	return strings.Join(messages, "; ")
}
