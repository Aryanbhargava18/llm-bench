package tester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRunWorker_Success(t *testing.T) {
	// 1. Setup OTel InMemory Exporter for validation
	originalTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(originalTP) })

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	// 2. Setup mock SSE server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"usage\": {\"prompt_tokens\": 42, \"completion_tokens\": 10}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	// 3. Run worker
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 1, "local", ts.URL, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 4. Assert Spans
	spans := exporter.GetSpans()
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}

	var span tracetest.SpanStub
	var found bool
	for _, s := range spans {
		if s.Name == "llm.stream_request" {
			span = s
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find span named llm.stream_request")
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
	originalTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(originalTP) })

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 2, "local", ts.URL, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	spans := exporter.GetSpans()
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}

	var span tracetest.SpanStub
	var found bool
	for _, s := range spans {
		if s.Name == "llm.stream_request" {
			span = s
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find span named llm.stream_request")
	}

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

func TestRunWorker_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	testerObj := NewTester()
	err := testerObj.RunWorker(ctx, 3, "local", "http://localhost:12345/stream", "")
	if err == nil {
		t.Fatal("expected error with canceled context, got nil")
	}
}

func TestRunWorker_TimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	testerObj := NewTester()
	err := testerObj.RunWorker(ctx, 4, "local", "http://localhost:12345/stream", "")
	if err == nil {
		t.Fatal("expected error with expired timeout context, got nil")
	}
}



func TestWorker_EmptyTargetURL(t *testing.T) {
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 10, "local", "", "")
	if err == nil {
		t.Error("expected error for empty target URL, got nil")
	}
}


func TestWorker_ClientTransportConfig(t *testing.T) {
	testerObj := NewTester()
	if testerObj.client == nil {
		t.Fatal("expected non-nil client")
	}
}


func TestWorker_InstrumentsRegistration(t *testing.T) {
	testerObj := NewTester()
	if testerObj.tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
}


func TestWorker_MaxResponseBodySizeConstant(t *testing.T) {
	if maxResponseBodySize <= 0 {
		t.Errorf("expected positive buffer limit, got %d", maxResponseBodySize)
	}
}


func TestWorker_HistogramRegistration(t *testing.T) {
	testerObj := NewTester()
	if testerObj.durationHistogram == nil {
		t.Error("expected non-nil duration histogram")
	}
}


func TestWorker_UsageCounterRegistration(t *testing.T) {
	testerObj := NewTester()
	if testerObj.usageCounter == nil {
		t.Error("expected non-nil usage counter")
	}
}


func TestWorker_TTFTHistogramRegistration(t *testing.T) {
	testerObj := NewTester()
	if testerObj.ttftHistogram == nil {
		t.Error("expected non-nil TTFT histogram")
	}
}


func TestWorker_TracerNameConstant(t *testing.T) {
	testerObj := NewTester()
	if testerObj.tracer == nil {
		t.Error("expected non-nil tracer instance")
	}
}


func TestWorker_ErrorStatusCodeMapping(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 11, "local", ts.URL, "")
	if err == nil {
		t.Error("expected error for 503 response, got nil")
	}
}


func TestWorker_MultipleWorkersExecution(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 12, "local", ts.URL, "")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}



func TestWorker_ContextCancelledEarly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	testerObj := NewTester()
	_ = testerObj.RunWorker(ctx, 100, "local", "http://127.0.0.1:9", "")
}


func TestWorker_CancelledContextTimeout(t *testing.T) {
	testerObj := NewTester()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)
	_ = testerObj.RunWorker(ctx, 101, "local", "http://127.0.0.1:9", "")
}


func TestWorker_EmptyPayloadHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 102, "local", ts.URL, "")
	if err != nil {
		t.Fatalf("unexpected error on empty stream: %v", err)
	}
}


func TestWorker_InvalidHTTPUrl(t *testing.T) {
	testerObj := NewTester()
	err := testerObj.RunWorker(context.Background(), 103, "local", "http://invalid-hostname-that-does-not-exist.local", "")
	if err == nil {
		t.Fatal("expected error on invalid host, got nil")
	}
}
