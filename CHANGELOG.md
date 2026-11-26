# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases are git-tagged. The submodules (`logsentry`, `goslog`) are tagged in
lockstep with the root module, e.g. `v1.0.7`, `logsentry/v1.0.7`, `goslog/v1.0.7`.

## [Unreleased]

## [1.5.0] - 2026-11-26

### Changed

- **Breaking:** raised the minimum Go version to 1.27.0 in every module, to use
  generic methods. v1.4.x is the last release line that supports Go 1.26;
  v1.5.x and later require Go 1.27.
- **Breaking:** these `Message` methods are generic, so named types no
  longer need a conversion at the call site:
  - `Str`, `StrMax`, `StrPtr`, `Strs` accept any type with `string` as
    underlying type (`Strs("tags", []Tag{...})`).
  - `Int`, `IntPtr`, `Ints` accept every signed integer type
    (`Int("retries", cfg.MaxRetries)`), `Uint`, `UintPtr`, `Uints` every
    unsigned integer type including `uintptr`, and `Float`, `FloatPtr`,
    `Floats` both `float32` and `float64` types. The writers still receive
    `int64`, `uint64` and `float64`.
  - The method the caller picks decides how a value is logged: `Int` logs
    an integer, `Str` a string. Named types are written as their underlying
    value even if they have their own format: `Int("timeout", time.Second)`
    now compiles and logs `1000000000`, and `Uint("mode", os.ModePerm)`
    logs `511`. Use `Duration` to log a `time.Duration` as `"1s"`, and
    `Stringer` to log the result of a `String` method, like `"-rwxrwxrwx"`
    for `Stringer("mode", os.ModePerm)`.
  - `Bool`, `BoolPtr`, `Bools` accept any type with `bool` as underlying
    type, `UUIDPtr` and `UUIDs` any type with `[16]byte` as underlying type
    (`UUIDs("ids", []uuid.UUID{...})`).
  - The fixed-size variants (`Int8`…`Int64`, `Uint8`…`Uint64`, `Float32`)
    and `UUID` keep their signatures. The fixed-size methods now forward to
    the generic ones instead of repeating their bodies, with the same output.
  - The generic methods never call the `Log` method of a type that
    implements `Loggable`: they write the underlying value under the given
    key. A named type that redacts a secret in `Log`, like
    `type Password string`, now compiles with `Str` without a conversion
    and is logged in plain text. Pass it to `Loggable` or `Any` instead.
    The same goes for a named type that hides its value in a `String`
    method, use `Stringer` for it, or in a `MarshalText` or `MarshalJSON`
    method, use `Any` for it.

  Calls with values of the old parameter types compile unchanged. These
  calls break:
  - An untyped constant whose default type is not in the method's type set:
    `Uint("n", 5)` and `Float("x", 1)` no longer compile. Write
    `Uint("n", uint(5))` or `Uint64("n", 5)`, and `Float("x", 1.0)`. The
    same goes for a float constant with a whole-number value passed to
    `Int` or `Uint`: `Int("limit", 1e6)` or `Int("n", 3.0)` used to convert
    to `int` and now fails because the constant defaults to `float64`.
    Write `Int("limit", 1_000_000)` or `Int("limit", int(1e6))`.
  - Untyped `nil` passed to a generic `…Ptr` or slice method, for example
    `IntPtr("n", nil)` or `Strs("s", nil)`, can't infer the type. For a
    literal `nil` pointer call `Nil("n")`, which logs the same; otherwise
    write a typed nil like `(*int)(nil)` or `[]string(nil)`. Pointer and
    slice variables work unchanged, nil or not.
  - A generic method can't satisfy an interface method, so `*Message` no
    longer implements interfaces declaring for example
    `Str(key, val string) *Message`.
  - Method values like `m.Str` or `(*Message).Int` need an explicit
    instantiation such as `m.Str[string]`.
  - Reflection no longer finds the generic methods: `reflect` leaves them
    out of the method set of `*Message`, so `MethodByName("Str")` returns
    false and a `text/template` call like `{{.Str "k" "v"}}` fails at run
    time. The fixed-size methods like `Int64` are still found.
