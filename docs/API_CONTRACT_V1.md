# API Contract v1 (Draft)

This document defines stable public API semantics for the `app`, `buildenv`, `builder`, and `stack` packages.

## Goals

- Stable behavior across minor releases.
- Explicit preconditions/postconditions for every exported API.
- Structured error model (no string-matching required).
- Backward-compatible migration path.

## Versioning and Compatibility

- This repository follows Go module major-version rules:
  - `v1.x.y` uses module path `github.com/gvallee/go_software_build`.
  - Any breaking API change requires a new major line with `/vN` suffix (for example `/v2`).
  - `v2+` requires importing `github.com/gvallee/go_software_build/v2` (and similarly for later majors).
- Within a major line, public APIs MUST preserve:
  - Function signatures.
  - Semantics defined in this document.
  - Sentinel error values and typed error wrappers.
- Within a major line:
  - New capabilities MUST be additive.
  - Behavior changes are breaking unless introduced via new methods/options.
- Consumers that need strict stability SHOULD pin within one major line (`v1.*` or `v2.*`).

## Error Model

Define package-level sentinel errors (wrapping allowed):

- `ErrInvalidArgument`
- `ErrInvalidState`
- `ErrNotFound`
- `ErrUnsupported`
- `ErrExternalTool`
- `ErrCommandFailed`
- `ErrAlreadyInstalled`

All returned errors SHOULD use `%w` and include enough context to diagnose path/tool/stage.

## Logging and Output Semantics

Library code MUST NOT write directly to stdout/stderr.

- No `fmt.Printf` in exported APIs.
- No unconditional `log.Printf` in exported APIs.
- Optional logging is provided through dependency injection:
  - `type Logger interface { Printf(string, ...any) }`
  - `nil` logger => silent.

## Context and Cancellation

Long-running APIs SHOULD have `Context` variants:

- `InstallContext(ctx)`
- `GetContext(ctx, app)`
- `InstallStackContext(ctx)`
- `ExportContext(ctx)` / `ImportContext(ctx)`

Legacy methods remain as wrappers using `context.Background()`.

## Determinism

APIs that generate files MUST be deterministic for equal inputs.

- Stable ordering for map-backed content generation.
- Stable naming and manifest conventions.

## Package Contracts

## `pkg/app`

### `type Info`
Contract:
- Data-only model object.
- No implicit mutation by consumers except fields explicitly documented by caller-owned workflows.

### `type SourceCode`
Contract:
- URL identifies source location and transport semantics.
- Branch fields only affect Git flows.

## `pkg/buildenv`

### `type Info`
Contract:
- Mutable execution context for source retrieval/build/install.
- `BuildDir`, `InstallDir`, and `ScratchDir` are caller-owned paths.

### `(*Info) Init() error`
Preconditions:
- `ScratchDir`, `BuildDir`, `InstallDir` are non-empty paths.
Postconditions:
- All three directories exist.
- Idempotent (safe to call repeatedly).
Errors:
- `ErrInvalidArgument` for empty required paths.
- Wrapped filesystem errors.

### `(*Info) Get(app *app.Info) error`
Preconditions:
- Non-nil app with non-empty source URL.
- Required destination dirs configured.
Postconditions:
- `SrcPath` and `SrcDir` updated to retrieved source location.
- No stdout/stderr writes from library.
Errors:
- `ErrInvalidArgument`, `ErrUnsupported`, `ErrExternalTool`, `ErrCommandFailed`.

### `(*Info) Unpack(app *app.Info) error`
Preconditions:
- `SrcPath`, `SrcDir`, `BuildDir` initialized.
Postconditions:
- Source unpacked when format supported.
- No-op for unsupported/single-file formats.
- `SrcDir` points to unpacked directory when applicable.
Errors:
- `ErrInvalidState`, `ErrUnsupported`, `ErrExternalTool`, `ErrCommandFailed`.

### `(*Info) RunMake(...) error`
Preconditions:
- Valid source/makefile directory.
Postconditions:
- Make command executed for stage.
Errors:
- `ErrInvalidArgument`, `ErrExternalTool`, `ErrCommandFailed`.

