BINARY := bin/devkit
PKG := ./cmd/devkit

.PHONY: build test run clean

build:
	go build -o $(BINARY) $(PKG)

test:
	go test ./...

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -rf bin/
