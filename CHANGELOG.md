# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases are git-tagged. The submodules (`logsentry`, `goslog`) are tagged in
lockstep with the root module, e.g. `v1.0.7`, `logsentry/v1.0.7`, `goslog/v1.0.7`.

## [Unreleased]

## [1.1.0] - 2026-06-16

### Changed

- **Breaking:** migrate to `getsentry/sentry-go` v0.46.2 and Go 1.25; route
  logging errors through `logsentry`; make `golog.ErrorHandler` a thread-safe
  accessor (#14). `golog.ErrorHandler` changed from a package variable to a
  getter function: replace `golog.ErrorHandler = fn` with
  `golog.SetErrorHandler(fn)`, and read the handler via `golog.ErrorHandler()`
  or `golog.ErrorHandlerOr(fallback)`.
- Bump `securego/gosec` to v2.27.1 in the `tools` submodule (stays on Go 1.25),
  pulling along `google.golang.org/grpc` 1.81.1 and assorted `golang.org/x/*`
  and `google.golang.org/*` updates.

### Fixed

- UUID parsing now accepts UUID versions 6, 7 and 8.

### Documentation

- Add `CLAUDE.md` with guidance for AI coding agents working in the repo.
- Fix the tag prefix in the redact breaking-change note.

## [1.0.7] - 2026-04-14

### Added

- `Format.Location` for fixed-timezone time formatting.

## [1.0.6] - 2026-04-14

### Added

- Tag-driven struct field logging with `omit`/`redact` modifiers.
- `Timestamp` type for flexible log timestamp parsing.

### Changed

- Move gosec into the `tools` submodule; expand `test-workspace.sh`.
- Bump Go to 1.24.9 and update dependencies (incl. `grpc` 1.79.3).

### Documentation

- Document `Timestamp` and `Format` defaults in the README.

## [1.0.5] - 2026-02-12

### Fixed

- Use `v0.0.0-00010101000000-000000000000` for locally replaced module versions.

## [1.0.4] - 2026-02-12

### Fixed

- `logsentry.WriterConfig`: add nil checks for hub and format in
  `NewWriterConfig`; handle a nil error in `WriteError`.
- Ensure `WriteError` implementations do not panic on a nil error.
- `IsTerminal`: add a `#nosec` directive for the file-descriptor conversion.

## [1.0.3] - 2026-02-10

### Changed

- `WriterConfig`: add `mergeWriterConfigs` and optimize
  `uniqueNonNilWriterConfigs`.

## [1.0.2] - 2026-02-03

### Fixed

- `SubLoggerContext`: self-referential `DerivedConfig` causing a stack overflow.

## [1.0.1] - 2026-01-30

### Added

- `DynDerivedConfig`; keep `DerivedConfig` fast without a mutex.

### Fixed

- `CallbackWriter`: nil pointer dereference in all slice `Write` methods.
- `logsentry` writer: improve panic-recovery error handling; add recovery in
  `CommitMessage` and `FlushUnderlying`.
- `logfile`: recover `RotatingWriter` after a failed rotation.

## [1.0.0] - 2026-01-28

### Added

- Initial tagged release: zero-allocation append-style text output, `Time`
  attrib type for zero-allocation `time.Time` logging, and the `tag-release`
  versioning script.

[Unreleased]: https://github.com/domonda/golog/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/domonda/golog/compare/v1.0.7...v1.1.0
[1.0.7]: https://github.com/domonda/golog/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/domonda/golog/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/domonda/golog/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/domonda/golog/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/domonda/golog/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/domonda/golog/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/domonda/golog/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/domonda/golog/releases/tag/v1.0.0