### `(*Info) Install(app *app.Info) error`
Preconditions:
- `SrcDir` set.
- `InstallCmd` empty means explicit no-op.
Postconditions:
- Installation command executed or no-op.
Errors:
- `ErrCommandFailed` (+ wrapped stderr/stdout context).

## `pkg/builder`

### `type Builder`
Contract:
- Orchestrates configure/build/install over a `buildenv.Info` and `app.Info`.

### `(*Builder) Load(persistent bool) error`
Preconditions:
- `App.Name`, `App.Source.URL`, `Env.ScratchDir`, `Env.BuildDir`, `Env.InstallDir` set.
Postconditions:
- Builder strategy initialized (configure function selected).
Errors:
- `ErrInvalidArgument` with field-specific context.

### `(*Builder) Install() advexec.Result`
Preconditions:
- `Load()` already called.
Postconditions:
- App installed in target install dir.
- Idempotent when app already installed.
Errors:
- `Result.Err` wraps typed sentinel and command context.

### `(*Builder) Compile() error`
Preconditions:
- App and env configured for local compile workflow.
Postconditions:
- Build/install attempted.
- `App.BinPath` resolved by deterministic precedence:
  1) `<install>/bin/<name>`
  2) `<install>/<name>`
  3) `<build>/<name>`
  4) `<src>/<name>`
Errors:
- `ErrInvalidArgument`, `ErrInvalidState`, `ErrCommandFailed`.

## `pkg/stack`

### `type Config`
Contract:
- Owns stack definition/config state and installation/export/module generation workflows.

### `(*Config) Load() error`
Preconditions:
- `DefFilePath`, `ConfigFilePath` are set.
Postconditions:
- `Data.StackDefinition`, `Data.StackConfig` set and validated.
- `Loaded=true`.
Errors:
- `ErrInvalidArgument`, `ErrCommandFailed` (file read/parse context).

### `(*Config) InstallStack() error`
Preconditions:
- Config loaded or loadable.
Postconditions:
- Components installed in dependency order.
- Component maps (`InstalledComponents`, `BuiltComponents`, optional `SrcComponents`) updated.
Errors:
- Wrapped sentinel errors with component/stage context.

### `(*Config) Export() error`
Preconditions:
- Config loaded.
- Stack base/install dirs exist.
Postconditions:
- `<stack>.tar.bz2` created under stack base dir.
Errors:
- `ErrInvalidState`, `ErrExternalTool`, `ErrCommandFailed`.

### `(*Config) Import(path string) error`
Preconditions:
- Config loaded.
- Tarball path exists and is readable.
Postconditions:
- Install tree extracted under stack base dir.
Errors:
- `ErrInvalidArgument`, `ErrExternalTool`, `ErrCommandFailed`.

### `(*Config) GenerateModules(copyright, prefix string) error`
Preconditions:
- Config loaded.
- Stack base dir exists.
Postconditions:
- Deterministic modulefiles generated under `<stack>/modulefiles`.
Errors:
- `ErrInvalidState`, `ErrCommandFailed`.

## Backward-Compatible Wrapper Plan

Phase 1 (additive, non-breaking):
- Add `Context` variants to long-running exported methods.
- Add optional logger injection fields/options.
- Keep legacy methods as wrappers.

Phase 2 (deprecation):
- Mark direct stdout/log side effects as deprecated behavior.
- Announce typed error guarantees as stable.

Phase 3 (next major only):
- Remove legacy wrappers only with major version bump and `/vN` module-path transition.

## Major Release Policy

- Breaking changes are intentionally batched and released as major lines (`v2`, `v3`, ...).
- Each new major MUST include:
  - A migration guide from previous major.
  - Explicit API delta/changelog.
  - Compatibility notes for import path updates.
- The previous major SHOULD continue to receive critical bug fixes for a defined deprecation window.

## Test Expectations for API Stability

- Contract tests for all preconditions/postconditions.
- Golden tests for generated files (modulefiles/manifests).
- Deterministic output tests for map-driven generation.
- Error classification tests (`errors.Is` / `errors.As`).
- Idempotency tests for `Init`, `Install`, `InstallStack` (where applicable).
