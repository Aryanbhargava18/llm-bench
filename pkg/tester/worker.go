package tester

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const maxResponseBodySize = 1024 * 1024 * 5 // 5MB limit for accumulation

// Tester encapsulates the HTTP client and OTel instruments to prevent recreation on every request.
type Tester struct {
	client            *http.Client
	tracer            trace.Tracer
	usageCounter      metric.Int64Counter
	durationHistogram metric.Float64Histogram
}

func NewTester() *Tester {
	meter := otel.Meter("llm-bench")
	usageCounter, _ := meter.Int64Counter("gen_ai.client.token.usage")
	durationHistogram, _ := meter.Float64Histogram("gen_ai.client.operation.duration")

	return &Tester{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100, // crucial for high concurrency benchmarking
			},
		},
		tracer:            otel.Tracer("llm-bench"),
		usageCounter:      usageCounter,
		durationHistogram: durationHistogram,
	}
}

func (t *Tester) RunWorker(ctx context.Context, id int, provider string, targetURL string, apiKey string) error {
	ctx, span := t.tracer.Start(ctx, "llm.stream_request")
	defer span.End()

	// Model name is populated below when building the request payload
	span.SetAttributes(
		attribute.String("gen_ai.system", provider),
	)

	fmt.Printf("[Worker %d] Starting request to %s...\n", id, targetURL)

	var req *http.Request
	var err error
	var modelName string

	if provider == "local" {
		modelName = "dummy-model"
		req, err = http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	} else if provider == "openai" {
		modelName = "gpt-4o-mini"
		payload := map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": "Say hello"},
			},
			"stream": true,
			"stream_options": map[string]bool{
				"include_usage": true,
			},
		}
		bodyBytes, _ := json.Marshal(payload)
		req, err = http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(bodyBytes))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	} else if provider == "anthropic" {
		modelName = "claude-3-haiku-20240307"
		payload := map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": "Say hello"},
			},
			"max_tokens": 1024,
			"stream": true,
		}
		bodyBytes, _ := json.Marshal(payload)
		req, err = http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(bodyBytes))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	} else {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Propagate trace context into HTTP headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Record dynamic model name
	span.SetAttributes(attribute.String("gen_ai.request.model", modelName))

	// Track dispatch time for operation duration and TTFT
	dispatchTime := time.Now()
	
	// Ensure total operation duration is recorded correctly per OTel spec
	defer func() {
		duration := time.Since(dispatchTime)
		t.durationHistogram.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
			attribute.String("gen_ai.system", provider),
		))
	}()

	resp, err := t.client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "network_error"))
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%d", resp.StatusCode)))
		return err
	}

	scanner := bufio.NewScanner(resp.Body)
	var ttft time.Duration
	firstChunk := true

	// bounded accumulation to prevent OOM
	var sb strings.Builder
	var totalBytes int
	var truncated bool

	for scanner.Scan() {
		line := scanner.Text()
		
		// enforce memory bounds for content, but continue scanning for trailing usage metrics
		lineBytes := len(line)
		if !truncated && totalBytes+lineBytes > maxResponseBodySize {
			fmt.Printf("[Worker %d] stream exceeded %d bytes, truncating content accumulation\n", id, maxResponseBodySize)
			truncated = true
		}
		
		if !truncated {
			totalBytes += lineBytes
			sb.WriteString(line)
			sb.WriteString("\n")
		}

		if strings.HasPrefix(line, "data:") {
			if firstChunk && line != "data: [DONE]" {
				ttft = time.Since(dispatchTime)
				firstChunk = false
				
				// record ttft_ms metric in trace (not the histogram, which is for total duration)
				span.SetAttributes(attribute.Float64("gen_ai.response.ttft_ms", float64(ttft.Milliseconds())))
				fmt.Printf("[Worker %d] TTFT: %v\n", id, ttft)
			}
			
			// extract usage metrics if present (typically in the final chunk)
			if strings.Contains(line, "\"usage\"") {
				var payload struct {
					Usage struct {
						PromptTokens     int `json:"prompt_tokens"`
						CompletionTokens int `json:"completion_tokens"`
						InputTokens      int `json:"input_tokens"`
						OutputTokens     int `json:"output_tokens"`
					} `json:"usage"`
				}
				jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if err := json.Unmarshal([]byte(jsonStr), &payload); err == nil {
					
					promptTokens := payload.Usage.PromptTokens
					if promptTokens == 0 {
						promptTokens = payload.Usage.InputTokens
					}
					
					completionTokens := payload.Usage.CompletionTokens
					if completionTokens == 0 {
						completionTokens = payload.Usage.OutputTokens
					}

					if promptTokens > 0 {
						span.SetAttributes(attribute.Int("gen_ai.usage.prompt_tokens", promptTokens))
						t.usageCounter.Add(ctx, int64(promptTokens), metric.WithAttributes(
							attribute.String("gen_ai.system", provider),
							attribute.String("gen_ai.token.type", "prompt"),
						))
					}
					if completionTokens > 0 {
						span.SetAttributes(attribute.Int("gen_ai.usage.completion_tokens", completionTokens))
						t.usageCounter.Add(ctx, int64(completionTokens), metric.WithAttributes(
							attribute.String("gen_ai.system", provider),
							attribute.String("gen_ai.token.type", "completion"),
						))
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	fmt.Printf("[Worker %d] Completed request. Total accumulated bytes: %d\n", id, totalBytes)
	return nil
}
