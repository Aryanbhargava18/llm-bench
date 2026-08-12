package tester

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func RunWorker(id int, provider string) error {
	ctx := context.Background()
	tracer := otel.Tracer("llm-bench")
	
	ctx, span := tracer.Start(ctx, "llm.stream_request")
	defer span.End()

	span.SetAttributes(
		attribute.String("gen_ai.system", provider),
		attribute.String("gen_ai.request.model", "dummy-model"),
	)

	fmt.Printf("[Worker %d] Starting request to %s...\n", id, provider)
	
	mockResponse := "data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\ndata: {\"choices\": [{\"delta\": {\"content\": \" World\"}}]}\n\ndata: [DONE]\n\n"
	resp := &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
	}
	
	if resp.StatusCode == 200 {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(mockResponse)))
		var ttft time.Duration
		startTime := time.Now()
		firstChunk := true

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				if firstChunk && line != "data: [DONE]" {
					ttft = time.Since(startTime)
					firstChunk = false
					
					span.SetAttributes(attribute.Float64("gen_ai.response.ttft_ms", float64(ttft.Milliseconds())))
					fmt.Printf("[Worker %d] TTFT: %v\n", id, ttft)
				}
			}
		}
	}
	
	fmt.Printf("[Worker %d] Completed request\n", id)
	return nil
}
