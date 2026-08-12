# TTFT Bench (Time-To-First-Token Benchmark)

A Go-based CLI tool to load-test and benchmark Server-Sent Events (SSE) streaming latency for GenAI APIs (like OpenAI and Anthropic).

## Features
- Fires concurrent requests to LLM endpoints.
- Parses SSE streams to calculate exact TTFT (Time-To-First-Token).
- Integrates with OpenTelemetry to emit `gen_ai.*` standard traces.

## Usage
```bash
make build
./bin/ttft-bench --provider=anthropic --concurrency=5
```
