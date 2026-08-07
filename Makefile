.PHONY: help build run test lint clean

help:
	@echo "Targets: build, run, test, lint, clean"

build:
	go build -o bin/llm-bench cmd/llm-bench/main.go

run: build
	./bin/llm-bench

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out
