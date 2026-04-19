---
layout: default
title: Usage
nav_order: 2
permalink: /usage
---

# Usage
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

---

## Overview

HallucinateAPI is an HTTP server that implements all GET and POST operations defined in an OpenAPI specification. Each operation is served by calling a language model using the operation's description as the LLM instruction and the response schema for structured JSON output. HallucinateAPI supports multiple LLM providers including Gemini on Vertex AI and any OpenAI-compatible Chat Completions API.

## Commands

### `serve` (default)

Starts the HTTP server. Runs all validations on startup and exits non-zero if validation fails.

```bash
# Gemini
hallucinate serve --provider gemini --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash

# OpenAI
hallucinate serve --provider openai --openapi-path /path/to/spec.yaml --model gpt-4o --api-key your-api-key
```

Running with no subcommand defaults to `serve`:

```bash
hallucinate --provider gemini --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash
```

### `validate`

Loads configuration and OpenAPI file, runs all validations, and outputs results in both JSON and human-readable text. Exits `0` if valid, non-zero if invalid.

```bash
hallucinate validate --provider gemini --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash
```

## Providers

The `--provider` flag is required and determines which LLM API format to use:

| Provider | Description |
|----------|-------------|
| `gemini` | Vertex AI Gemini models |
| `openai` | OpenAI-compatible Chat Completions API |

## Configuration

All settings can be set via environment variables or CLI flags. **CLI flags take precedence** over environment variables.

If a `.env` file is present in the working directory when the application starts, it is loaded automatically. Variables already set in the environment take precedence over values in the `.env` file.

### Required Settings

| Flag | Environment Variable(s) | Description |
|------|------------------------|-------------|
| `--provider` | `HALLUCINATE_PROVIDER` | LLM provider: `gemini` or `openai` (required) |
| `--openapi-path` | `HALLUCINATE_OPENAPI_PATH` | Path to the OpenAPI specification file (JSON or YAML) |
| `--model` | `HALLUCINATE_MODEL` | Model name (e.g., `gemini-2.5-flash`, `gpt-4o`) |

### Gemini-only Settings

These options only apply when `--provider=gemini`:

| Flag | Environment Variable(s) | Required | Description |
|------|------------------------|----------|-------------|
| `--gcp-project` | `GOOGLE_CLOUD_PROJECT`, `HALLUCINATE_GCP_PROJECT` | yes* | Google Cloud Platform project ID (unless `--url` is provided) |
| `--gcp-location` | `HALLUCINATE_GCP_LOCATION` | yes* | Vertex AI location (e.g., `us-central1` or `global`) (unless `--url` is provided) |

When `--gcp-location=global`, requests are sent to `https://aiplatform.googleapis.com`. Regional locations use `https://<location>-aiplatform.googleapis.com`.

When `--provider=openai`, using `--gcp-project` or `--gcp-location` will result in an error.

### OpenAI-only Settings