- **Breaking:** `NewBoolsCopy` is generic and accepts a slice of any type with
  `bool` as underlying type, like the other `New…Copy` constructors. An
  untyped `nil` argument (`NewBoolsCopy("k", nil)`) can't infer the type,
  write `[]bool(nil)`; function values need `NewBoolsCopy[bool]`.
  `NewUintsCopy` also accepts `uintptr` types.

## [1.4.1] - 2026-11-25

### Fixed

- Logging never panics or hangs because of a logged value:
  - A panic of a method that golog calls on a logged value, like `Error`,
    `String`, `Log`, `IsNull`, `IsZero`, `MarshalJSON` and `MarshalText`,
    is recovered and logged as error string, like
    `golog: recovered panic in Error method of T: ...`, also when the
    panic value panics again while it is formatted. This includes
    `Message.Err`, `Message.Error`, `Message.Errors`, `Message.Stringer`,
    `Message.AsJSON`, `Message.Loggable`, `Message.Any`, `Message.Print`,
    `Message.StructFields`, the formatting methods like `Logger.Infof`,
    `LevelWriter` and the attribs of sub-loggers. `Message.Err` and
    `Message.Error` log a nil pointer error as `null` instead of calling
    its `Error` method, `Message.Stringer` logs a nil pointer as `null`
    if its `String` method panics, like with a value receiver, and
    `Message.Request` logs nothing for a nil request instead of panicking. `Message.Loggable` logs nothing for the
    Loggables that `Message.Any` writes as `null`: a nil pointer, a nil
    `LoggableFunc` or a pointer to one, and a Loggable whose `IsNull`
    method returns true, instead of logging the recovered panic of
    calling it as `error` attrib.
  - A value that references itself is logged with the error string
    `golog: encountered a cycle via T` where the cycle closes, instead of
    recursing until the stack overflows, which is a fatal error that
    recover can't catch. This covers a `[]any` or map that contains itself
    in `Message.Any`, `Message.Print`, the formatting methods like
    `Logger.Infof` and `Logger.Errorf` for every verb, `LevelWriter` and
    `log.FatalAndPanic`, also when passed as `reflect.Value` or as pointer
    to a struct, slice, array or map, which fmt formats like `&{...}`
    with the value that it points to; Loggables
    that log each other, through slices, pointers or `Message.Loggable`;
    values that reference themselves through `Attribs`, which
    `encoding/json` marshals with `Attribs.MarshalJSON`, which reports a
    cycle when the same goroutine marshals the same `Attribs` nested in
    themselves, without making goroutines wait for each other; and
    values in the
    attribs of sub-loggers, which end a cycle at its first repetition,
    so a value that contains itself many times is no longer written in
    time and space exponential in the number of times.
  - Loggables that can't be told apart, like struct values and
    `LoggableFunc`s, that log themselves end with an error after at most
    100000 `Log` calls inside one log message. Values nested more than
    256 deep through pointers, slices, arrays, maps and Loggables are written with
    the error `golog: T nested more than 256 deep`.
  - A value that references the same values many times, like slices that
    contain the same slice several times on every level, also nested
    slices of numbers like `[][][]int`, is written as error instead of
    taking exponential time: the fmt package and `encoding/json` write a
    value for every reference. The values of a log message may have at
    most 1000000 slices, arrays, maps and structs, counted for every
    reference, with at most 16777216 elements, and be nested at most 10000
    deep to be formatted or marshalled, and the Loggables of a log message
    may make at most 100000 `Log` calls. Numbers and strings in an
    interface, like the elements of a `[]any`, are not counted as values,
    so a large `[]any` of numbers is still written, and bytes count as
    one element. Every writer of a message writes its values the same
    way. `Attribs` that `encoding/json` marshals may be nested at most
    1000 deep in each other, and inside one value the same `Attribs` may
    be marshalled again at most 1000 times. `Attribs.MarshalJSON` tracks
    this only for `Attribs` with values that can lead to `Attribs`: through
    interfaces, Loggables or `MarshalJSON`, `MarshalText`, `Error` and
    `String` methods of types outside of the standard library, which
    makes marshalling such `Attribs` slower.
  - Not covered by the limits: a `String` or `Error` method that the fmt
    package calls for a value nested in a formatted value and that panics
    with a value that contains itself, `Attribs` next to each other in a
    value that `encoding/json` marshals, which each get their own limits,
    and the bytes of strings, so a slice that references the same large
    string many times is written completely. A struct field that
    `encoding/json` leaves out because another field has its name is
    checked like the others.
  - A `Log`, `Error`, `String` or `MarshalJSON` method that panics because
    it logs a message at `GlobalPanicLevel` is recovered like any other
    panic of the method and logged as error string instead of panicking.
  - Values that `encoding/json` marshals are checked like it marshals
    them: a pointer cycle is reported as error after 1000 nested pointers,
    maps and slices, which used to use 512 MB of stack, embedded fields
    with a name in their json tag are checked as regular fields like
    `encoding/json` marshals them, and the fields
    that it leaves out, like a `json:"-"` field, and the values that it
    marshals with a `MarshalJSON` or `MarshalText` method, also with
    pointer receiver, are not counted, so a tree with a `json:"-"` parent
    pointer is logged. The elements of slices, arrays and maps behind
    pointers, also in struct fields, are counted for every reference.
  - A typed nil error returned by a `MarshalJSON` or `MarshalText` method
    is logged as `null` instead of panicking.
  - The formatting methods check each argument with the methods that fmt
    uses for its verbs, like fmt parses the format, also for `%w` outside
    of `Errorf` and `%#w`, which fmt formats without the `Error` method,
    invalid argument indexes like `%[0]v`, and verbs like `%s` for which
    fmt formats a pointer nested in a value with the value that it points
    to. An argument of `%T`, of `%p` of a pointer, and of a `*` width or
    precision is not checked. A panic value is described by its `Error` or
    `String` method, and the formatting methods pass the format and its
    arguments on to fmt unchanged when they don't replace one, so
    `go vet` checks the format strings of `Logger.Infof` and the like.
  - `Message.Any` and the `Any` attrib write a pointer to an array that
    contains the pointer, like `a := &[1]any{}; a[0] = a`, with a cycle
    error, and arrays nested more than 256 deep with an error, instead of
    crashing with a stack overflow.
  - `Message.Any` follows at most 8 pointers and interfaces and writes a
    pointer cycle like `var v any; v = &v` or a pointer type like
    `type P *P` that points to itself as `null`, and so does a chain of
    more than 8 pointers to a value.
  - `Message.StructFields` and `Message.TaggedStructFields` no longer hang
    for a pointer type like `type P *P` that points to itself, and no
    longer crash with a stack overflow for a struct that embeds a pointer
    to itself, directly or through other embedded structs: it is logged
    once, and embedded pointers are followed at most 256 deep. A struct
    that several embedded pointers point to is logged once instead of
    once for every path to it, which could take exponential time. They no
    longer panic for an embedded field that is no struct, like an embedded
    interface or string type. Like `encoding/json`, such a field is a
    regular field under its type name if that is exported.
