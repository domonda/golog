# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases are git-tagged. The submodules (`logsentry`, `goslog`) are tagged in
lockstep with the root module, e.g. `v1.0.7`, `logsentry/v1.0.7`, `goslog/v1.0.7`.

## [Unreleased]

## [1.4.0] - 2026-11-01

### Added

- `ContextWithOwnedAttribs` and `RequestWithOwnedAttribs` hand the passed
  attribs over to the context instead of cloning them, for attribs that are
  built for the call like
  `ContextWithOwnedAttribs(ctx, NewUUID("requestID", requestID))`. Cloning
  those allocates a second attrib and orphans the passed one, because a
  context never returns its attribs to the mempool and the caller has no
  reference left to `Free` them with. Attribs inherited from the parent
  context are still cloned. The HTTP middleware, `ContextWithRequestUUID`
  and `ContextWithRequestID` use them now, so they no longer orphan a pooled
  attrib per request.
- `Logger.CtxAttribs` returns the attribs that a logger inherited from a
  context via `Message.SubLoggerContext`.

### Changed

- **Breaking:** `Message.SubLoggerContext` now also adds the attribs from the
  passed context to the returned sub-logger, so that the sub-logger and the
  returned context have the same union of attribs attached to them. Attribs
  recorded at the message still take precedence over context attribs with the
  same key. Previously the context attribs ended up only in the returned
  context, so the sub-logger lost them when used with a different context.
  Existing code that keeps such a sub-logger therefore logs attribs that it
  did not log before.
  The attribs taken from the context stay context attribs of the sub-logger
  instead of becoming logger attribs: they are logged after the attribs of the
  context that a message is logged with, so an attrib with the same key from
  that context takes precedence over the inherited one. A sub-logger created
  from a request scoped context can therefore be used with the context of
  another request without mislabeling its messages.
- `NewLogger` and `NewLoggerWithPrefix` now store clones of the passed per
  message attribs instead of the attribs and the variadic slice of the caller,
  like every other method that derives a logger. Sharing an attrib let
  `Logger.RemoveAttribs` of one logger return an attrib to the mempool that
  another logger or the caller still used, and sharing the variadic slice put
  a slice that never came from the mempool into it, where it is too small to
  be reused but is taken out and put back on every call. The caller now keeps
  ownership of the passed attribs and may keep using and finally `Free` them.
- `Logger.RemoveAttribs` now also removes and frees the attribs that the
  logger inherited from a context.
- `Attribs.AddToContext` now documents its non-consuming contract explicitly:
  the returned context owns clones of the receiver's attribs and of the attribs
  inherited from the parent context, so the caller keeps ownership of its
  attribs and may keep using and finally `Free` them. A context must own its
  attribs because it has no end of life: an attrib that another owner returns to
  the mempool would be recycled while the context still references it. Same
  contract for `ContextWithAttribs`, `RequestWithAttribs`, and
  `Attribs.AddToRequest`. Behavior is unchanged from v1.2.1.
- `mempool.Slice.GetOrMake` now returns a slice with at least the requested
  capacity. It used to reuse a pooled slice that only fit the requested
  length, so appending up to the requested capacity could allocate a new,
  non-pooled backing array. `Attribs.Add` and
  `Attribs.CloneAndAppendNonExistingCloned` depend on this guarantee.
- **logsentry:** the `Writer` instances and their key-value maps are pooled
  with `mempool.Pointer` and `mempool.Map` instead of hand-rolled
  `sync.Pool`s, so the `mempool` test callbacks also track them.

### Fixed

- `Message.IsAttribRecorder` no longer reports true for a message whose
  writers all filtered out its level, or that was created by a logger without
  writer configs. Such a message has no writers, which used to be the marker
  of an attrib recorder, so `Message.SubLogger` and `Message.SubLoggerContext`
  returned a sub-logger fabricated from the attribs of a message that the
  caller still owns and logs, and put that message back into the mempool,
  which could then hand the same message to two callers at once.
- `Attribs.CloneAndAppendNonExistingCloned` now clones the receiver's attribs as
  documented instead of copying the pooled pointers, so the returned Attribs no
  longer share attribs with the receiver that another owner may free
  (`Logger.RemoveAttribs`, `Attribs.Free`). This affects `Logger.WithClonedAttribs`
  and `Logger.WithCtx`, whose sub-logger no longer shares attribs with its parent.
