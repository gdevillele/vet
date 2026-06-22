# Rust Implementation

This directory will contain the Rust-native `vet` runner.

Expected usage:

```sh
cargo run -- path/to/project
```

The Rust implementation should consume the shared rule contract in `../../spec`
and emit the same diagnostic shape as the other implementations.

Likely parser options:

- `syn` for standalone Rust source parsing;
- rust-analyzer syntax crates for richer project-aware analysis;
- compiler-integrated linting only if the rule set needs semantic type data.
