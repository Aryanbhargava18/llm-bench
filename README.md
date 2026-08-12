# llm-bench

A sandbox tool to validate OpenTelemetry `gen_ai.*` attribute semantics and bounded SSE streaming behavior before contributing to OpenTelemetry instrumentation upstream.

## Why this exists

When instrumenting GenAI SDKs (like `openai-go` or `anthropic-sdk-go`), Server-Sent Events (SSE) streams can grow indefinitely during long completions. Accumulating these in a single string can cause OOM panics. 

Additionally, we need to ensure correct `gen_ai.*` attributes (like `gen_ai.system`, `gen_ai.request.model`, and `gen_ai.response.ttft_ms`) are emitted consistently across both standard and streaming response paths.

`llm-bench` isolates these problems outside of the complex `otelc` compile-time toolchain. It spins up a local mock SSE server, fires concurrent HTTP requests, parses the stream with explicit memory bounds (truncated via `strings.Builder`), and emits OpenTelemetry traces **and metrics**.

> **Note on Naming:** This repository was originally built as a naive latency benchmarker (hence `llm-bench`). It has since been entirely repurposed and rebuilt as a specialized sandbox to validate OpenTelemetry GenAI semantic conventions in a controlled environment.

## Testing Methodology

To prove semantic compliance programmatically, `llm-bench` uses `go.opentelemetry.io/otel/sdk/trace/tracetest`. The `worker_test.go` suite spins up in-memory `httptest` servers to stream mocked responses, capturing the emitted spans via `tracetest.InMemoryExporter`. 

This guarantees that memory truncation limits are respected and that all emitted attributes (like `gen_ai.response.ttft_ms` and `error.type`) strictly adhere to OTel conventions—solving the "testability and data quality" challenges often encountered in upstream GenAI adapters.

## Usage

```bash
make build
./bin/llm-bench --provider=local --concurrency=5
```

This uses the local test server by default so it works out-of-the-box without requiring API keys.

## What it actually does

- Makes real HTTP requests and parses `text/event-stream` chunks over the network.
- Uses explicit length truncation (`maxResponseBodySize`) during string accumulation to prevent OOM panics.
- Continues parsing the stream after content truncation to intercept trailing usage metrics, mirroring the exact architectural constraint of upstream OTel streaming instrumentation.
- Emits standard `error.type` attributes upon HTTP failures, mapping transport errors to semantic conventions.
- Records OTel Metrics natively, initializing a `MeterProvider` to capture `gen_ai.client.token.usage` (Counters) and `gen_ai.client.operation.duration` (Histograms).
- Injects `traceparent` context into HTTP headers using `propagation.TraceContext`.
- Accurately measures Time-To-First-Token from HTTP dispatch to the first `data:` chunk parsed from the network buffer.

## Requirements
- Go 1.21+