- A custom `Writer` whose `WriteError` method calls the `Error` method has
  to call it before writing anything, as `Writer.WriteError` now documents,
  because golog recovers a panic of the `Error` method and calls
  `WriteError` again with an error that reports the panic.
- **goslog:** the `slog.Handler` resolves a `slog.LogValuer` with
  `slog.Value.Resolve`, which logs a panic of its `LogValue` method as
  error instead of panicking, logs groups nested more than 64 deep, like
  a `LogValuer` whose group contains itself, as error instead of
  recursing until the stack overflows, and writes at most 10000
  attributes per record, so groups that contain themselves several times
  don't take exponential time. A record with exactly 10000 attributes is
  written completely, the error replaces the first attribute above the
  limit.
- **logsentry:** an error whose `Error` method panics no longer drops the
  whole Sentry event; the event has the recovered panic as error.
- The text writer no longer writes a second separator after an error in
  a slice whose `Error` method panicked.
- `Message.Any` and the `Any` attrib log a nil `LoggableFunc`, also as an
  element of a slice or array and behind a pointer, as `null` (the text
  writer as `nil`) instead of panicking.
- `Message.Any` and the `Any` attrib find a `Loggable`, an `error` or an
  `IsNull` method with pointer receiver behind several pointers. A `**T`
  whose `*T` has the methods was written by `encoding/json` as `{}`
  instead of as `null` for a null value or as the attribs its `Log` method
  logs. Likewise a `**golog.Attribs` is written as JSON object instead of
  as array of one object, and a slice behind a pointer to an interface,
  like a `*any`, as array instead of as values without array, which was
  invalid JSON.
