---
name: hallucinateapi
description: Host an OpenAPI-defined HTTP API with hallucinateapi when the API has a specification but no backend implementation. The tool uses the OpenAPI routes, inputs, response schemas, and operation descriptions to have an LLM hallucinate JSON responses for supported GET and POST requests through Gemini or OpenAI-compatible providers.
---

# hallucinateapi

`hallucinate` is the command line application for **HallucinateAPI**, an command line application that provides an OpenAPI-driven API server that uses an LLM to implement API operations.

Use this tool when a user wants to expose LLM-generated behavior through an HTTP API defined by an OpenAPI specification. The application reads the OpenAPI file, uses each supported operation's description as the LLM instruction, constrains the model output with the declared JSON response schema, and serves the result as a normal API endpoint.

This is a way to use the knowledge and reasoning capabilities inside an LLM to hallucinate an arbitrary API implementation, while using OpenAPI to define the API surface, inputs, and output shapes.

## Key model

The OpenAPI specification is the source of truth.

For each supported operation:

- The path and method define the served endpoint.
- Path parameters, query parameters, and JSON request bodies define the accepted input.
- The operation `description` tells the LLM what the endpoint is supposed to do.
- The HTTP `200` `application/json` response schema constrains the JSON the LLM must return (in `single-pass` mode).
- In `two-pass` mode, the LLM first selects a response type from all numeric status codes defined in the spec, then generates a body matching that response's schema.

The command line flags and environment variables choose how the server is run and which LLM provider is used.

## Providers

HallucinateAPI supports two provider modes:

| Provider | Use for |
|---|---|
| `gemini` | Gemini models on Vertex AI |
| `openai` | OpenAI and OpenAI-compatible Chat Completions APIs |

Always set the provider explicitly with `--provider` or `HALLUCINATE_PROVIDER`.

Do not assume OpenAI-only behavior. Do not assume Gemini-only behavior. The selected provider changes which provider-specific options are valid.

## Commands

Use the built-in help output for the authoritative command syntax:

```bash
hallucinate --help
hallucinate serve --help
hallucinate validate --help
```

### `serve`

Starts the HTTP server.

```bash
hallucinate serve [flags]
```

Running `hallucinate` with no subcommand defaults to `serve`:

```bash
hallucinate [flags]
```

### `validate`

Loads configuration and the OpenAPI file, then validates the setup without starting the server.

```bash
hallucinate validate [flags]
```

Use this before serving when checking whether a spec and provider configuration are valid.

### `--version`

Prints version and build information.

```bash
hallucinate --version
```

## OpenAPI requirements

The input file may be JSON or YAML.

Supported OpenAPI versions:

- OpenAPI 3.0.x
- OpenAPI 3.1.x where accepted by the parser

Implemented operation methods:

- `GET`
- `POST`

Other HTTP methods are not implemented as generated LLM-backed operations.

Each implemented `GET` or `POST` operation must define:

- An HTTP `200` response.
- `application/json` response content.
- A JSON schema for the response body.

In `two-pass` mode, operations may define multiple responses with numeric status codes (e.g., `200`, `404`, `500`). Each response must have `application/json` content with a JSON schema. Non-numeric codes (e.g., `default`) are ignored.

Each implemented `POST` operation must also define:

- A request body.
- `application/json` request content.
- A JSON schema for the request body.

The following paths are reserved and must not be defined in the OpenAPI spec:

```text
/
/openapi.json
/openapi.yaml
```

## How requests are handled

For `GET` operations:

- Input comes from path parameters and query parameters declared in the OpenAPI spec.
- Unknown query parameters are rejected.

For `POST` operations:

- The request must use `Content-Type: application/json`.
- The JSON body is validated against the OpenAPI request body schema.
- Unknown fields are rejected when the request schema uses `additionalProperties: false`.

For all implemented operations:

- The LLM is called only after request validation succeeds.
- The LLM response must be valid JSON.
- The response is generated according to the operation description and response schema.

## Built-in server endpoints

When the server is running:

| Path | Purpose |
|---|---|
| `/` | Swagger UI |
| `/openapi.json` | Serves the OpenAPI spec when the input is JSON |
| `/openapi.yaml` | Serves the OpenAPI spec when the input is YAML |

## Response generation mode

Set with `--mode` or `HALLUCINATE_MODE`. Controls how many LLM calls are made per request and which HTTP status codes can be returned.

| Mode | Description |
|---|---|
| `single-pass` (default) | One LLM call per request. Always returns HTTP 200. Uses the 200 response schema as the structured output constraint. |
| `two-pass` | Two LLM calls per request. The first call selects an HTTP status code from the responses defined in the OpenAPI spec. The second call generates the response body using the schema for the selected status code. Returns the selected status code. |

### When to use two-pass mode

Use `--mode two-pass` when your OpenAPI spec defines multiple response types (e.g., 200 and 404) and you want the LLM to choose which one to return. This is the only mode that can return non-200 responses.

In two-pass mode, the OpenAPI spec must define multiple responses with numeric status codes and JSON schemas. Non-numeric response codes (e.g., `default`) are ignored. The first LLM call receives the available numeric response options and selects one; the second call generates the body constrained by the schema of the selected response.

## Configuration precedence

