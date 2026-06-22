# vet

`vet` is a strict, multi-language code quality tool for agent-written code.

The project intentionally uses **per-language native implementations** instead
of one universal binary. A Go project should be able to run the Go vet runner
with Go tooling, a Swift project with Swift tooling, and a Rust project with
Rust tooling.

The shared part is the rule contract: rule IDs, defaults, diagnostics, and
conformance fixtures live in `spec/`. Each language implementation translates
its syntax into those shared rules.

## Repository Layout

```text
spec/                   Shared rule definitions and conformance fixtures.
implementations/go/     Go-native vet runner.
implementations/rust/   Rust-native vet runner scaffold.
implementations/swift/  Swift-native vet runner scaffold.
docs/                   Architecture notes.
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

The default enabled rule is `VET001`, which rejects functions with more than one
parameter.

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
```

Run with:

```sh
go run ./implementations/go/cmd/vet --config .vet.yaml ./...
```

Explicit CLI flags override values from the config file.

## Architecture Decision

Use per-language tools at the execution boundary, but keep one shared rule
specification.

That means:

- Go users can run `go run`.
- Swift users can run `swift run`.
- Rust users can run `cargo run` or install a Rust-native binary.
- Rule semantics do not drift between implementations.

See [docs/architecture.md](docs/architecture.md) for the rationale and planned
implementation boundaries.
