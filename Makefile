.PHONY: fmt download build test clean

all: fmt download build

fmt:
	go fmt

download:
	go mod download

build:
	go build -o bin/cachenator

test: fmt download build
	tests/run_tests.sh

clean:
	rm -rf bin/
	rm -f go.sum
	go clean -modcache
