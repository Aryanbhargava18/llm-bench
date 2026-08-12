package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Aryanbhargava18/llm-bench/pkg/telemetry"
	"github.com/Aryanbhargava18/llm-bench/pkg/tester"
)

func runMockServer(port int, fail bool) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// simulate TTFT delay
		time.Sleep(100 * time.Millisecond)

		chunks := []string{
			"data: {\"choices\": [{\"delta\": {\"content\": \"Hello\"}}]}\n\n",
			"data: {\"choices\": [{\"delta\": {\"content\": \" world\"}}]}\n\n",
			"data: {\"choices\": [{\"delta\": {\"content\": \" from\"}}]}\n\n",
			"data: {\"choices\": [{\"delta\": {\"content\": \" local\"}}]}\n\n",
			"data: {\"choices\": [{\"delta\": {\"content\": \" server\"}}]}\n\n",
			"data: {\"usage\": {\"prompt_tokens\": 10, \"completion_tokens\": 5}}\n\n",
			"data: [DONE]\n\n",
		}

		for _, chunk := range chunks {
			fmt.Fprint(w, chunk)
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("mock SSE server running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("mock server failed: %v", err)
	}
}

func main() {
	provider := flag.String("provider", "local", "LLM provider (local, openai, anthropic, gemini)")
	concurrency := flag.Int("concurrency", 1, "Number of concurrent requests")
	port := flag.Int("port", 8080, "Port for local mock server")
	fail := flag.Bool("fail", false, "Simulate a 429 Too Many Requests error")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tp, mp, err := telemetry.InitProvider()
	if err != nil {
		log.Fatalf("failed to initialize otel provider: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()
	defer func() { _ = mp.Shutdown(context.Background()) }()

	targetURL := "http://localhost:8080/stream"
	var apiKey string

	if *provider == "local" {
		targetURL = fmt.Sprintf("http://localhost:%d/stream", *port)
		go runMockServer(*port, *fail)
		time.Sleep(200 * time.Millisecond) // wait for server
	} else if *provider == "openai" {
		targetURL = "https://api.openai.com/v1/chat/completions"
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("OPENAI_API_KEY environment variable is required for provider=openai")
		}
	} else if *provider == "anthropic" {
		targetURL = "https://api.anthropic.com/v1/messages"
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			log.Fatal("ANTHROPIC_API_KEY environment variable is required for provider=anthropic")
		}
	}

	fmt.Printf("target: %s, concurrency: %d\n", targetURL, *concurrency)

	t := tester.NewTester()

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			err := t.RunWorker(ctx, workerID, *provider, targetURL, apiKey)
			if err != nil {
				log.Printf("Worker %d failed: %v", workerID, err)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("done.")
}
