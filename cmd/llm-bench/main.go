package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/Aryanbhargava18/llm-bench/pkg/telemetry"
	"github.com/Aryanbhargava18/llm-bench/pkg/tester"
)

func main() {
	provider := flag.String("provider", "anthropic", "LLM provider (anthropic, gemini)")
	concurrency := flag.Int("concurrency", 1, "Number of concurrent requests")
	flag.Parse()

	tp, err := telemetry.InitProvider()
	if err != nil {
		log.Fatalf("failed to initialize otel provider: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	fmt.Printf("Starting TTFT Benchmark\nProvider: %s\nConcurrency: %d\n", *provider, *concurrency)

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			err := tester.RunWorker(workerID, *provider)
			if err != nil {
				log.Printf("Worker %d failed: %v", workerID, err)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("Benchmark complete.")
}
