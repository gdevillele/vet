# Architecture

## Decision

`vet` should have one native implementation per target ecosystem, not one
implementation that requires every user to install one runtime.

For example:

- Go projects should be able to run `go run`.
- Swift projects should be able to run `swift run`.
- Rust projects should be able to run `cargo run` or install a Cargo binary.

The shared layer is not a compiled library. It is the rule contract:

- stable rule IDs;
- default configuration;
- diagnostic shape;
- language-specific conformance fixtures;
- shared documentation for what each rule means.

## Why Not One Binary?

A single binary is easier for the tool author, but worse for adoption inside
language-specific projects. A Go-only repository should not need Rust installed
just to run its quality gate. The same applies to Swift and Rust projects.

Native runners also allow each implementation to use the best parser and build
integration for that ecosystem:

- Go: `go/parser`, `go/ast`, and later `go/packages`;
- Swift: SwiftSyntax and SwiftPM plugins;
- Rust: rust-analyzer syntax crates, `syn`, or compiler-integrated tooling.

## Avoiding Rule Drift

The main risk of per-language implementations is divergence. `VET001` must mean
the same thing everywhere even if Go, Swift, and Rust have different syntax.

To control that:

1. Every rule is defined in `spec/rules/v1.yaml`.
2. Every implementation emits the same diagnostic fields.
3. Every implementation should add conformance fixtures under `spec/conformance`.
4. Language-specific behavior must be documented next to the rule when exact
   equivalence is impossible.

## Implementation Boundary

Each language implementation owns:

- file discovery for that ecosystem;
- parsing;
- mapping syntax nodes to rule inputs;
- CLI packaging and installation;
- ecosystem-specific ignores and generated-file handling.

The shared spec owns:

- rule identity;
- default thresholds;
- severity;
- diagnostic vocabulary;
- conformance fixtures.

## Initial Rule

`VET001` enforces a maximum number of function parameters. The default maximum
is `1`.

The rule counts explicit function parameters. In Go, method receivers are not
counted as parameters for this rule.
