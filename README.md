# hallucinateapi

Implements every GET and POST operation in an OpenAPI spec by using each operation's description as the LLM instruction, validating inputs, and returning schema-constrained JSON that behaves like a normal API response.

## Features

- Serves all GET and POST operations from an OpenAPI 3.0 specification
- Uses Gemini on Vertex AI for response generation with structured JSON output
- Validates requests and responses against the OpenAPI schema
- Swagger UI at `/` and spec served at `/openapi.json` or `/openapi.yaml`
- Configuration via environment variables and CLI flags
- Built-in spec validation with `validate` subcommand

## Quick Start

```bash
# Validate your OpenAPI spec
hallucinate validate --openapi-path spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.0-flash

# Start the server
hallucinate serve --openapi-path spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.0-flash
```

## Documentation

See the [documentation site](https://hallucinateapi.unitvectory.com/) for full details including:

- [Usage guide](https://hallucinateapi.unitvectory.com/usage) - Configuration, commands, and deployment
- [OpenAPI requirements](https://hallucinateapi.unitvectory.com/openapi) - Spec format, schema constraints, and examples
