# goperf

CLI HTTP load testing tool (like `hey` or `wrk`). Zero third-party dependencies.

## Build & Run

```bash
go build -o goperf ./cmd/goperf
./goperf -url https://example.com -concurrency 10 -duration 5s
```

## CLI Flags

- `-url` — Target URL (required)
- `-method` — HTTP method (default: GET)
- `-concurrency` — Number of concurrent workers (default: 10)
- `-duration` — Test duration (default: 10s)
- `-timeout` — Per-request timeout (default: 10s)

## Test

```bash
go test ./...                    # run all tests
go test ./... -v                 # verbose
go test -race ./...              # with race detector
go test -cover ./...             # with coverage
go test -run TestName ./internal/...  # run specific test
```

## Lint & Format

```bash
gofmt -w .
go vet ./...
```

## Project Structure

```
cmd/goperf/          # CLI entry point
internal/config/     # CLI flag parsing + validation
internal/engine/     # Orchestrates workers, aggregates results
internal/worker/     # Single worker loop: send requests, push results
internal/report/     # Percentile computation + formatted output
```

## Project Conventions

- **Go version**: 1.22+
- **Dependencies**: stdlib only — no third-party packages
- **Project layout**: Follow standard Go project layout (`cmd/`, `internal/`)
- **Error handling**: Return errors, don't panic. Wrap errors with `fmt.Errorf("context: %w", err)`
- **Naming**: Use idiomatic Go naming — camelCase for unexported, PascalCase for exported
- **Tests**: Place tests in the same package (`_test.go` suffix). Use table-driven tests
- **Config**: Use CLI flags via `flag.NewFlagSet` (testable, no global state)

## Code Style

- Keep functions short and focused
- Prefer composition over inheritance
- Use interfaces at the consumer side, not the producer side
- No global mutable state
- Always handle errors — never use `_` to discard them
