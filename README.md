# hallucinateapi

Implements every GET and POST operation in an OpenAPI spec by using each operation's description as the LLM instruction, validating inputs, and returning schema-constrained JSON that behaves like a normal API response.

## Features

- Serves all GET and POST operations from an OpenAPI 3.0 specification
- Supports multiple LLM providers: Gemini on Vertex AI and OpenAI-compatible APIs
- Validates requests and responses against the OpenAPI schema
- Swagger UI at `/` and spec served at `/openapi.json` or `/openapi.yaml`
- Configuration via environment variables and CLI flags
- Built-in spec validation with `validate` subcommand

## Quick Start

```bash
# Validate your OpenAPI spec (Gemini)
hallucinate validate --provider gemini --openapi-path spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash

# Start the server (Gemini)
hallucinate serve --provider gemini --openapi-path spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash

# Start the server (OpenAI)
hallucinate serve --provider openai --openapi-path spec.yaml --model gpt-4o --api-key your-api-key
```
