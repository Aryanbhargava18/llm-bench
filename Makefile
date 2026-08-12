.PHONY: build run clean

build:
	go build -o bin/llm-bench cmd/llm-bench/main.go

run: build
	./bin/llm-bench

clean:
	rm -rf bin/
