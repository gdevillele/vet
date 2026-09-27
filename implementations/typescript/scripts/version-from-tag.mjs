#!/usr/bin/env node
/**
 * Map a git release tag to an npm package version and optionally write it
 * into package.json.
 *
 * Examples:
 *   v0.1.0              -> 0.1.0
 *   refs/tags/v1.2.3    -> 1.2.3
 *   v1.0.0-alpha.1      -> 1.0.0-alpha.1
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

/** Loose semver: major.minor.patch with optional prerelease or build metadata. */
const SEMVER_RE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

/**
 * @param {string} ref - Git tag name or refs/tags/... form
 * @returns {string} npm package version (no leading "v")
 */
export function versionFromTag(ref) {
  let tag = String(ref ?? "").trim();
  if (tag.length === 0) {
    throw new Error("release tag is empty");
  }
  if (tag.startsWith("refs/tags/")) {
    tag = tag.slice("refs/tags/".length);
  }
  if (!tag.startsWith("v")) {
    throw new Error(`release tag must start with 'v', got: ${JSON.stringify(ref)}`);
  }
  const version = tag.slice(1);
  if (!SEMVER_RE.test(version)) {
    throw new Error(
      `invalid semver after stripping leading 'v': ${JSON.stringify(version)}`,
    );
  }
  return version;
}

/**
 * @param {string} packageJsonPath
 * @param {string} version
 * @returns {{ previous: string, version: string, path: string }}
 */
export function applyVersionToPackageJson(packageJsonPath, version) {
  const raw = fs.readFileSync(packageJsonPath, "utf8");
  const pkg = JSON.parse(raw);
  const previous = pkg.version;
  pkg.version = version;
  // Preserve trailing newline if present.
  const body = `${JSON.stringify(pkg, null, 2)}\n`;
  fs.writeFileSync(packageJsonPath, body, "utf8");
  return { previous, version, path: packageJsonPath };
}

function printUsage(stream = process.stderr) {
  stream.write(
    "Usage: node scripts/version-from-tag.mjs [--write] [--package-json <path>] <tag>\n" +
      "  Prints the npm version derived from a release tag.\n" +
      "  With --write, also updates package.json (default: this package root).\n",
  );
}

function main(argv) {
  const args = argv.slice(2);
  let write = false;
  /** @type {string | undefined} */
  let packageJsonPath;
  const positionals = [];
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--write" || arg === "-w") {
      write = true;
      continue;
    }
    if (arg === "--package-json") {
      const next = args[++i];
      if (!next) {
        process.stderr.write("--package-json requires a path\n");
        return 2;
      }
      packageJsonPath = next;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      printUsage(process.stdout);
      return 0;
    }
    if (arg.startsWith("-")) {
      process.stderr.write(`unknown flag: ${arg}\n`);
      printUsage();
      return 2;
    }
    positionals.push(arg);
  }

  if (positionals.length !== 1) {
    printUsage();
    return 2;
  }

  const version = versionFromTag(positionals[0]);
  if (write) {
    const here = path.dirname(fileURLToPath(import.meta.url));
    const target =
      packageJsonPath ?? path.join(here, "..", "package.json");
    const result = applyVersionToPackageJson(target, version);
    process.stdout.write(
      `set version ${result.previous} -> ${result.version} in ${result.path}\n`,
    );
  } else {
    process.stdout.write(`${version}\n`);
  }
  return 0;
}

const isMain =
  process.argv[1] &&
  path.resolve(fileURLToPath(import.meta.url)) === path.resolve(process.argv[1]);

if (isMain) {
  try {
    process.exit(main(process.argv));
  } catch (err) {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
  }
}
