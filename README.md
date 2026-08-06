# vet

[![CI Status](https://github.com/gdevillele/vet/actions/workflows/vet.yml/badge.svg?branch=main)](https://github.com/gdevillele/vet/actions/workflows/vet.yml?query=branch%3Amain)

`vet` is a strict, multi-language code quality tool for agent-written code.

The project intentionally uses **per-language runners** instead of one
universal binary. A Go project should be able to run the Go vet runner with Go
tooling, a Swift project with Swift tooling, and a Rust project with Rust
tooling. TypeScript projects use the Node/TypeScript runner. C/C++ is analyzed
by a dedicated runner hosted in Go (see the C/C++ section below).

The shared part is the rule contract: rule IDs, defaults, language
compatibility, diagnostics, and conformance fixtures live in `spec/`. Each
language implementation translates its syntax into those shared rules.

## Repository Layout

```text
spec/                        Shared rule definitions and conformance fixtures.
implementations/go/          Go-native vet runner.
implementations/rust/        Rust-native vet runner.
implementations/swift/       Swift-native vet runner.
implementations/typescript/  TypeScript-native vet runner (Node).
implementations/cpp/         C/C++ vet runner (hosted in Go).
docs/                        Architecture notes.
```

## Current Go Runner

From the repository root:

```sh
go run ./implementations/go/cmd/vet ./implementations/go
```

From `implementations/go`:

```sh
go run ./cmd/vet ./...
```

From `implementations/swift`:

```sh
swift run vet ../../spec/conformance/source-format/swift
```

From `implementations/rust`:

```sh
cargo run -- ../../spec/conformance/max-function-parameters/rust
```

From `implementations/cpp` (C/C++ sources, runner written in Go):

```sh
go run ./cmd/vet ../../spec/conformance/source-format/cpp
```

From `implementations/typescript` (after `npm install`):

```sh
npm run vet -- ../../spec/conformance/max-function-parameters/typescript
```

The default enabled structural rule for the Go, Rust, and TypeScript runners is
`VET001`, which rejects functions with more than one parameter. Formatting
(`VET008`) is enabled by default and delegates to industry tools: `go/format`
(gofmt), `rustfmt`, `swift-format`, Prettier (TypeScript), and `clang-format`.
The Swift and C/C++ runners are **subset** runners: headers, file length,
standard format checks, and GitHub Actions pinning only. Function-shape and
casing rules are not supported on those runners (explicit CLI flags error with
“not supported”).

See [docs/rule-review.md](docs/rule-review.md) for the keep/replace/remove
decisions for every rule.

Header rules are available behind CLI flags:

```sh
go run ./implementations/go/cmd/vet \
  -require-file-header \
  -min-file-header-length 40 \
  -max-file-header-length 400 \
  ./implementations/go/...
```

`-min-file-header-length 0` and `-max-file-header-length 0` disable the
corresponding length bound. Length bounds apply to files that have headers;
combine them with `-require-file-header` to make missing headers fail too.

Projects can put rule settings in a YAML config file:

```yaml
version: 1
rules:
  max-function-parameters:
    enabled: true
    max: 1
  source-file-header:
    required: true
    min-length: 40
    max-length: 400
  max-source-file-lines:
    max: 300
  max-function-body-lines:
    max: 20
  function-docstring:
    policy: optional
  format:
    enabled: true
  casing:
    enabled: false
    functions: language-default
    variables: language-default
    types: language-default
    constants: language-default
    ignore-names: []
    ignore-patterns: []
  github-actions-pinned:
    enabled: false
languages:
  go:
    files:
      - implementations/go/...
    exclude:
      - "**/*_test.go"
    rules:
      format:
        enabled: true
  swift:
    files:
      - implementations/swift/Sources/...
    exclude:
      - "**/*Tests.swift"
    rules:
      format:
        enabled: true
  rust:
    files:
      - implementations/rust/src/...
    exclude:
      - "target/**"
    rules:
      format:
        enabled: true
  cpp:
    files:
      - src/...
      - include/...
    exclude:
      - "build/**"
    rules:
      format:
        enabled: true
```

When `-c` or `--config` is omitted, vet loads `vet.yaml` from the current
directory if it exists.

Run with explicit paths:

```sh
go run ./implementations/go/cmd/vet --config vet.yaml ./...
```

Explicit CLI flags override values from the config file.
Explicit CLI paths override `languages.<language>.files`.

Default text output prints only the first diagnostic after sorting by file,
line, column, and rule ID. This keeps agent feedback short and focused. Use
`--format json` when tooling needs the complete diagnostic list.

When no CLI paths are supplied, a native runner uses
`languages.<language>.files` as its input set. Entries may be files,
directories, recursive directories using the existing `...` suffix, or
shell-style glob patterns such as `cmd/tool/*.go`. `exclude` patterns are
matched against the collected files; basename patterns such as `*_test.go`,
recursive suffix patterns such as `**/*Tests.swift`, and directory prefixes
such as `vendor/**` are supported.

Top-level `rules` apply as global defaults. `languages.<language>.rules`
overrides those defaults for a native runner such as `go` or `swift`.

Additional strictness flags:

```sh
--max-source-file-lines 300
--max-function-body-lines 20
--function-docstring-policy optional
--check-format
--casing
--function-casing language-default
--variable-casing language-default
--type-casing language-default
--constant-casing language-default
--github-actions-pinned
```

The docstring policy accepts `forbidden`, `optional`, or `mandatory`.
`--check-format` / `format.enabled` requires sources to match the language
standard formatter (gofmt / rustfmt / swift-format / Prettier / clang-format).
Casing styles accept `off`, `language-default`, `camelCase`,
`UpperCamelCase`, `snake_case`, or `SNAKE_CASE_FULL_CAPS`. The casing rule is
disabled by default and is implemented for Go, Rust, and TypeScript.

`--github-actions-pinned` enables `VET014`, which scans GitHub workflow files
under `.github/workflows/*.yml` and `.github/workflows/*.yaml` by default.
Explicit workflow files or workflow directories can also be passed as paths.
The rule checks only `jobs.<job>.steps[*].uses`: external actions must use a
40-character hexadecimal commit SHA after `@`; local `./...` actions, Docker
`docker://...` actions, and job-level reusable workflow calls are ignored.

## Architecture Decision

Use per-language tools at the execution boundary, but keep one shared rule
specification.

That means:

- Go users can run `go run`.
- Swift users can run `swift run`.
- Rust users can run `cargo run` or install a Rust-native binary.
- C/C++ users run the C/C++ runner (Go-hosted binary or `go run`).
- Rule semantics do not drift between implementations.

The shared rule spec records both compatibility and implementation status for
Go, Rust, Swift, TypeScript, and C/C++ (`cpp`). Go, Rust, and TypeScript
implement the full structural rule set. Swift and C/C++ implement the safe
subset (headers, file length, standard format orchestration, GitHub Actions
pinning); function-shape and casing rules are `unimplemented` for those
runners, not scheduled as future work.

See [docs/architecture.md](docs/architecture.md) for the rationale and
implementation boundaries, including why the C/C++ runner is not written in
C/C++, and [docs/rule-review.md](docs/rule-review.md) for per-rule keep /
replace / remove decisions.
