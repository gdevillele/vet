# Swift Implementation

This directory contains the Swift-native `vet` runner.

Usage:

```sh
swift run vet path/to/project
```

With config:

```sh
swift run vet --config ../../spec/config/v1.example.yaml path/to/project
swift run vet -c ../../spec/config/v1.example.yaml path/to/project
```

## Supported rules

| Rule   | Name                         | Status        |
|--------|------------------------------|---------------|
| VET002 | source-file-header-required  | implemented   |
| VET003 | source-file-header-min-length| implemented   |
| VET004 | source-file-header-max-length| implemented   |
| VET005 | max-source-file-lines        | implemented   |
| VET008 | source-format                | implemented   |
| VET014 | github-actions-pinned        | implemented   |
| VET015 | no-comments                  | implemented   |
| VET016 | forbidden-files              | implemented   |

## Unimplemented structural rules

These rules are **not supported** for Swift and are **disabled by default**.
SwiftParser/SwiftSyntax is currently used only for comment detection. The shared YAML schema
still accepts their keys for multi-language configs. Explicit CLI flags for them
exit with an error (`not supported for Swift`). Non-default values from a config
file produce a warning and are not enforced:

- `VET001` max-function-parameters
- `VET006` max-function-body-lines
- `VET007` function-docstring-policy
- `VET010`–`VET013` identifier casing

## Format check (VET008)

Format enforcement delegates to the industry tool `swift-format`:

```sh
swift-format lint --strict
```

Config:

```yaml
rules:
  format:
    enabled: true   # default
```

CLI:

```sh
swift run vet --check-format path/to/project
swift run vet --check-format=false path/to/project
```

If `swift-format` is missing from `PATH`, the runner fails with exit code 2 and
a clear error (it does not silently skip format checks when enabled).

## Notes

- Indentation-only rules (`indent.type` / `indent.width` / VET009) are removed
  from the product; use `format.enabled` instead.
- Header and line-count rules use simple, robust scanning without a full AST.

## Comment prohibition (VET015)

Use `rules.no-comments: { enabled: true }` to forbid all comments. If
`source-file-header.required: true`, the required header is implicitly allowed;
all other comments remain forbidden. Without a required header, even header
comments are forbidden. CLI overrides are `--no-comments[=false]` and
`--require-file-header[=false]`.
SwiftPM resolves SwiftParser/SwiftSyntax;
comment detection handles nested comments, raw strings, regex literals, and
comments inside string interpolations.

## Forbidden files (VET016)

Use top-level `rules.forbidden-files` with `enabled`, `patterns`, `exclude`, and
an optional `reason` to fail on any file whose path matches a pattern, such as
`"**/*.py"`. The rule is repo-wide: with no paths it walks the whole tree from the
working directory, otherwise only the given paths, always skipping `.git`.
Setting it under `languages.<lang>.rules` is a config error. CLI override is
`--forbidden-files[=false]`; patterns come only from the config file.
