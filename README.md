# llm-bench

A high-throughput, concurrent load-testing engine and telemetry validator for Server-Sent Events (SSE) streaming APIs across LLM providers (OpenAI, Anthropic, Gemini), with native OpenTelemetry distributed tracing and metrics validation.

```text
                                  +---------------------------------------+
                                  |           llm-bench CLI               |
                                  |   (--concurrency=50 --provider=openai)|
                                  +-------------------+-------------------+
                                                      |
                                        [ Cloned http.Transport ]
                                        (MaxIdleConnsPerHost=100)
                                                      |
                    +---------------------------------+---------------------------------+
                    |                                 |                                 |
                    v                                 v                                 v
            +---------------+                 +---------------+                 +---------------+
            |   Worker 1    |                 |   Worker 2    |                 |   Worker N    |
            | (SSE Stream)  |                 | (SSE Stream)  |                 | (SSE Stream)  |
            +-------+-------+                 +-------+-------+                 +-------+-------+
                    |                                 |                                 |
                    | Line-by-line Scan               | Line-by-line Scan               | Line-by-line Scan
                    v                                 v                                 v
          +-------------------+             +-------------------+             +-------------------+
          | TTFT Calculation  |             | TTFT Calculation  |             | TTFT Calculation  |
          | (1st Event Delta) |             | (1st Event Delta) |             | (1st Event Delta) |
          +---------+---------+             +---------+---------+             +---------+---------+
                    |                                 |                                 |
                    | Running-Max Usage Extraction (Zero Buffer Accumulation)           |
                    +---------------------------------+---------------------------------+
                                                      |
                                                      v
                                      +-------------------------------+
                                      |   OpenTelemetry SDK Exporter  |
                                      |  * gen_ai.response.ttft_ms    |
                                      |  * gen_ai.usage.prompt_tokens |
                                      |  * gen_ai.usage.completion_tokens
                                      +---------------+---------------+
                                                      |
                                                      v
                                        [ OTLP / InMemoryExporter ]
```

---

## Architectural Problem & Design

Load testing streaming LLM endpoints presents two foundational systems challenges that standard HTTP load generators (like `wrk` or `hey`) fail to handle:

1. **Unbounded SSE Buffer Allocation (OOM Panics):** Accumulating streaming chunk deltas into memory buffers under high concurrency causes memory consumption to scale linearly with stream duration and response length ($O(N \times L)$). `llm-bench` implements a line-by-line stream scanner that measures total byte throughput via running counters while extracting trailing usage metadata without in-memory text concatenation ($O(1)$ memory per worker).
2. **Observability Verification:** Verifying that instrumentation accurately captures Time-To-First-Token (TTFT) and normalizes divergent provider schemas (e.g. Anthropic's split `message_start` vs OpenAI's trailing `usage` chunk) requires native OTel span and metric validation.

---

## Architecture & Mechanics

* **Connection Pool Tuning:** Clones `http.DefaultTransport` and explicitly tunes `MaxIdleConns` and `MaxIdleConnsPerHost` to prevent TCP socket exhaustion and ephemeral port starvation while maintaining HTTP/2 multiplexing.
* **Running-Max Usage Accumulator:** Prevents token double-counting across multi-frame SSE streams by taking monotonic maximums across cumulative frame updates.
* **Provider Schema Normalization:**
  * **OpenAI:** Extracts `usage.prompt_tokens` and `usage.completion_tokens` from final chunk frames.
  * **Anthropic:** Normalizes nested `message.usage.input_tokens` from `message_start` events and top-level `usage` from `message_delta` events, folding `cache_read_input_tokens` into total prompt accounting.
* **Native OpenTelemetry Instrumentation:** Every worker execution is wrapped in a root trace span exporting exact semantic convention attributes:
  * `gen_ai.system` (`openai` | `anthropic` | `local`)
  * `gen_ai.request.model`
  * `gen_ai.response.ttft_ms`
  * `gen_ai.usage.prompt_tokens`
  * `gen_ai.usage.completion_tokens`
  * `error.type` (HTTP status code or transport error)

---

## CLI Reference

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--provider` | string | `local` | Target LLM provider (`openai`, `anthropic`, `local`) |
| `--concurrency` | int | `5` | Number of parallel worker goroutines |
| `--endpoint` | string | `""` | Custom API base URL (defaults to provider standard) |
| `--model` | string | `""` | Model identifier (defaults: `gpt-4` / `claude-3-5-sonnet-20241022`) |

### Usage Examples

```bash
# Build binary
make build

# 1. Benchmark local mock SSE server with 10 concurrent streams
./bin/llm-bench --provider=local --concurrency=10

# 2. Run against OpenAI with distributed trace export
export OPENAI_API_KEY="sk-..."
./bin/llm-bench --provider=openai --concurrency=20 --model=gpt-4o

# 3. Run against Anthropic Claude streaming endpoint
export ANTHROPIC_API_KEY="sk-ant-..."
./bin/llm-bench --provider=anthropic --concurrency=15
```

---

## Sample Trace Output

```text
[Worker 1] [TraceID: 3775544a0eeabb94829baab52cdb48ce] Starting request to http://127.0.0.1:59757...
[Worker 1] [TraceID: 3775544a0eeabb94829baab52cdb48ce] TTFT: 783.208µs
[Worker 1] [TraceID: 3775544a0eeabb94829baab52cdb48ce] Completed request. Total bytes: 127
[Worker 2] [TraceID: 8a9310c822eabf41029baab52cdb99fe] Starting request to http://127.0.0.1:59757...
[Worker 2] [TraceID: 8a9310c822eabf41029baab52cdb99fe] TTFT: 812.140µs
[Worker 2] [TraceID: 8a9310c822eabf41029baab52cdb99fe] Completed request. Total bytes: 127
```

---

## Testing & Telemetry Verification

The test suite validates semantic attribute compliance using `go.opentelemetry.io/otel/sdk/trace/tracetest.InMemoryExporter`. Tests spin up local HTTP/SSE servers, execute concurrent workers, and assert span names, status codes, and `gen_ai.*` key-value pairs.

```bash
# Execute unit and telemetry compliance test suite
go test -v -race ./...
```

---

## License

Apache 2.0

<!-- Note: verify Prometheus scrape interval with otelcol -->
