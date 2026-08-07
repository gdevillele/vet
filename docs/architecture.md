# Architecture

## Decision

`vet` should have one native implementation per target ecosystem, not one
implementation that requires every user to install one runtime.

For example:

- Go projects should be able to run `go run`.
- Swift projects should be able to run `swift run`.
- Rust projects should be able to run `cargo run` or install a Cargo binary.
- TypeScript projects should be able to run `npm run vet` / `npx vet` from the
  TypeScript runner package.
- C/C++ projects run a dedicated runner that analyzes C/C++ sources. That runner
  is hosted in Go rather than written in C/C++ (see below).

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

- Go: `go/parser`, `go/ast`, and later `go/packages`; formatting via `go/format`
  (gofmt semantics);
- Swift: line/comment analysis for headers and file length; formatting via
  `swift-format`. Structural rules (parameters, body lines, docstrings, casing)
  are unimplemented without a dependable SwiftSyntax-based analyzer;
- Rust: `syn` for structural rules; formatting via `rustfmt`;
- TypeScript: TypeScript compiler API for structural rules; formatting via
  Prettier (`prettier.format`);
- C/C++: comment/line analysis for headers and file length; formatting via
  `clang-format`. Function-shape and casing rules are not supported for C/C++.

### Why the C/C++ Runner Is Not Written in C/C++

Go, Swift, and Rust each have a standard "run this package" entry point that
makes a same-language runner low-friction. C/C++ does not: projects use many
build systems, and shipping a C++-native tool forces every consumer through a
toolchain install that is often heavier than installing a static binary.

Writing C/C++ analysis in C/C++ also has a higher implementation risk for a
quality gate. Preprocessor macros, templates, and ambiguous declarations make
hand-rolled parsing brittle. Hosting the C/C++ runner in Go keeps the tool
memory-safe, easy to distribute, and aligned with the existing Go runner while
still analyzing C and C++ sources under a **subset** of the shared rule contract
(line/comment/format/CI rules only). Language-aware function-shape and casing
rules remain unsupported for C/C++ rather than scheduled as future work.

## Avoiding Rule Drift

The main risk of per-language implementations is divergence. `VET001` must mean
the same thing everywhere even if Go, Swift, Rust, and C/C++ have different
syntax.

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

All remaining rules are compatible with Go, Rust, Swift, TypeScript, and C/C++
(`cpp`). Implementation coverage differs:

- **Go, Rust, and TypeScript** implement structural rules (`VET001`, `VET006`,
  `VET007`, `VET010`–`VET013`) with standard parsers (`go/parser`, `syn`,
  TypeScript compiler API), plus headers, file length, format, and GitHub
  Actions pinning.
- **Swift and C/C++** are **subset** runners: headers, file length, standard
  formatters (`swift-format` / `clang-format`), and GitHub Actions pinning.
  Function-shape and casing rules are `implementation: unimplemented` (not a
  roadmap promise). See [rule-review.md](rule-review.md).

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

Native runners collect and sort the complete diagnostic set. Default text
output renders only the first sorted diagnostic so agent feedback stays short
and actionable. JSON output remains the complete machine-readable diagnostic
payload.

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

`VET008` (`source-format`) requires sources to match the language standard
formatter when enabled (default on). Enforcement delegates to industry tools:
`go/format` (gofmt) for Go, `rustfmt` (edition 2021, compare formatted output to
input) for Rust, `swift-format lint --strict` for Swift, Prettier for
TypeScript, and `clang-format --dry-run --Werror` for C/C++. Missing external
formatters fail the run clearly rather than skipping the check.

`VET009` (`indent-width`) was **removed**; indent width/style is owned by the
standard formatters under `VET008`.

`VET010` through `VET013` enforce casing for functions, variables, types, and
constants on Go and Rust only. The grouped `casing` config is disabled by
default, and each kind defaults to `language-default`. Go's language default
follows export visibility: exported identifiers use `UpperCamelCase`, while
unexported identifiers use `camelCase`. Rust's language default uses
`snake_case` for functions and variables, `UpperCamelCase` for types, and
`SNAKE_CASE_FULL_CAPS` for constants.

`VET014` enforces pinned GitHub Actions step references when explicitly enabled.
It scans workflow files under `.github/workflows/*.yml` and
`.github/workflows/*.yaml` by default, and checks only
`jobs.<job>.steps[*].uses`. External actions must use a full 40-character
hexadecimal commit SHA after `@`. Local `./...` actions, Docker `docker://...`
actions, and job-level reusable workflow calls are outside the first-version
scope.