- A value that implements `error` is logged as its error string instead of
  as the struct or slice that implements it, also as an element of a slice
  or array: `Message.Any` writes a slice type that is an error, like
  `type Errs []string` with an `Error` method, as one string instead of
  as array of the same string, and a slice type whose `IsNull` method
  returns true as `null` instead of `[null]`. A value whose `Error` method
  has a pointer receiver is no `error` and is logged as the value.
- Secrets that a type redacts itself are no longer logged:
  - `Message.Any` uses the `MarshalJSON` or `MarshalText` method of every
    kind of value like `encoding/json`, for example of a
    `type Secret string` or a slice type, which it used to log raw.
    A named UUID type with such a method, like a session token, is logged
    with the method, unless the method formats it as UUID like the UUID
    type of `github.com/google/uuid`, which is still logged as UUID, and
    a nil UUID as `null`.
  - `Message.Any` marshals an addressable struct or map, like one passed as
    pointer or an element of a slice, through its pointer like
    `encoding/json`, so a `MarshalJSON` or `MarshalText` method with
    pointer receiver is used instead of logging a copy. Such a method is
    now called when the value is logged, so like with `encoding/json` it
    must not lock a mutex that the caller holds while logging.
  - `Message.StructFields` logs the fields of a struct passed as pointer
    through their addresses, so the `MarshalJSON` and `MarshalText`
    methods with pointer receiver of the fields, and of the structs and
    array elements in them, are used. For a sub-logger it records a copy
    that doesn't change with the struct.
  - `Message.Any` writes a slice nested in a slice as JSON array like the
    `Any` attrib, if its elements are no plain numbers, strings or bools,
    so the `MarshalJSON` methods of the elements redact them, instead of
    formatting it with the fmt package as one string like `"[pw]"`. For
    example `[]any{[]any{1, "a"}}` is written as `[[1,"a"]]` instead of
    `["[1 a]"]`. A slice of plain values like `[][]int` is still written
    as strings like `["[1 2]"]`, and so is a slice whose elements have a
    `String` method, but no `MarshalJSON`, `MarshalText`, `Error` or `Log`
    method, like `[]time.Duration`, so a `String` method that hides a
    secret keeps it hidden. A nested slice or array type with such a
    `String` method is written with it, and so is every element of such a
    type in a slice that is nested in a slice, also in a `[]any`.
  - `Any.ValueString` writes a value that is no plain number, string or
    bool like it is logged as JSON instead of in Go syntax, which showed
    what its `Log`, `Error` or `MarshalJSON` methods hide. Complex
    numbers, channels and functions, which JSON can't encode, are still
    written in Go syntax.
  - A Loggable in a slice that is logged with `Message.Any` inside another
    Loggable's `Log` method is logged with its `Log` method instead of by
    `encoding/json`, which logged the fields that `Log` hides.
- The `Any` attrib, which records the values of sub-loggers and of
  Loggables logged with `Message.Any`, writes values like `Message.Any`:
  - `null` for a value whose `IsNull` method returns true, also with
    pointer receiver and for an `error`, a Loggable or a type with a
    `MarshalJSON` method, and for a pointer to a zero `time.Time`.
  - Slices and arrays element by element, also behind pointers and
    interfaces, so errors are written as error strings, for example
    `[]error` as `["a",null]` instead of as `[{},null]` by
    `encoding/json`, and Loggables, UUIDs, times and bytes like single
    values.
  - UUIDs behind pointers and interfaces and named UUID types as UUID
    strings instead of arrays of 16 numbers, and bytes behind a pointer to
    an interface as string instead of base64.
  - `Message.Any` and the `Any` attrib decide which values they write as
    one value, like a Loggable, an error, a UUID or a value with a
    `MarshalJSON` method, in one place, so they write every value the
    same way.
- `Message.Any` logs a `[16]byte` as UUID whatever it holds, also behind
  a pointer, where one that holds no valid UUID was logged as array of
  one UUID, and as array of 16 numbers by the `Any` attrib. A named
  `[16]byte` type is logged as UUID only if it holds a valid or nil UUID.
