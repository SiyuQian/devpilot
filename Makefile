BINARY := bin/devpilot
PKG := ./cmd/devpilot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test run clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	go test ./...

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -rf bin/
