# GitHub Release Workflow Design

## Goal

Automate building and publishing pre-built binaries to GitHub Releases when a version tag is pushed.

## Trigger

Push a tag matching `v*` (e.g., `v0.1.0`).

## Target Platforms

| OS      | Arch  | Artifact Name         |
|---------|-------|-----------------------|
| darwin  | arm64 | devpilot-darwin-arm64   |
| darwin  | amd64 | devpilot-darwin-amd64   |
| linux   | amd64 | devpilot-linux-amd64    |

## Workflow Steps

1. Checkout code
2. Set up Go
3. Build binaries for all 3 targets using `GOOS`/`GOARCH` cross-compilation
4. Inject version via `-ldflags -X main.version=<tag>`
5. Generate SHA256 checksums file
6. Create GitHub Release with `gh release create`
7. Upload all binaries + checksums

## Version Injection

Add a `version` variable to `cmd/devpilot/main.go` and a `--version` flag on the root command. The build injects the tag value at compile time via ldflags.

## Artifact Naming

`devpilot-{os}-{arch}` — no `.exe` suffix since Windows is not supported.

## User Installation

```bash
curl -LO https://github.com/siyuqian/devpilot/releases/latest/download/devpilot-darwin-arm64
chmod +x devpilot-darwin-arm64
sudo mv devpilot-darwin-arm64 /usr/local/bin/devpilot
```

## Approach

Pure GitHub Actions (no GoReleaser). Simple, zero external dependencies, easy to maintain. Can migrate to GoReleaser later if needed (Homebrew tap, etc.).