- `Message.Any` writes a `*json.RawMessage` as JSON like a
  `json.RawMessage`, instead of as array of one value.
- `Message.Any` writes the most common types like strings, numbers,
  `time.Time`, UUIDs and errors without reflection, and checks named UUID
  types without allocating. `Message.Loggable` doesn't allocate for the
  checks that end a Loggable that logs itself, and the checks before
  marshalling or formatting a value don't copy the structs in it.

## [1.4.0] - 2026-11-24

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
- `BinaryEncoder` configures how binary data is encoded for logging,
  for example `golog.BinaryEncoder = hex.EncodeToString`. It is used by
  `Message.Bytes`, for bytes that are no valid UTF-8 in `Message.StrBytes`,
  `Message.StrBytesMax` and `Message.Any`, and by the `Any` attrib.
  The default is unchanged, base64.RawURLEncoding.
- `mempool.RaceEnabled` reports whether the binary was built with the race
  detector, which makes `sync.Pool` drop items instead of reusing them.
- **logsentry:** `NewLogfileWriteCallback` returns a `logfile.WriteCallback`
  that sends errors of a `logfile.RotatingWriter` as Sentry events,
  with the start of the unwritten log data and at most one event
  per passed interval, because a full disk makes every write fail.
- **logfile:** `FreeDiskSpace` returns the bytes that the process can still
  write to the volume of a file or directory, on Windows and Unix systems,
  for a `WriteCallback` that warns before the disk is full.
  The root module now requires `golang.org/x/sys` directly; it was already
  an indirect dependency via `golang.org/x/term`.
- **logsentry:** the `WithContextHub` option of `NewWriterConfig` sends the
  events of messages logged with a context that has a Sentry hub, like the
  per-request hub of `sentryhttp`, to that hub, so the events get the request
  data, user, tags and breadcrumbs of its scope. Without the option all events
  still go to the hub passed to `NewWriterConfig`.

### Changed

- A logger with attributes logs a message without allocating. `mempool.Slice`
  put slice values into its `sync.Pool`, which allocates to box the slice
  header, so every message of such a logger allocated 24 bytes when it gave
  its attribute slice back. The pool now stores pointers to slices and reuses
  them, so the comparative benchmark of a sub-logger with attributes went
  from 1 allocation and 287 ns to 0 allocations and about 250 ns per message.
- **Breaking:** `Message.SubLoggerContext` now also adds the attribs from the
  passed context to the returned sub-logger, so that the sub-logger and the
  returned context have the same union of attribs attached to them. Attribs
  recorded at the message still take precedence over context attribs with the
  same key. Previously the context attribs ended up only in the returned
  context, so the sub-logger lost them when used with a different context.
  Existing code that keeps such a sub-logger therefore logs attribs that it
  did not log before.
  The attribs taken from the context stay context attribs of the sub-logger
  instead of becoming logger attribs. They are logged after the attribs of the
  context that a message is logged with, so an attrib with the same key from
  that context takes precedence over the inherited one. A sub-logger created
  from a request scoped context can therefore be used with the context of
  another request without mislabeling its messages. Once logged, an inherited
  attrib blocks later attribs with the same key from `Message.Ctx` or a
  `Message` field method, like a logger attrib does, so no key is written twice.
- A message no longer writes a key twice when it repeats a key of the
  context it was logged with. `Message.Ctx` and the `Message` field methods
  now drop attribs whose key was already written from a context, as they
  already did for logger attribs. Previously
  `log.InfoCtx(ctx, "msg").Str("requestID", id)` with a `requestID` in `ctx`
  wrote the key twice, and so did `log.InfoCtx(ctx, "msg").Ctx(ctx)`.
  The message borrows the attribs of its context, so this allocates nothing.
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
  attribs because it has no end of life, and an attrib that another owner returns
  to the mempool would be recycled while the context still references it. Same
  contract for `ContextWithAttribs`, `RequestWithAttribs`, and
  `Attribs.AddToRequest`. Until v1.3.2 this only held when the receiver or the
  parent context had no attribs. With attribs on both sides the context shared
  the receiver's attribs, see `Attribs.CloneAndAppendNonExistingCloned` under
  Fixed.