Configuration may be supplied through CLI flags or environment variables.

CLI flags take precedence over environment variables.

If a `.env` file is present in the working directory, it is loaded automatically. Already-set environment variables take precedence over values in `.env`.

## Command line and environment variable cheat sheet

| Purpose | CLI flag | Environment variable(s) | Applies to |
|---|---|---|---|
| Provider | `--provider` | `HALLUCINATE_PROVIDER` | All providers |
| OpenAPI spec path | `--openapi-path` | `HALLUCINATE_OPENAPI_PATH` | All providers |
| Model name | `--model` | `HALLUCINATE_MODEL` | All providers |
| GCP project | `--gcp-project` | `GOOGLE_CLOUD_PROJECT`, `HALLUCINATE_GCP_PROJECT` | `gemini` |
| GCP location | `--gcp-location` | `HALLUCINATE_GCP_LOCATION` | `gemini` |
| API URL override | `--url` | `HALLUCINATE_URL` | All providers |
| API key | `--api-key` | `OPENAI_API_KEY`, `HALLUCINATE_API_KEY` | All providers |
| OpenAI strict schema mode | `--strict-schema` | `HALLUCINATE_STRICT_SCHEMA` | `openai` |
| Listen address | `--listen-addr` | `HALLUCINATE_LISTEN_ADDR` | Server |
| Base system prompt prefix | `--base-system-prefix` | `HALLUCINATE_SYSTEM_PREFIX` | Server |
| Prompt format | `--prompt-format` | `HALLUCINATE_PROMPT_FORMAT` | Server |
| Maximum request size | `--max-request-bytes` | `HALLUCINATE_MAX_REQUEST_BYTES` | Server |
| LLM timeout | `--timeout-seconds` | `HALLUCINATE_TIMEOUT_SECONDS` | Server |
| Schema profile override | `--schema-profile` | `HALLUCINATE_SCHEMA_PROFILE` | All providers |
| Skip TLS certificate verification | `--insecure` | `HALLUCINATE_INSECURE` | Outbound LLM calls |
| Response generation mode | `--mode` | `HALLUCINATE_MODE` | Server |

## Required settings

These are required for all provider modes:

| Setting | How to set |
|---|---|
| Provider | `--provider` or `HALLUCINATE_PROVIDER` |
| OpenAPI spec path | `--openapi-path` or `HALLUCINATE_OPENAPI_PATH` |
| Model name | `--model` or `HALLUCINATE_MODEL` |

Additional provider-specific requirements apply.

## Gemini provider notes

Use `--provider gemini`.

The Gemini provider targets Vertex AI Gemini models.

Unless `--url` is provided, Gemini requires:

- GCP project
- GCP location

Gemini authentication uses Google Application Default Credentials by default. Supplying `--api-key` or `HALLUCINATE_API_KEY` provides a bearer token instead.

Do not use `--strict-schema` with the Gemini provider.

## OpenAI provider notes

Use `--provider openai`.

The OpenAI provider targets OpenAI-compatible Chat Completions APIs.

Unless a custom `--url` is used for an endpoint that does not require authentication, provide an API key with `--api-key`, `OPENAI_API_KEY`, or `HALLUCINATE_API_KEY`.

The `--url` option can point to custom or local OpenAI-compatible Chat Completions endpoints.

Do not use `--gcp-project` or `--gcp-location` with the OpenAI provider.

`--strict-schema` is only valid with the OpenAI provider.

## URL behavior

| Condition | URL behavior |
|---|---|
| `--url` is provided | Uses the provided URL directly |
| `--provider gemini` and no `--url` | Builds a Vertex AI Gemini endpoint from GCP project, location, and model |
| `--provider openai` and no `--url` | Uses the default OpenAI Chat Completions endpoint |

## Prompt format

The request payload sent to the LLM can be serialized as:

| Value | Meaning |
|---|---|
| `json` | JSON prompt payload |
| `toon` | TOON text notation prompt payload |

Set this with `--prompt-format` or `HALLUCINATE_PROMPT_FORMAT`.

## Schema profile

HallucinateAPI validates response schemas against a structured-output compatibility profile before serving.

Default schema profile by provider:

| Provider | Default |
|---|---|
| `gemini` | `GEMINI_202602` |
| `openai` | `OPENAI_202602` |

Supported profile values include:

```text
OPENAI_202602
GEMINI_202602
GEMINI_202503
MINIMAL_202602
```

Override the profile with `--schema-profile` or `HALLUCINATE_SCHEMA_PROFILE`.

Response schemas should be inline because the extracted response schema is sent to the provider as the structured output constraint.

## Docker

The containerized application runs the same CLI. Mount the OpenAPI spec into the container and provide configuration through environment variables or command arguments.

The application listens on the configured listen address. The Dockerfile exposes port `8080`.

## Agent guidance

When using this tool:

- Treat the OpenAPI file as the API contract and behavior source.
- Use `validate` to check configuration and the spec before relying on `serve`.
- Choose `gemini` or `openai` deliberately.
- Do not invent provider-specific flags. Use the help command for exact syntax.
- Do not assume endpoint behavior outside what the OpenAPI operation descriptions and schemas define.
- For better generated API behavior, ensure the OpenAPI operation descriptions explain what the LLM-backed endpoint should return.
