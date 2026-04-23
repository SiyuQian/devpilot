BINARY := bin/devpilot
PKG := ./cmd/devpilot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint lint-fix run clean sync-skills check-skills-sync

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

# Sync source skills/ into the installed .claude/skills/ copy.
# Run after editing any skill; CI uses check-skills-sync to guard against drift.
sync-skills:
	rsync -a --delete skills/ .claude/skills/

check-skills-sync:
	@diff -qr skills/ .claude/skills/ >/dev/null 2>&1 || { \
		echo "skills/ and .claude/skills/ have drifted. Run 'make sync-skills'."; \
		exit 1; \
	}
