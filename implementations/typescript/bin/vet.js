#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const cliTs = path.join(here, "..", "src", "cli.ts");
const cliJs = path.join(here, "..", "dist", "cli.js");

// Prefer compiled dist when present; otherwise run TypeScript via tsx.
import fs from "node:fs";
if (fs.existsSync(cliJs)) {
  const result = spawnSync(process.execPath, [cliJs, ...process.argv.slice(2)], {
    stdio: "inherit",
  });
  process.exit(result.status ?? 1);
}

const result = spawnSync(
  process.execPath,
  ["--import", "tsx", cliTs, ...process.argv.slice(2)],
  { stdio: "inherit" },
);
process.exit(result.status ?? 1);