- `Message.SubContext` no longer hands the returned context the attribs that the
  still usable message keeps ownership of.
- `Logger.WithPrefix` now gets clones of the logger's attribs instead of sharing
  the same slice, like every other method that derives a logger. Sharing it let
  `Logger.RemoveAttribs` on either logger nil out the attribs of the other one
  and return their slice to the mempool while it was still referenced, so the
  next message of that logger panicked or logged recycled attribs.
- Merging attribs with the same key now works the same for the attribs of a
  sub-logger and of a context: both append every attrib of the merged attribs
  whose key is not present in the attribs merged into. `Attribs` explicitly
  allows duplicate keys, and for those the sub-logger created by
  `Message.SubLoggerContext` used to end up with fewer attribs than the
  returned context.
- Merging attribs only grows the pooled slice by the number of attribs that
  are actually appended, and returns the replaced slice to the mempool.
  A slice without spare capacity used to be replaced by a larger pooled one
  even when every key was already present and nothing was appended, and the
  replaced slice was left to the garbage collector instead of being reused.
- `mempool.Slice.GetOrMake` no longer panics when it allocates a new slice
  for a capacity smaller than the length while `MinCap` is zero.

### Documentation

- `AttribsFromContext`, `AttribFromContext`, `Logger.Attribs` and
  `Logger.CtxAttribs` document that the returned attribs are owned by the
  context or logger they come from: don't `Add` to them and don't `Free`
  them, because both return attribs and their slice to the mempool while the
  context or logger still references them. Use `Attribs.Clone` for an owned
  copy.

## [1.3.1] - 2026-10-25

### Changed

- Refreshed dependencies across the workspace: `golang.org/x/sys` v0.49.0
  in every module and `x/term` v0.47.0 in every module that uses it (a direct
  requirement of the root module); **logsentry:** `x/text` v0.43.0;
  **tools:** gosec v2.30.0. The minimum Go version stays 1.26.0.

### Removed

- Dropped the test-only dependency on `ungerik/go-fs`, and with it the
  indirect `fsnotify` and `pkg/xattr`, so they no longer appear in the root
  module's dependency graph. The `logfile` tests now use `t.TempDir` and
  `path/filepath` instead.

### Fixed

- `tag-version.sh` now stops when the `VERSION` is already tagged locally or
  on origin, instead of failing to tag and then deleting the existing release
  tags when the push is declined. It pushes only the new tags (not every local
  tag), and on a declined or failed push it deletes only the tags it created.

## [1.3.0] - 2026-10-13

### Changed

- **Breaking:** raised the minimum Go version to 1.26.0 in every module, to use
  `golang.org/x/sys` v0.48.0 and `x/term` v0.46.0 (both require Go 1.26).
- **Breaking (logsentry):** now requires `sentry-go` v0.50.0 (was v0.48.0),
  so upgrading logsentry also upgrades sentry-go in your build. Its breaking
  changes affect code that configures Sentry directly: v0.49.0 removed
  `ClientOptions.DisableLogs` and `DisableMetrics`, and since v0.50.0
  `Hub.Client()` returns a disabled no-op client instead of nil when no client
  is bound, so replace `hub.Client() != nil` checks with
  `hub.Client().IsEnabled()`. logsentry's own AttachStacktrace lookup now uses
  `IsEnabled`.
- **logsentry:** with `AttachStacktrace` enabled, a message logged while a
  panic unwinds (e.g. from a deferred or recover handler) now carries a
  call-site stack that ends at the panic origin (sentry-go v0.49.0 trims the
  frames above the panic).
- Refreshed dependencies across the workspace: testify v1.12.1;
  **benchmarks:** logrus v1.10.2; **tools:** gosec v2.29.0. Removed the stale
  Go 1.24 `replace` pins for `golang.org/x/sys`, `x/term` and `x/text` from
  `go.work`.

### Fixed

- **logsentry:** `NewSentryDebugWriter` now forwards sentry-go's
  "Failed to create request from envelope" delivery failure, which previously
  dropped the event without reaching the error handler, and the
  "Failed to encode client report" and "Failed to create client report
  request" failures. Every marker now matches a sentry-go v0.50.0 message: the
  `error creating` marker, which matches nothing since sentry-go v0.47.0, was
  removed.

### Documentation

