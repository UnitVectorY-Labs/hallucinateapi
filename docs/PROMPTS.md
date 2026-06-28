---
layout: default
title: Prompts
nav_order: 3
permalink: /prompts
---

# Prompts

This document describes how prompts are constructed for each LLM call.

## Prompt Files

All prompt templates are stored as `.txt` files under `prompts/` and embedded into the binary at compile time via Go's `//go:embed` directive.

| File | Purpose |
|------|---------|
| `system.txt` | Base system instructions for all LLM calls |
| `selection_instruction.txt` | Instruction for two-pass selection phase |
| `selection_context_prefix.txt` | Prefix for two-pass generation phase user prompt |

## System Prompt Construction

Every LLM call receives a system prompt built by `prompt.BuildSystemPrompt()`. It is assembled from three parts:

1. **Custom prefix** (optional): If `--base-system-prefix` is set, it is prepended with a double newline.
2. **System prompt template**: The content of `prompts/system.txt`.
3. **Operation context**: A JSON object containing the operation's `method`, `path`, `operationId`, `summary`, `description`, `parameters`, `requestBody` (if applicable), and `responseSchema`.

The operation context is produced by `openapi.BuildOperationContext()` and appended directly after the `"OPERATION DETAILS:"` line in `system.txt`.

## User Prompt Construction

The user prompt is built by `prompt.BuildUserPrompt()` and contains the request data:

- `pathParameters`: path parameters from the URL (only those defined in the spec)
- `queryParameters`: query parameters (only those defined in the spec)
- `body`: request body for POST operations

The format is controlled by `--prompt-format`:

- **`json`** (default): minified JSON object
- **`toon`**: indented key-value notation with alphabetically sorted keys

## Response Schema

The response schema is never embedded as text in the prompt. It is always sent as a **structured output constraint** via the LLM provider's API:

- **OpenAI provider**: sent in `response_format.json_schema.schema`
- **Gemini provider**: sent in `generationConfig.responseJsonSchema`

---

## Single-Pass Mode (Default)

One LLM call per request. The server always returns HTTP 200.

| Component | Content |
|-----------|---------|
| System prompt | Custom prefix + `system.txt` + operation context (200 response schema) |
| User prompt | Request data (JSON or TOON format) |
| Response schema | HTTP 200 response schema from the OpenAPI spec |

### Example

**System prompt** (assembled):
```
You are a precise API implementation engine. Your sole purpose is to implement the specific API operation described below.

STRICT RULES:
1. You MUST return ONLY valid JSON matching the response schema exactly.
...

OPERATION DETAILS:
{
  "method": "GET",
  "path": "/users/{userId}",
  "operationId": "getUser",
  "description": "Returns user details...",
  "parameters": [...],
  "responseSchema": { ... }
}
```

**User prompt** (JSON format):
```json
{"pathParameters":{"userId":"123"}}
```

**Response schema** (structured output constraint): the 200 response schema from the OpenAPI spec.

---

## Two-Pass Mode

Two LLM calls per request. The first call selects the HTTP response type, the second generates the response body using the selected schema. The server returns the selected status code.

### Pass 1: Selection

The model chooses which HTTP response type to return based on the request context.

| Component | Content |
|-----------|---------|
| System prompt | Same as single-pass (custom prefix + `system.txt` + operation context with 200 schema) |
| User prompt | JSON object containing the selection instruction, the original request data, and available response options |
| Response schema | Hardcoded selection schema: `{statusCode: enum["200", "404", ...]}` |

The selection prompt payload is constructed as:
```json
{
  "instruction": "<content of prompts/selection_instruction.txt>",
  "request": "<original user prompt>",
  "responses": [
    {"statusCode": "200", "description": "User found"},
    {"statusCode": "404", "description": "User not found"}
  ]
}
```

The response schema constrains the model to return `{"statusCode": "<code>"}` where `<code>` is one of the numeric HTTP status codes defined in the OpenAPI spec. Non-numeric codes (e.g., `default`) are excluded.

### Pass 2: Generation

The model generates the response body using the schema for the selected response type.

| Component | Content |
|-----------|---------|
| System prompt | Rebuilt with the **selected** response schema (e.g., the 404 schema instead of 200) |
| User prompt | Original user prompt + selection context suffix |
| Response schema | Schema for the selected response type |

The selection context suffix is:
```
<content of prompts/selection_context_prefix.txt>{"selectedResponseType":{"statusCode":"404","description":"User not found"}}
```

The system prompt is rebuilt so that the `responseSchema` field in the operation context reflects the selected response type rather than the default 200 schema.

### Example Two-Pass Flow

**Pass 1 — Selection:**

- System prompt: standard (200 schema in context)
- User prompt: `{"instruction":"Choose the most appropriate...","request":"{\"pathParameters\":{\"userId\":\"does-not-exist\"}}","responses":[{"statusCode":"200","description":"User found"},{"statusCode":"404","description":"User not found"}]}`
- Response schema: `{"type":"object","required":["statusCode"],"properties":{"statusCode":{"type":"string","enum":["200","404"]}}}`
- Model returns: `{"statusCode":"404"}`

**Pass 2 — Generation:**

- System prompt: rebuilt with 404 schema in context
- User prompt: `{"pathParameters":{"userId":"does-not-exist"}}\n\nResponse selection context: {"selectedResponseType":{"statusCode":"404","description":"User not found"}}`
- Response schema: 404 response schema from the OpenAPI spec
- Model returns: `{"error":"User not found."}`
- Server returns: HTTP 404
