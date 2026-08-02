# C/C++ Implementation

This directory contains the C/C++ `vet` runner.

Unlike the Go, Rust, and Swift runners, this implementation is **written in
Go**, not C or C++. That is intentional: C/C++ projects do not have a single
standard package-manager entry point comparable to `go run`, `cargo run`, or
`swift run`, and reliable C/C++ parsing (preprocessor, macros, templates) is a
poor fit for a first-pass quality gate. A Go-hosted runner keeps the tool
memory-safe, easy to ship as a static binary, and consistent with the existing
Go implementation patterns.

Usage from the repository root:

```sh
go run ./implementations/cpp/cmd/vet path/to/project
```

From `implementations/cpp`:

```sh
go run ./cmd/vet path/to/project
```

Run the local test suite:

```sh
go test ./...
```

## Supported file types

`.c`, `.h`, `.cc`, `.cpp`, `.cxx`, `.c++`, `.hh`, `.hpp`, `.hxx`, `.h++`,
`.ipp`, `.tpp`, `.inl`

## Currently implemented rules

| Rule   | Name                         | Status      |
|--------|------------------------------|-------------|
| VET002 | source-file-header-required  | implemented |
| VET003 | source-file-header-min-length| implemented |
| VET004 | source-file-header-max-length| implemented |
| VET005 | max-source-file-lines        | implemented |
| VET008 | indent-type                  | implemented |
| VET009 | indent-width                 | implemented |
| VET014 | github-actions-pinned        | implemented |

Function-shape and casing rules (`VET001`, `VET006`, `VET007`, `VET010`–`VET013`)
are **not supported** for C/C++ and are **disabled by default**. The shared
spec marks them `implementation: unimplemented` (compatible but not scheduled
for this Go-hosted, line-based runner). Explicit CLI flags for those rules exit
with an error (`not supported for C/C++`). Non-default values from a config file
produce a stderr warning and are ignored. Enforcing them safely would need a
real C/C++ syntax model (macros, templates); that is intentionally out of scope
here.

## Language defaults

- indentation uses spaces (`language-default` resolves to spaces)
- header detection accepts leading `//` and `/* ... */` comments
- generated-code markers (`Code generated ... DO NOT EDIT.`) are not treated as
  file headers

The runner consumes the shared rule contract in `../../spec` and emits the same
diagnostic shape as the other implementations.
