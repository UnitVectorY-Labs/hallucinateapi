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

HallucinateAPI is an HTTP server that implements all GET and POST operations defined in an OpenAPI specification. Each operation is served by calling a Gemini model on Vertex AI, using the operation's description as the LLM instruction and the response schema for structured JSON output.

## Commands

### `serve` (default)

Starts the HTTP server. Runs all validations on startup and exits non-zero if validation fails.

```bash
hallucinate serve --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash
```

Running with no subcommand defaults to `serve`:

```bash
hallucinate --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash
```

### `validate`

Loads configuration and OpenAPI file, runs all validations, and outputs results in both JSON and human-readable text. Exits `0` if valid, non-zero if invalid.

```bash
hallucinate validate --openapi-path /path/to/spec.yaml --gcp-project my-project --gcp-location us-central1 --model gemini-2.5-flash
```

## Configuration

All settings can be set via environment variables or CLI flags. **CLI flags take precedence** over environment variables.

### Required Settings

| Flag | Environment Variable(s) | Description |
|------|------------------------|-------------|
| `--openapi-path` | `HALLUCINATE_OPENAPI_PATH` | Path to the OpenAPI specification file (JSON or YAML) |
| `--gcp-project` | `GOOGLE_CLOUD_PROJECT`, `HALLUCINATE_GCP_PROJECT` | Google Cloud Platform project ID |
| `--gcp-location` | `HALLUCINATE_GCP_LOCATION` | Vertex AI location (e.g., `us-central1` or `global`) |
| `--model` | `HALLUCINATE_MODEL` | Gemini model name (e.g., `gemini-2.5-flash`) |

When `--gcp-location=global`, requests are sent to `https://aiplatform.googleapis.com`. Regional locations use `https://<location>-aiplatform.googleapis.com`.

### Server Settings

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--listen-addr` | `HALLUCINATE_LISTEN_ADDR` | `:8080` | Address and port to listen on |
| `--base-system-prefix` | `HALLUCINATE_SYSTEM_PREFIX` | *(empty)* | Custom prefix added to the system prompt for all operations |
| `--prompt-format` | `HALLUCINATE_PROMPT_FORMAT` | `json` | Prompt serialization format: `json` or `toon` |
| `--max-request-bytes` | `HALLUCINATE_MAX_REQUEST_BYTES` | `10240` (10 KB) | Maximum request body size in bytes |
| `--timeout-seconds` | `HALLUCINATE_TIMEOUT_SECONDS` | `300` | Outbound Gemini API call timeout in seconds |

## Authentication

HallucinateAPI uses **Google Application Default Credentials (ADC)** to authenticate with Vertex AI. No API keys are required.

Ensure ADC is configured in your environment:

```bash
# For local development
gcloud auth application-default login

# For GKE or Cloud Run, ADC is typically configured automatically
```

## Running with Docker

Mount your OpenAPI spec file into the container:

```bash
docker run -p 8080:8080 \
  -v /path/to/spec.yaml:/spec.yaml \
  -e HALLUCINATE_OPENAPI_PATH=/spec.yaml \
  -e GOOGLE_CLOUD_PROJECT=my-project \
  -e HALLUCINATE_GCP_LOCATION=us-central1 \
  -e HALLUCINATE_MODEL=gemini-2.5-flash \
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
