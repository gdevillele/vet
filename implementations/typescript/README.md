# vet (TypeScript)

TypeScript-native runner for the shared [vet](../../README.md) rule contract.

## Run

From this directory (after `npm install`):

```sh
npm run vet -- ../../spec/conformance/max-function-parameters/typescript
```

Or via the package binary:

```sh
npx --no-install vet ../../spec/conformance/source-format/typescript
```

From the repository root:

```sh
npm --prefix implementations/typescript install
npm --prefix implementations/typescript run vet -- ./spec/conformance/max-function-parameters/typescript
```

## Implemented rules

Uses the **TypeScript compiler API** for structural analysis and **Prettier** for
`VET008` (`source-format`). All remaining contract rules are implemented:

| Rule   | Notes |
|--------|--------|
| VET001 | max function parameters (`typescript` AST) |
| VET002–VET004 | file headers |
| VET005 | max source file lines |
| VET006 | max function body lines |
| VET007 | function docstring / JSDoc policy |
| VET008 | Prettier format check (`prettier.format`) |
| VET010–VET013 | identifier casing (opt-in) |
| VET014 | GitHub Actions pin check |

Language defaults for casing: `camelCase` functions/variables/constants,
`UpperCamelCase` types.

Missing Prettier (broken install) fails with a clear error rather than skipping
format checks.

## Config

Loads `vet.yaml` from the working directory when present, including
`languages.typescript` file selection and rule overrides. Shared keys match
other runners (`format.enabled`, `max-function-parameters`, …).
