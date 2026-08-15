# llm-bench

A concurrent CLI tool for load-testing Server-Sent Events (SSE) streams from LLM providers (OpenAI, Anthropic) and measuring observability correctness via OpenTelemetry.

## Why this exists

Benchmarking streaming LLM responses at high concurrency often leads to two problems:
1. **OOM Panics:** Blindly accumulating massive text streams in memory during concurrent load tests causes out-of-memory crashes.
2. **Observability Blindspots:** Measuring Time-To-First-Token (TTFT) and token usage across hundreds of parallel streams is impossible without proper distributed tracing.

`llm-bench` solves this by introducing strict memory bounds (content truncation) during stream parsing, while seamlessly tracking operation durations, TTFT, and trailing usage metrics natively via OpenTelemetry.

## Features

- **Connection Pooling:** Clones `http.DefaultTransport` and configures `MaxIdleConns`/`MaxIdleConnsPerHost` to prevent socket exhaustion under concurrent load while preserving HTTP/2 multiplexing and OS-level TLS defaults.
- **Memory-Bounded Stream Parsing:** Tracks byte throughput via a running counter without accumulating SSE content in memory. Trailing usage frames are extracted from the scanner buffer line-by-line and discarded. This prevents OOM under concurrent load.
- **Provider Agnostic:** Normalizes divergent usage schemas from OpenAI (`prompt_tokens`) and Anthropic (`input_tokens`).
- **Native OpenTelemetry:** Emits OTel Traces and Metrics natively. Measures `gen_ai.response.ttft_ms` and `gen_ai.client.token.usage` using exact GenAI semantic conventions.
- **Graceful Shutdown:** Native SIGINT listening ensures running workers and OpenTelemetry Providers gracefully flush telemetry on cancellation.

## Usage

```bash
make build

# Run against a local mock SSE server
./bin/llm-bench --provider=local --concurrency=5

# Run against real providers (requires API keys)
export OPENAI_API_KEY="sk-..."
./bin/llm-bench --provider=openai --concurrency=10
```

## Testing Methodology

`llm-bench` includes a test suite using `tracetest.InMemoryExporter` to assert the structure and attribute values of emitted OpenTelemetry spans against expected `gen_ai.*` values.

```bash
make test
```

## Requirements
- Go 1.21+
