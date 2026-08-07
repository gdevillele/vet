# Rust Implementation

This directory contains the Rust-native `vet` runner.

Usage:

```sh
cargo run -- path/to/project
```

Run the local test suite:

```sh
cargo test
```

The Rust runner consumes the shared rule contract in `../../spec`, emits the
same diagnostic shape as the Go and Swift implementations, and enforces the
same CLI/config behavior for Rust source files.

Rust language defaults:

- source format is checked with `rustfmt` (edition 2021; formatted output must match input; VET008; requires `rustfmt` on `PATH`);
- functions and variables use `snake_case`;
- types use `UpperCamelCase`;
- constants use `SNAKE_CASE_FULL_CAPS`.
