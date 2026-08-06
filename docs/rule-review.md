# VET rule review (keep / replace / remove)

Review against the project goal: **one multi-language entry point and config**,
prefer **industry-standard tools** where they own the check, keep only **simple
and robust** custom rules, and drop fragile or exotic enforcement.

Decision tests applied to every rule:

1. **Semantics** — simple and clear, not exotic or overly custom.
2. **Industry tool** — if the custom path reimplements a standard tool’s primary
   job, **replace** with orchestration of that tool.
3. **Robustness** — keep only dependable implementations (standard parsers such
   as `go/parser` / `syn`, or trivial line/comment analysis). Hand-rolled
   structural analysis must not be the sole gate.

Languages: Go, Rust, Swift, TypeScript, C/C++ (`cpp`).

| Rule | Name | Decision | Languages after review | Rationale |
|------|------|----------|------------------------|-----------|
| **VET001** | `max-function-parameters` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Simple limit. Go/`go/parser`, Rust/`syn`, and TypeScript compiler API are dependable. Swift used a hand-rolled lexer — not robust enough to keep. Not a primary job of gofmt/rustfmt/clang-format/swift-format/Prettier. Default `max: 1` is intentionally strict for agent-written code (product default), not exotic semantics. |
| **VET002** | `source-file-header-required` | **keep** custom | Go, Rust, Swift, TypeScript, cpp | Simple opt-in header presence. Line/comment analysis is enough; no standard formatter owns license/file headers. |
| **VET003** | `source-file-header-min-length` | **keep** custom | Go, Rust, Swift, TypeScript, cpp | Simple length bound on the same header model as VET002. |
| **VET004** | `source-file-header-max-length` | **keep** custom | Go, Rust, Swift, TypeScript, cpp | Same as VET003. |
| **VET005** | `max-source-file-lines` | **keep** custom | Go, Rust, Swift, TypeScript, cpp | Trivial physical line count; clear and dependable. |
| **VET006** | `max-function-body-lines` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Simple size limit with standard parsers. Swift hand-rolled body counting removed. |
| **VET007** | `function-docstring-policy` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Clear policies (`forbidden` / `optional` / `mandatory`). Robust on real ASTs; Swift hand-rolled path removed. |
| **VET008** | `source-format` | **replace** with industry tools | Go, Rust, Swift, TypeScript, cpp | Custom leading-whitespace indent scanning duplicated what formatters already own. Enforcement now delegates: Go → `go/format` (gofmt semantics), Rust → `rustfmt` (edition 2021; formatted output must match input), Swift → `swift-format lint --strict`, TypeScript → Prettier, C/C++ → `clang-format --dry-run --Werror`. Missing tools fail clearly (not silent skip). |
| **VET009** | `indent-width` | **remove** | — | Pure formatter concern (indent width/style). Subsumed by VET008 standard-format checks. Custom space-multiple scanning deleted. |
| **VET010** | `function-casing` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Optional naming style is not the primary job of formatters. Dependable with real parsers only; Swift hand-rolled casing removed. |
| **VET011** | `variable-casing` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Same as VET010. |
| **VET012** | `type-casing` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Same as VET010. |
| **VET013** | `constant-casing` | **keep** custom | Go, Rust, TypeScript implemented; Swift + cpp **unimplemented** | Same as VET010. |
| **VET014** | `github-actions-pinned` | **keep** custom | Go, Rust, Swift, TypeScript, cpp | Simple, security-relevant pin check on `jobs.*.steps[].uses`. Not a formatter concern; YAML-based check is adequate. Full actionlint suites are out of scope. |

## Summary counts

- **Keep custom:** VET001–VET007, VET010–VET014 (language support as above).
- **Replace with standard tools:** VET008 (`source-format`).
- **Remove:** VET009 (`indent-width`).

## Config surface after review

```yaml
rules:
  max-function-parameters: { enabled: true, max: 1 }
  source-file-header: { required: false, min-length: 0, max-length: 0 }
  max-source-file-lines: { max: 0 }
  max-function-body-lines: { max: 0 }
  function-docstring: { policy: optional }
  format: { enabled: true }          # was indent: { type, width }
  casing: { enabled: false, ... }
  github-actions-pinned: { enabled: false }
```

Removed: `indent.type`, `indent.width`, and CLI `--indent-type` / `--indent-width`.  
Added: `format.enabled` and CLI `--check-format`.

## Swift subset note

Without SwiftSyntax, structural rules (parameters, body lines, docstrings, casing)
cannot be enforced dependably. The Swift runner matches the C/C++ subset pattern
for those rules: `implementation: unimplemented`, no silent pass when the user
explicitly requests them on the CLI.
