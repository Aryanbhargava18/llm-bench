# ttft-bench-go

A concurrent Go CLI tool for benchmarking Time-To-First-Token (TTFT) latency across streaming LLM API endpoints. Parses Server-Sent Events (SSE) in real-time and emits OpenTelemetry `gen_ai.*` traces for observability.

## Overview

`ttft-bench` fires concurrent requests against streaming LLM providers, parses the SSE stream byte-by-byte, and measures the precise time between request dispatch and the first meaningful token arrival. All measurements are emitted as OpenTelemetry spans following the [`gen_ai.*` semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/).

## Features

- **Concurrent load testing** — configurable worker pool for parallel request dispatch
- **SSE stream parsing** — real-time chunked parsing of `text/event-stream` responses
- **OpenTelemetry instrumentation** — emits `gen_ai.system`, `gen_ai.request.model`, and `gen_ai.response.ttft_ms` span attributes
- **Multi-provider support** — pluggable provider interface for OpenAI, Anthropic, Gemini, and custom endpoints
- **Context propagation** — manual span context management for accurate distributed trace correlation

## Architecture

```
cmd/ttft-bench/
  main.go              # CLI entry point, OTel SDK initialization
pkg/
  telemetry/
    otel.go            # TracerProvider setup, stdout/OTLP exporter config
  tester/
    worker.go          # Concurrent worker pool, SSE parsing, span emission
```

## Usage

```bash
# Build
make build

# Run benchmark against a provider
./bin/ttft-bench --provider=anthropic --concurrency=5

# Export traces to Jaeger
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 ./bin/ttft-bench --provider=openai --concurrency=10
```

## Trace Output

Each benchmark run emits spans with the following attributes:

| Attribute | Description |
|-----------|-------------|
| `gen_ai.system` | Provider name (e.g., `openai`, `anthropic`) |
| `gen_ai.request.model` | Model identifier |
| `gen_ai.response.ttft_ms` | Time-To-First-Token in milliseconds |

## Requirements

- Go 1.21+
- Valid API keys for target providers (set via environment variables)

## License

Apache-2.0
