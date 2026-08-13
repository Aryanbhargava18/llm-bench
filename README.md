# llm-bench

A high-performance, concurrent CLI tool for benchmarking Server-Sent Events (SSE) streams from LLM providers like OpenAI and Anthropic.

## Why this exists

Benchmarking streaming LLM responses at high concurrency often leads to two problems:
1. **OOM Panics:** Blindly accumulating massive text streams in memory during concurrent load tests causes out-of-memory crashes.
2. **Observability Blindspots:** Measuring Time-To-First-Token (TTFT) and token usage across hundreds of parallel streams is impossible without proper distributed tracing.

`llm-bench` solves this by introducing strict memory bounds (content truncation) during stream parsing, while seamlessly tracking operation durations, TTFT, and trailing usage metrics natively via OpenTelemetry.

## Features

- **High Concurrency HTTP:** Utilizes a custom `http.Transport` optimized for massive parallel connections.
- **Graceful Degradation:** Safely truncates response accumulation at 5MB to prevent memory exhaustion, while continuing to parse the stream to intercept trailing usage payloads.
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

`llm-bench` includes a robust test suite that utilizes `tracetest.InMemoryExporter` to mathematically assert the integrity of emitted OpenTelemetry spans and metrics.

```bash
make test
```

## Requirements
- Go 1.21+
