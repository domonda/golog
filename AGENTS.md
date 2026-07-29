# CLAUDE.md

Guidance for agents working in this repository.

## Project

`github.com/domonda/golog` — a fast, zero-allocation structured logging library for Go
(inspired by zerolog). Production-tested at domonda since 2023. Go 1.25.

The public API surface is large but mechanical (see `message.go`, ~87 typed `Message`
field methods). Read `doc.go` and `README.md` for the user-facing overview before adding
features; both must be kept in sync with code changes.

## Multi-module workspace

This is a `go.work` workspace (`go.work`) with several modules. The root module is
intentionally dependency-light; heavier integrations live in their own modules so users
don't pull in transitive deps they don't need.

| Path          | Module                                     | Notes                                                        |
| ------------- | ------------------------------------------ | ------------------------------------------------------------ |
| `.`           | `github.com/domonda/golog`                 | Core library. Released as the root tag (`vX.Y.Z`).           |
| `goslog/`     | `github.com/domonda/golog/goslog`          | `log/slog` backend. Separate module + tag (`goslog/vX.Y.Z`). |
| `logsentry/`  | `github.com/domonda/golog/logsentry`       | Sentry writer. Separate module + tag (`logsentry/vX.Y.Z`).   |
| `benchmarks/` | `github.com/domonda/golog/benchmarks`      | Comparative benchmarks vs zerolog/zap/logrus. Not released.  |
| `examples/`   | `github.com/domonda/golog/examples`        | Runnable examples. Not released.                             |
| `tools/`      | `github.com/domonda/golog/tools`           | `go tool` directive for gosec only. Not released.            |

`goslog` and `logsentry` `replace` the root with `..`, so local edits to the core are
picked up automatically.

Sub-packages **without** their own `go.mod` (part of the root module):
- `log/` — ready-to-use package-level logger (`log.Logger`, `log.Info(...)`, etc.),
  configured from the `LOG_LEVEL` env var with terminal auto-detection (text on a TTY,
  JSON otherwise). See `log/config.go`.
- `logfile/` — size-based rotating file writer.
- `mempool/` — generic pooling primitives (`Pointer[T]`, `Slice[T]`) used everywhere for
  zero-alloc reuse.

## Build, test, lint

Always operate across all modules — a change to the root can break `goslog`/`logsentry`.

```bash
./test-workspace.sh          # build, go vet, gosec, go test across all modules except tools (gosec also skips examples)
./test-workspace.sh -v       # extra args are forwarded to `go test`
./run-gosec.sh               # gosec on the root module only
```

Tests run with `-p 1 -count=1` (serial, no cache) because logging touches global state
(`GlobalPanicLevel`, `ErrorHandler`, the package registry).

Single module / single test during development:
```bash
go build ./... && go vet ./... && go test ./...
go test -run TestName ./...
go tool gosec ./...          # gosec is wired via tools/go.mod `tool` directive
```

CI: `.github/workflows/go.yml` (build + `go test -v` on the root) and `gosec.yml`.

## Conventions specific to this repo

This is a **public, dependency-light library**, so the user's global Go conventions in
`~/.claude/CLAUDE.md` do NOT apply here:
- Uses the **standard library** `errors.New` / `fmt.Errorf`, **not** `go-errs`.
- UUIDs are plain **`[16]byte`** with helpers in `uuid.go` (`UUIDv4`, `ParseUUID`,
  `FormatUUID`), **not** the `uu` package. The `Message.UUID` / `Writer.WriteUUID` API
  takes `[16]byte`.
- Match the existing style: keep zero-allocation discipline (pool and reuse, avoid
  `reflect` on the hot path), and add the corresponding `*_test.go` coverage — most files
  have a paired test.

## Releasing

The `VERSION` file uses Go module version syntax — a leading `v` followed by semver
(`vMAJOR.MINOR.PATCH`, e.g. `v1.1.0`), matching the git release tags.

`./tag-version.sh [message]` reads the version from the `VERSION` file and tags the
root, `goslog/`, and `logsentry/` modules together (it prints the current tags first).
Ask before pushing tags.
