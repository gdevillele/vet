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
- language compatibility and implementation status;
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
3. Every rule declares compatibility for every language listed by the spec.
4. Every implementation should add conformance fixtures under `spec/conformance`.
5. Language-specific behavior must be documented next to the rule when exact
   equivalence is impossible.

## Language Compatibility

Each rule has a `language_compatibility` map keyed by language. Compatibility
status answers whether the rule is meaningful for that language; implementation
status answers whether the native runner currently enforces it.

Compatibility statuses:

- `compatible`: the rule applies to the language.
- `incompatible`: the rule does not apply to the language and must include a
  reason.

Implementation statuses:

- `implemented`: the language runner enforces the rule.
- `planned`: the rule is compatible, but implementation work is still planned.
- `unimplemented`: the rule is compatible, but no implementation is scheduled.
- `not-applicable`: the rule is incompatible with the language.

All current rules are compatible with Go, Rust, and Swift. Go and Swift
currently implement them; Rust is compatible but still planned.

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
- language compatibility and implementation status;
- config schema examples;
- severity;
- diagnostic vocabulary;
- conformance fixtures.

## Initial Rules

`VET001` enforces a maximum number of function parameters. The default maximum
is `1`.

The rule counts explicit function parameters. In Go, method receivers are not
counted as parameters for this rule.

`VET002` requires source files to have a leading file header when enabled.

`VET003` and `VET004` enforce minimum and maximum header lengths. Header length
is counted after stripping comment delimiters and surrounding whitespace. A
value of `0` disables the corresponding length bound.

`VET005` enforces a maximum number of physical lines in a source file.

`VET006` enforces a maximum number of physical lines inside a function body,
excluding the opening and closing brace lines.

`VET007` enforces function docstring policy. Supported policies are
`forbidden`, `optional`, and `mandatory`.

`VET008` enforces indentation type. Supported types are `tabs`, `spaces`, and
`language-default`; the language default is tabs for Go and spaces for Swift
and Rust.

`VET009` enforces space indentation width when the effective indentation type is
spaces. A width of `0` disables the width check.

`VET010` through `VET013` enforce casing for functions, variables, types, and
constants. The grouped `casing` config is disabled by default, and each kind
defaults to `language-default` to avoid changing existing projects. Go's
language default follows export visibility: exported identifiers use
`UpperCamelCase`, while unexported identifiers use `camelCase`. Swift's
language default uses `camelCase` for functions, variables, and constants, and
`UpperCamelCase` for types.
