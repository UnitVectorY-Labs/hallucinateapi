---
layout: default
title: hallucinateapi
nav_order: 1
permalink: /
---

# hallucinateapi

HallucinateAPI is an API gateway that implements every GET and POST operation defined in an OpenAPI specification using a Gemini language model on Vertex AI. Instead of writing business logic, each operation's description serves as the instruction for the LLM, and the response schema constrains the model output to valid, structured JSON.

## Key Features

- **OpenAPI-driven**: Define your API with an OpenAPI 3.0 spec, and HallucinateAPI implements it automatically.
- **Gemini-powered**: Uses Google's Gemini models via Vertex AI for response generation with structured output constraints.
- **Schema-validated**: Both requests and responses are validated against the OpenAPI schema for type safety.
- **Swagger UI included**: Interactive API documentation served at the root path.
- **Simple deployment**: Single binary with configuration via environment variables or CLI flags. Container-ready with Docker support.
- **Secure by design**: Strict input allowlisting, request size limits, and prompt injection resistance.

## How It Works

1. Load an OpenAPI specification defining your API endpoints.
2. HallucinateAPI validates the spec and registers routes for all GET and POST operations.
3. When a request arrives, inputs are validated and formatted into a prompt.
4. The prompt is sent to Gemini with the response schema as a structured output constraint.
5. The model response is validated and returned as the API response.

## Quick Start

```bash
hallucinate serve \
  --openapi-path /path/to/your/spec.yaml \
  --gcp-project your-gcp-project \
  --gcp-location us-central1 \
  --model gemini-2.0-flash
```

Visit `http://localhost:8080` for the Swagger UI.