These options only apply when `--provider=openai`:

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--strict-schema` | `HALLUCINATE_STRICT_SCHEMA` | Enable strict mode for JSON schema validation |

When `--provider=gemini`, using `--strict-schema` will result in an error.

The `--strict-schema` flag enables OpenAI's [strict mode](https://platform.openai.com/docs/guides/structured-outputs?api-mode=chat) for structured outputs, which will return an error if the JSON schema contains unsupported constructs.

### Server Settings

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--listen-addr` | `HALLUCINATE_LISTEN_ADDR` | `:8080` | Address and port to listen on |
| `--base-system-prefix` | `HALLUCINATE_SYSTEM_PREFIX` | *(empty)* | Custom prefix added to the system prompt for all operations |
| `--prompt-format` | `HALLUCINATE_PROMPT_FORMAT` | `json` | Prompt serialization format: `json` or `toon` |
| `--max-request-bytes` | `HALLUCINATE_MAX_REQUEST_BYTES` | `10240` (10 KB) | Maximum request body size in bytes |
| `--timeout-seconds` | `HALLUCINATE_TIMEOUT_SECONDS` | `300` | Outbound LLM API call timeout in seconds |
| `--schema-profile` | `HALLUCINATE_SCHEMA_PROFILE` | *(auto — see below)* | Schema profile override for response schema validation. See [jsonschemaprofiles](https://jsonschemaprofiles.unitvectorylabs.com/) for available profiles. |
| `--url` | `HALLUCINATE_URL` | *(auto)* | Override the default API URL |
| `--api-key` | `OPENAI_API_KEY`, `HALLUCINATE_API_KEY` | *(empty)* | API key for bearer authentication |
| `--insecure` | `HALLUCINATE_INSECURE` | `false` | Skip TLS certificate verification for outbound LLM calls |

## Schema Profile Validation

Before making any API call, HallucinateAPI validates response schemas using the [jsonschemaprofiles](https://github.com/UnitVectorY-Labs/jsonschemaprofiles) library. This ensures the schema conforms to the subset of JSON Schema supported by the target provider.

The default profile is automatically selected based on the provider:

| Provider | Default Profile  |
|----------|------------------|
| `gemini` | `GEMINI_202602`  |
| `openai` | `OPENAI_202602`  |

Use the `--schema-profile` flag to override the default.

Available profiles: `OPENAI_202602`, `GEMINI_202602`, `GEMINI_202503`, `MINIMAL_202602`

## URL Behavior

| Condition | URL Used |
|-----------|----------|
| `--url` provided | Uses the provided URL verbatim |
| `--provider=gemini` (no `--url`) | Constructed from `--gcp-project` and `--gcp-location` |
| `--provider=openai` (no `--url`) | `https://api.openai.com/v1/chat/completions` |

The `--url` flag allows using custom endpoints, including:
- Google Cloud's OpenAI-compatible endpoint
- Ollama local instances
- Any OpenAI-compatible API

When connecting to an HTTPS endpoint with a self-signed or otherwise untrusted certificate, `--insecure` disables TLS certificate verification for the HTTP client. This is intended for local development and test environments and should not be used for production traffic.

## Authentication

| Provider | Default Auth | `--api-key` Behavior |
|----------|--------------|---------------------|
| `gemini` | Google Application Default Credentials (ADC) | Overrides ADC with bearer token |
| `openai` | Required via flag or env (unless `--url` is used) | Used as bearer token |

**Gemini provider:** Authenticate with:

```bash
gcloud auth application-default login
# or
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
```

**OpenAI provider:** Provide an API key via:

```bash
--api-key "your-api-key"
# or
export OPENAI_API_KEY="your-api-key"
```

When using `--url` with the openai provider (for local servers like Ollama), the API key is optional.

## Running with Docker

Mount your OpenAPI spec file into the container:

```bash
# Gemini
docker run -p 8080:8080 \
  -v /path/to/spec.yaml:/spec.yaml \
  -e HALLUCINATE_PROVIDER=gemini \
  -e HALLUCINATE_OPENAPI_PATH=/spec.yaml \
  -e GOOGLE_CLOUD_PROJECT=my-project \
  -e HALLUCINATE_GCP_LOCATION=us-central1 \
  -e HALLUCINATE_MODEL=gemini-2.5-flash \
  ghcr.io/unitvectory-labs/hallucinateapi:latest

# OpenAI
docker run -p 8080:8080 \
  -v /path/to/spec.yaml:/spec.yaml \
  -e HALLUCINATE_PROVIDER=openai \
  -e HALLUCINATE_OPENAPI_PATH=/spec.yaml \
  -e HALLUCINATE_MODEL=gpt-4o \
  -e OPENAI_API_KEY=your-api-key \
  ghcr.io/unitvectory-labs/hallucinateapi:latest
```

## Built-in Endpoints

The server hosts the following endpoints automatically:

| Endpoint | Description |
|----------|-------------|
| `/` | Swagger UI for interactive API exploration |
| `/openapi.json` | OpenAPI specification (served if input is JSON) |
| `/openapi.yaml` | OpenAPI specification (served if input is YAML) |

These paths are reserved and must not be defined in your OpenAPI specification.
