BINARY := sonora
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint run fmt clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/sonora

test:
	go test -race ./...

lint:
	gofmt -l .
	go vet ./...
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin
