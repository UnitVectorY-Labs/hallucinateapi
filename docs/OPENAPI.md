---
layout: default
title: OpenAPI
nav_order: 3
permalink: /openapi
---

# OpenAPI Specification Requirements
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

---

## Supported Versions

HallucinateAPI supports **OpenAPI 3.0.x** specifications. OpenAPI 3.1.x is also accepted where the parsing library supports it.

The specification file can be provided in either **JSON** or **YAML** format.

## Supported Operations

Only **GET** and **POST** operations are implemented. Any other HTTP methods (PUT, PATCH, DELETE, etc.) defined in the spec will return `405 Method Not Allowed` at runtime.

### GET Operations

- Input is derived from **path parameters** and **query parameters** defined in the OpenAPI spec.
- Unknown query parameters are rejected with a `400` error.
- Parameters must include schema definitions (type, format, enum, etc.) for proper validation.

### POST Operations

- Must define a `requestBody` with `application/json` content type and a schema.
- POST operations without an `application/json` request body schema will fail validation.
- Request body must be valid JSON and the `Content-Type` header must be `application/json` (or `application/json; charset=utf-8`).
- Request bodies are validated against the declared schema before the LLM is called.
- Object schemas that set `additionalProperties: false` reject unknown fields with `400 REQUEST_VALIDATION_FAILED`.

## Response Schema Requirements

Every implemented operation (GET or POST) must define:

- An **HTTP 200** response
- With **`application/json`** content type
- Including a **JSON Schema** for the response body

### Gemini Structured Output Compatibility

The response schema is sent to Gemini as the structured output constraint. HallucinateAPI follows Gemini's documented JSON Schema subset and applies a small number of additional implementation-specific restrictions.

#### Gemini-Documented Support

Gemini's structured output support includes object schemas with:

- `properties`
- `required`
- `additionalProperties`
- `items`
- `prefixItems`
- `anyOf`
- `oneOf` (Gemini treats this the same as `anyOf`)
- Numeric constraints such as `minimum` and `maximum`
- Common metadata such as `type`, `format`, `title`, `description`, and `enum`

Gemini ignores unsupported schema properties rather than enforcing them. Because this server does **not** perform full response-schema validation after generation, you should keep important response constraints within Gemini's documented subset.

#### HallucinateAPI Restrictions

HallucinateAPI currently rejects the following response-schema patterns:

- `$ref`
  - Gemini supports JSON Schema references, but HallucinateAPI sends only the extracted response schema fragment to Gemini, not the full OpenAPI document with `components`. Response schemas therefore need to be fully inline.
- `oneOf`
  - Gemini treats `oneOf` the same as `anyOf`. HallucinateAPI rejects it because the server does not preserve true `oneOf` semantics after generation.
- `allOf`
- `not`

#### Supported Keywords

The following keywords are accepted in response schemas by HallucinateAPI:

- `type`, `properties`, `additionalProperties`, `items`, `required`
- `enum`, `format`, `description`, `title`, `propertyOrdering`
- `nullable`, `example`, `default`
- `minimum`, `maximum`, `minItems`, `maxItems`
- `minLength`, `maxLength`, `pattern`, `prefixItems`
- `anyOf`

#### Schema Depth Limit

Response schemas are limited to a maximum nesting depth of **10 levels**. Deeply nested schemas may cause issues with Gemini's structured output generation.

## Reserved Paths

The following paths are reserved for built-in server functionality and **must not** be defined in your OpenAPI specification:

| Path | Purpose |
|------|---------|
| `/` | Swagger UI |
| `/openapi.json` | OpenAPI spec (JSON format) |
| `/openapi.yaml` | OpenAPI spec (YAML format) |

Defining operations at any of these paths will cause a validation failure (`ROUTE_CONFLICT`).

## Example OpenAPI Spec

The following is a minimal valid OpenAPI specification that passes all validations:

```yaml
openapi: "3.0.3"
info:
  title: Example API
  version: "1.0.0"
  description: A simple example API
paths:
  /api/greeting:
    get:
      operationId: getGreeting
      summary: Get a greeting
      description: Returns a personalized greeting message based on the provided name.
      parameters:
        - name: name
          in: query
          required: false
          schema:
            type: string
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
                    description: The greeting message
  /api/echo:
    post:
      operationId: echoMessage
      summary: Echo a message
      description: Accepts a message and returns it with metadata.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                text:
                  type: string
                  description: The text to echo
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              schema:
                type: object
                properties:
                  original:
                    type: string
                  timestamp:
                    type: string
```

## Validation

Use the `validate` command to check your OpenAPI spec before starting the server:

```bash
hallucinate validate --openapi-path /path/to/spec.yaml \
  --gcp-project my-project \
  --gcp-location us-central1 \
  --model gemini-2.0-flash
```

This will output validation results in both JSON and human-readable text format.