- The thread safety guarantee in the package documentation and README
  now lists the rules it depends on. The `io.Writer` of a JSON or text writer
  config must be safe for concurrent use, package variables must be set before
  logging starts, and `Logger.RemoveAttribs` must not run concurrently with
  other uses of the logger.
- A `MessageCallback` gets every element of a slice in one attrib. A slice
  whose elements have different types, like a `[]any`, or that contains nil
  or JSON elements, is passed as one `Any` attrib holding a `[]any` of all
  elements in their order, instead of losing elements or passing them as
  several attribs with the same key. An empty slice is passed as an `Any`
  attrib with an empty `[]any` instead of a nil attrib that made the
  callback panic. Callbacks that switch on the attrib type now see the types
  of the logged values, for example `*Ints` instead of `*Strings` for
  `Message.Any(key, []any{1, 2})`, `*String` for bytes and `*Time` for a
  `time.Time`.

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
- A `time.Time` or `*time.Time` logged with `Message.Any` is now written like
  with `Message.Time`, using the `Format.TimeFormat` and `Format.Location` of
  the writer and null for the zero time. It used to be written by
  `time.Time.MarshalJSON`, as an RFC3339Nano string in its own location.
- Bytes have the same meaning everywhere except in `Message.Bytes`, which
  always logs binary data. They are a string if valid UTF-8, else base64
  encoded, like `Message.StrBytes` logs them. This applies to a `[]byte`,
  a named byte slice type or a pointer to one logged with `Message.Any`,
  also as element of a slice or as struct field with `Message.StructFields`,
  which used to write a slice of numbers, and to `Any.AppendJSON` and
  `Any.ValueString` of such values. A `json.RawMessage` logged with
  `Message.Any` is written as JSON like with `Message.JSON`.
  A byte slice type with a format of its own, like `net.IP`,
  `net.HardwareAddr` or a type that redacts a secret, is logged as one value
  with its `Loggable`, `Error`, `MarshalJSON`, `MarshalText` or `String`
  method instead of as bytes, also by the `Any` attrib. Like with
  encoding/json, methods with pointer receiver are used for a pointer
  to the type and for slice elements.
- `Message.StrBytesMax` now logs nil bytes as null like `Message.StrBytes`,
  instead of an empty string, and valid UTF-8 that contains the Unicode
  replacement character U+FFFD as string instead of base64 encoded.
- A `Loggable` logged with `Message.Any`, for example `Attribs` or a single
  attrib, is now written as a JSON object of the attribs it logs under the
  key, also as element of a slice. It used to produce invalid JSON: a key
  without a value, followed by the attribs once per writer.
  `Message.Loggable` still logs them as attribs of the message.
- Elements of a `[]any` logged with `Message.Any` are now written with their
  own type, for example numbers as numbers instead of strings, and a
  `time.Duration` as number of nanoseconds like a single `time.Duration`
  logged with `Message.Any`.
- The text writer now separates a JSON value from the previous element of
  a slice, for example `[{"a":1},2]` instead of `[{"a":1}2]`.
- `Any.AppendJSON` of a `time.Time` now uses the library-wide
  `DefaultTimeFormat` like the `Time` attrib, instead of `time.RFC3339`,
  which dropped the sub-seconds.
- `Attribs.AppendJSON` of empty Attribs now prepends a comma separator after
  a preceding JSON value, like it does for non-empty Attribs, instead of
  producing invalid JSON such as `[1null`.
- `Errors.ValueString` and `Errors.String` no longer panic for an `Errors`
  attrib that holds only nil errors, and return `<nil>` instead.
- `JSON.AppendJSON` and `Any.AppendJSON` with an empty `json.RawMessage` now
  append JSON null, as logging the attrib does, instead of a key without a
  value that made `Attribs.MarshalJSON` return invalid JSON.
- The tests pass under `go test -race`. They no longer expect `sync.Pool` to
  hand out an item again that was put back, which the race detector prevents
  at random.
- **goslog:** handlers derived from the same parent with `WithAttrs`
  (for example `base.With("request", "A")` and `base.With("request", "B")`)
  no longer write their attrs into the shared spare capacity of the parent's
  attrs, which made one logger log the attrs of the other and was a data race
  when per-request loggers were created concurrently.
- `HTTPMiddlewareRespondPlaintextCtxLogsIfNotOK` collects the logs in a
  buffer guarded by a mutex, so a wrapped handler can log with the request
  context from multiple goroutines without a data race.
