BINARY := bin/devpilot
PKG := ./cmd/devpilot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint lint-fix run clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	go test ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -rf bin/
