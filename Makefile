.PHONY: build run clean

build:
	go build -o bin/ttft-bench cmd/ttft-bench/main.go

run: build
	./bin/ttft-bench

clean:
	rm -rf bin/
