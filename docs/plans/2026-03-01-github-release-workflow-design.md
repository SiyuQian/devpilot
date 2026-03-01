# GitHub Release Workflow Design

## Goal

Automate building and publishing pre-built binaries to GitHub Releases when a version tag is pushed.

## Trigger

Push a tag matching `v*` (e.g., `v0.1.0`).

## Target Platforms

| OS      | Arch  | Artifact Name         |
|---------|-------|-----------------------|
| darwin  | arm64 | devkit-darwin-arm64   |
| darwin  | amd64 | devkit-darwin-amd64   |
| linux   | amd64 | devkit-linux-amd64    |

## Workflow Steps

1. Checkout code
2. Set up Go
3. Build binaries for all 3 targets using `GOOS`/`GOARCH` cross-compilation
4. Inject version via `-ldflags -X main.version=<tag>`
5. Generate SHA256 checksums file
6. Create GitHub Release with `gh release create`
7. Upload all binaries + checksums

## Version Injection

Add a `version` variable to `cmd/devkit/main.go` and a `--version` flag on the root command. The build injects the tag value at compile time via ldflags.

## Artifact Naming

`devkit-{os}-{arch}` — no `.exe` suffix since Windows is not supported.

## User Installation

```bash
curl -LO https://github.com/siyuqian/developer-kit/releases/latest/download/devkit-darwin-arm64
chmod +x devkit-darwin-arm64
sudo mv devkit-darwin-arm64 /usr/local/bin/devkit
```

## Approach

Pure GitHub Actions (no GoReleaser). Simple, zero external dependencies, easy to maintain. Can migrate to GoReleaser later if needed (Homebrew tap, etc.).
