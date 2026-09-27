# vet (TypeScript)

TypeScript-native runner for the shared [vet](../../README.md) rule contract.

**Package name:** `@vet/typescript`  
**CLI binary:** `vet`

## Install (consumers)

Add the published package as a dev dependency:

```sh
npm install -D @vet/typescript
```

Wire it into your project scripts:

```json
{
  "scripts": {
    "vet": "vet ."
  }
}
```

Then:

```sh
npm run vet
# or
npx vet .
```

Configuration: place a `vet.yaml` in the working directory (optional). See the
[root README](../../README.md) for rule keys and CLI flags.

## Run from this repository (development)

From this directory (after `npm install`):

```sh
npm run vet -- ../../spec/conformance/max-function-parameters/typescript
```

Or via the package binary (uses `dist/` when built, otherwise `tsx` + `src/`):

```sh
npm run build
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

## Publishing (maintainers)

Releases are published to the npm registry by GitHub Actions when a **version
tag** is pushed. Workflow:
[`.github/workflows/publish-typescript.yml`](../../.github/workflows/publish-typescript.yml).

The workflow tag filter is a **GitHub Actions filter glob**
(`v[0-9]*.[0-9]*.[0-9]*`), not a regular expression. Strict semver is enforced
in the job by `scripts/version-from-tag.mjs` before `npm publish`.

### Tag → version

| Git tag | npm `version` |
|---------|----------------|
| `v0.1.0` | `0.1.0` |
| `v1.2.3-alpha.1` | `1.2.3-alpha.1` |

The workflow runs `node scripts/version-from-tag.mjs --write <tag>` so the
published package version always matches the tag (with the leading `v` removed).

### One-time GitHub and npm setup

1. **npm package / scope**  
   Own the `@vet` organization (or change `name` in `package.json` to a scope
   you control). Create an **automation** or granular access token with
   **publish** permission for that package.

2. **GitHub Environment `production`**  
   In the repository settings → Environments → create **`production`**.  
   Add a secret named **`NPM_TOKEN`** with the npm token value.  
   Optional: add environment protection rules (required reviewers) so only
   intentional releases publish.

3. **Release**  
   On `main` (or the commit you want to ship):

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

   That starts the `publish-typescript` workflow, which installs, tests, builds,
   sets the version from the tag, and runs `npm publish` using
   `NODE_AUTH_TOKEN` from the `production` environment.

Live `npm publish` is not required for local validation; use
`npm publish --dry-run` or `npm pack` from this directory after `npm run build`.