- `DynDerivedConfig.SetParent` combines writer configs set with
  `SetAdditionalWriterConfigs` with the writer configs of the new parent
  instead of keeping those of the old parent.
- `Registry.PackagesSortedByName` returns the package paths
  instead of empty strings.
- **logsentry:** `NewWriterConfig` copies the passed `extra` map, so that
  changing it later does not race with logging.
- The `Any` attrib writes its value like `Message.Any` logs it, in
  `AppendJSON`, `Attribs.MarshalJSON` and `ValueString`: an error as its
  message instead of `{}`, a nil UUID and a zero time as null, a pretty
  printed `json.RawMessage` compacted to one line, invalid JSON as string,
  a `Loggable` as JSON object of its attribs, and every element of a `[]any`
  by the same rules.
- `Message.Any` of a nil pointer to a type whose `Error` or `IsNull` method
  has a value receiver logs null instead of panicking.
- The `UUID` and `UUIDs` attribs write a nil UUID as null in `AppendJSON`
  like `Message.UUID` and `Message.UUIDs` log it, so a `Loggable` logged
  as JSON object with `Message.Any` writes the same value.

### Documentation

- `AttribsFromContext`, `AttribFromContext`, `Logger.Attribs` and
  `Logger.CtxAttribs` document that the returned attribs are owned by the
  context or logger they come from. Don't `Add` to them and don't `Free`
  them, because both return attribs and their slice to the mempool while the
  context or logger still references them. Use `Attribs.Clone` for an owned
  copy.

## [1.3.2] - 2026-11-24

### Added

- **logfile:** `RotatingWriter.SetWriteCallback` registers a `WriteCallback`
  that is called after every `Write`, including writes that rotate the file,
  with the log file path, the data passed to `Write`, and the byte count and
  error returned by `Write`. Use it to report log write errors by other means,
  retry or redirect the unwritten part `data[n:]` of failed writes, or check
  the free disk space after writes. A panic in the callback
  is recovered and passed to `golog.ErrorHandler`.
- **logfile:** `NewRotatingWriterWithCallback` creates a `RotatingWriter`
  with a `WriteCallback` that is set before the first write.
- **logfile:** `RotateError` is returned by `RotatingWriter.Write` when
  rotating the log file fails, wrapping the error of closing or renaming it.
  Detect it with `errors.As`, also in a `WriteCallback`, to tell a failed
  rotation apart from a failed write.

### Changed

- Updated dependencies: `golang.org/x/term` v0.48.0, `golang.org/x/sys`
  v0.50.0 and `go.yaml.in/yaml/v3` v3.0.6, and for **logsentry**
  `golang.org/x/text` v0.44.0.
- `mempool.Slice.GetOrMake` now returns a slice with at least the requested
  capacity. It used to reuse a pooled slice that only fit the requested
  length, so appending up to the requested capacity could allocate a new,
  non-pooled backing array. `Attribs.Add` and
  `Attribs.CloneAndAppendNonExistingCloned` depend on this guarantee.
- **logsentry:** the `Writer` instances and their key-value maps are pooled
  with `mempool.Pointer` and `mempool.Map` instead of hand-rolled
  `sync.Pool`s, so the `mempool` test callbacks also track them.

### Fixed

- **logfile:** a `RotatingWriter` whose log file could not be renamed
  dropped every write after the file reached the rotation size, because each
  write retried the rotation, failed again and skipped the data. The data is
  now written to the current file, which grows past the rotation size, and
  the rotation is retried at most once per minute.
- **logfile:** a `RotatingWriter` no longer stops working for good when
  closing the file or opening it again during a rotation fails. The next
  write opens the log file again, and `Close` keeps later writes from
  reopening it.
- `mempool.Slice.GetOrMake` no longer panics when it allocates a new slice
  for a capacity smaller than the length while `MinCap` is zero.

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

[Unreleased]: https://github.com/domonda/golog/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/domonda/golog/compare/v1.4.1...v1.5.0
[1.4.1]: https://github.com/domonda/golog/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/domonda/golog/compare/v1.3.2...v1.4.0
[1.3.2]: https://github.com/domonda/golog/compare/v1.3.1...v1.3.2
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
