import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { run } from "../src/cli.js";
import { prettierRunner } from "../src/format.js";

function capture() {
  let stdout = "";
  let stderr = "";
  return {
    get stdout() {
      return stdout;
    },
    get stderr() {
      return stderr;
    },
    invocation: {
      stdout: { write(chunk: string) {
        stdout += chunk;
      } },
      stderr: { write(chunk: string) {
        stderr += chunk;
      } },
    },
  };
}

describe("CLI entry", () => {
  it("reports VET001 on a negative fixture via run()", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-cli-"));
    const file = path.join(dir, "invalid.ts");
    fs.writeFileSync(
      file,
      `export function rejected(left: number, right: number) {}\n`,
      "utf8",
    );
    const cap = capture();
    const code = await run({
      args: ["--check-format=false", "--max-function-parameters", "1", file],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: dir,
    });
    assert.equal(code, 1);
    assert.match(cap.stdout, /VET001/);
  });

  it("exits 0 for a clean file with format disabled", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-cli-ok-"));
    const file = path.join(dir, "ok.ts");
    fs.writeFileSync(file, `export function ok(value: number) {\n  return value;\n}\n`, "utf8");
    const cap = capture();
    const code = await run({
      args: ["--check-format=false", "--max-function-parameters", "1", file],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: dir,
    });
    assert.equal(code, 0);
    assert.equal(cap.stdout, "");
  });

  it("reports VET008 when prettier would reformat the file", async () => {
    // Real prettier path: unformatted input must fail format check.
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-fmt-"));
    const file = path.join(dir, "ugly.ts");
    // Double spaces before brace / odd spacing Prettier fixes.
    const ugly = `export function ok(  )   {return 1}\n`;
    fs.writeFileSync(file, ugly, "utf8");

    // Sanity: real prettier changes it
    const pretty = await prettierRunner.format(ugly, file);
    assert.notEqual(pretty, ugly);

    const cap = capture();
    const code = await run({
      args: ["--check-format", "--max-function-parameters", "99", file],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: dir,
    });
    assert.equal(code, 1);
    assert.match(cap.stdout, /VET008/);
  });

  it("loads languages.typescript file selection from config", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-sel-"));
    const srcDir = path.join(dir, "src");
    fs.mkdirSync(srcDir);
    fs.writeFileSync(
      path.join(srcDir, "bad.ts"),
      `export function rejected(a: number, b: number) {}\n`,
      "utf8",
    );
    fs.writeFileSync(
      path.join(dir, "vet.yaml"),
      `version: 1
rules:
  format:
    enabled: false
  max-function-parameters:
    enabled: true
    max: 1
languages:
  typescript:
    files:
      - src
`,
      "utf8",
    );
    const cap = capture();
    const code = await run({
      args: [],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: dir,
    });
    assert.equal(code, 1);
    assert.match(cap.stdout, /VET001/);
  });

  it("finds nested .github/workflows under an explicit project directory (VET014)", async () => {
    // Real entry path: project dir with only unpinned workflow (no .ts required).
    // Discovery must look at <dir>/.github/workflows like Go/Rust, not only readdir(dir).
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-gha-dir-"));
    const workflows = path.join(dir, ".github", "workflows");
    fs.mkdirSync(workflows, { recursive: true });
    fs.writeFileSync(
      path.join(workflows, "ci.yml"),
      `name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`,
      "utf8",
    );
    // A valid TS file so source analysis is clean when format is off.
    fs.writeFileSync(
      path.join(dir, "ok.ts"),
      `export function ok(value: number) {\n  return value;\n}\n`,
      "utf8",
    );

    const cap = capture();
    const code = await run({
      args: [
        "--check-format=false",
        "--max-function-parameters",
        "99",
        "--github-actions-pinned",
        dir,
      ],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: os.tmpdir(), // not the project dir — discovery must use the explicit path
    });
    assert.equal(code, 1, `stderr=${cap.stderr} stdout=${cap.stdout}`);
    assert.match(cap.stdout, /VET014/);
    assert.match(cap.stdout, /actions\/checkout@v4/);
  });

  it("scans cwd/.github/workflows when no paths are passed and GHA is enabled", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-gha-cwd-"));
    const workflows = path.join(dir, ".github", "workflows");
    fs.mkdirSync(workflows, { recursive: true });
    fs.writeFileSync(
      path.join(workflows, "ci.yml"),
      `name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@main
`,
      "utf8",
    );
    fs.writeFileSync(
      path.join(dir, "ok.ts"),
      `export function ok(value: number) {\n  return value;\n}\n`,
      "utf8",
    );
    fs.writeFileSync(
      path.join(dir, "vet.yaml"),
      `version: 1
rules:
  format:
    enabled: false
  max-function-parameters:
    enabled: true
    max: 99
  github-actions-pinned:
    enabled: true
languages:
  typescript:
    files:
      - ok.ts
`,
      "utf8",
    );

    const cap = capture();
    const code = await run({
      args: [],
      stdout: cap.invocation.stdout,
      stderr: cap.invocation.stderr,
      cwd: dir,
    });
    assert.equal(code, 1, `stderr=${cap.stderr} stdout=${cap.stdout}`);
    assert.match(cap.stdout, /VET014/);
  });
});
