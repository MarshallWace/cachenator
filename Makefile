.PHONY: fmt download build clean

all: fmt download build

fmt:
	go fmt

download:
	go mod download

build:
	go build -o bin/falcon

clean:
	rm -rf bin/
	go clean -modcache
