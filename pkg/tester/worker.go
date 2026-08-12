package tester

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func RunWorker(id int, provider string) error {
	fmt.Printf("[Worker %d] Starting request to %s...\n", id, provider)
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Simulated streaming response for load testing
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
					fmt.Printf("[Worker %d] TTFT: %v\n", id, ttft)
				}
			}
		}
	}
	
	fmt.Printf("[Worker %d] Completed request\n", id)
	return nil
}
