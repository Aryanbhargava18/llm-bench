package tester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRunWorker_Success(t *testing.T) {
	// 1. Setup OTel InMemory Exporter for validation
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	// 2. Setup mock SSE server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"usage\": {\"prompt_tokens\": 42, \"completion_tokens\": 10}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	// 3. Run worker
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 1, "test-provider", ts.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 4. Assert Spans
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "llm.stream_request" {
		t.Errorf("expected span name llm.stream_request, got %s", span.Name)
	}

	// 5. Assert Semantic Attributes
	var foundTTFT, foundPrompt, foundCompletion bool
	for _, attr := range span.Attributes {
		if attr.Key == "gen_ai.response.ttft_ms" {
			foundTTFT = true
		}
		if attr.Key == "gen_ai.usage.prompt_tokens" && attr.Value.AsInt64() == 42 {
			foundPrompt = true
		}
		if attr.Key == "gen_ai.usage.completion_tokens" && attr.Value.AsInt64() == 10 {
			foundCompletion = true
		}
	}

	if !foundTTFT {
		t.Error("missing gen_ai.response.ttft_ms attribute")
	}
	if !foundPrompt {
		t.Error("missing or incorrect gen_ai.usage.prompt_tokens attribute")
	}
	if !foundCompletion {
		t.Error("missing or incorrect gen_ai.usage.completion_tokens attribute")
	}
}

func TestRunWorker_ErrorMapping(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 2, "test-provider", ts.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", span.Status.Code)
	}

	var foundErrorType bool
	for _, attr := range span.Attributes {
		if attr.Key == "error.type" && attr.Value.AsString() == "429" {
			foundErrorType = true
		}
	}

	if !foundErrorType {
		t.Error("missing or incorrect error.type attribute mapping")
	}
}
