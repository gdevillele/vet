#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const cliJs = path.join(here, "..", "dist", "cli.js");
const cliTs = path.join(here, "..", "src", "cli.ts");

// Prefer compiled dist (required for published installs). Local monorepo
// development may fall back to TypeScript sources via tsx when dist is absent.
if (fs.existsSync(cliJs)) {
  const result = spawnSync(process.execPath, [cliJs, ...process.argv.slice(2)], {
    stdio: "inherit",
  });
  process.exit(result.status ?? 1);
}

if (fs.existsSync(cliTs)) {
  const result = spawnSync(
    process.execPath,
    ["--import", "tsx", cliTs, ...process.argv.slice(2)],
    { stdio: "inherit" },
  );
  process.exit(result.status ?? 1);
}

console.error(
  "vet: compiled CLI not found (dist/cli.js). " +
    "If you installed from npm, reinstall the package. " +
    "For local development, run: npm run build",
);
process.exit(1);
