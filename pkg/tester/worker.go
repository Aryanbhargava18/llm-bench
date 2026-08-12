package tester

import (
	"fmt"
	"net/http"
	"time"
)

func RunWorker(id int, provider string) error {
	fmt.Printf("[Worker %d] Starting request to %s...\n", id, provider)
	
	// Simulate HTTP request to LLM provider
	client := &http.Client{Timeout: 30 * time.Second}
	
	// We will implement the actual API calls in the next iteration.
	// For now, this just proves the worker scaffold is correct.
	_ = client
	
	time.Sleep(1 * time.Second)
	fmt.Printf("[Worker %d] Completed request\n", id)
	return nil
}