- **logsentry:** note that with `Debug: true` sentry-go echoes Sentry Logs
  message bodies to `DebugWriter`, so application messages containing a marker
  phrase are forwarded as if they were delivery failures.

## [1.2.1] - 2026-07-30

### Changed

- Refreshed dependencies across the workspace and bumped the Go toolchain
  directive to 1.25.8 in every module. **logsentry:** `sentry-go` v0.46.2 →
  v0.48.0 (none of the v0.47/v0.48 breaking changes affect the writer, which
  builds events directly). **benchmarks:** zerolog v1.35.1, zap v1.28.0.
  **examples:** fatih/color v1.19.0. Root module: `ungerik/go-fs` v0.1.0.
  **tools:** gosec v2.28.0. Plus various transitive updates
  (`golang.org/x/sys`, `x/term`, `go-isatty`, `go-colorable`).

## [1.2.0] - 2026-07-01

### Added

- **logsentry:** the error logged via `Message.Err` is now surfaced as a proper
  Sentry exception. Its text becomes the issue subtitle (instead of
  `(No error message)`) while the log message stays the issue title. The
  exception carries the error's own origin stack trace when the error exposes
  one (via a `pkg/errors`-style `StackTrace()`/`StackFrames()` method, extracted
  by `sentry.ExtractStacktrace`), falling back to the current call stack when
  `AttachStacktrace` is enabled. golog-internal frames are filtered out either
  way. Errors logged in a slice (`Message.Errs`) or under other keys stay
  context data only, keeping issue titles and grouping clean.

## [1.1.1] - 2026-06-17

### Added

- Runnable `Example` functions for the primary `Logger` API (`Example`,
  `ExampleLogger`) and the `log` subpackage, so the common usage paths appear
  on pkg.go.dev and are verified by `go test`.
- Package documentation comment for the `log` subpackage.

### Changed

- **logsentry:** bump the indirect `golang.org/x/text` dependency from v0.34.0
  to v0.37.0, aligning it with the `tools` submodule.

### Fixed

- A nil `*Logger` no longer panics on the level methods (`Trace`, `Debug`,
  `Info`, `Warn`, `Error`, `Fatal` and their `At`/`Ctx`/`f`/`fCtx` variants).
  They now return a nil `*Message`, honoring the nil-safe contract documented
  in `doc.go`. Previously they dereferenced `l.config` while reading the level
  and crashed with a nil pointer dereference.

### Documentation

- Clarify that duplicate-key prevention applies to keys inherited from the
  logger, a sub-logger, or the context (the inherited value wins), not to
  duplicate keys added within a single message chain.
- Fix the `doc.go` context example to use `golog.NewString` (there is no
  `golog.Str` constructor).

## [1.1.0] - 2026-06-16

### Changed

- **Breaking:** migrate to `getsentry/sentry-go` v0.46.2 and Go 1.25; route
  logging errors through `logsentry`; make `golog.ErrorHandler` a thread-safe
  accessor (#14). `golog.ErrorHandler` changed from a package variable to a
  getter function: replace `golog.ErrorHandler = fn` with
  `golog.SetErrorHandler(fn)`, and read the handler via `golog.ErrorHandler()`
  or `golog.ErrorHandlerOr(fallback)`.
- **logsentry:** structured log values now populate the Sentry event's `log`
  context instead of the removed `Event.Extra`; a logged `type` key is sent as
  `type_` (reserved-name remap) (#14).
- Bump `securego/gosec` to v2.27.1 in the `tools` submodule (stays on Go 1.25),
  pulling along `google.golang.org/grpc` 1.81.1 and assorted `golang.org/x/*`
  and `google.golang.org/*` updates.
- Replace `tag-release.sh` (version passed as an argument) with `tag-version.sh`,
  which reads the release version from the `VERSION` file.

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

[Unreleased]: https://github.com/domonda/golog/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/domonda/golog/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/domonda/golog/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/domonda/golog/compare/v1.2.1...v1.3.0
[1.2.1]: https://github.com/domonda/golog/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/domonda/golog/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/domonda/golog/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/domonda/golog/compare/v1.0.7...v1.1.0
[1.0.7]: https://github.com/domonda/golog/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/domonda/golog/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/domonda/golog/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/domonda/golog/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/domonda/golog/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/domonda/golog/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/domonda/golog/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/domonda/golog/releases/tag/v1.0.0
