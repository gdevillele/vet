import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { run } from "../src/cli.js";
import { defaultConfig, loadConfigFile } from "../src/config.js";

function writeConfig(yaml: string): { dir: string; file: string } {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-reason-"));
  const file = path.join(dir, "vet.yaml");
  fs.writeFileSync(file, yaml, "utf8");
  return { dir, file };
}

describe("rule reason", () => {
  it("loads reasons and lets a language block clear one", () => {
    const { file } = writeConfig(`version: 1
rules:
  max-function-parameters:
    reason: group related values
  casing:
    reason: match the team style guide
languages:
  typescript:
    rules:
      casing:
        reason: ""
`);

    const cfg = loadConfigFile({
      path: file,
      base: defaultConfig(),
      language: "typescript",
    });
    assert.equal(cfg.maxFunctionParameters.reason, "group related values");
    assert.equal(cfg.casing.reason, "");
  });

  it("rejects a non-string reason", () => {
    const { file } = writeConfig(`version: 1
rules:
  format:
    reason: [not, text]
`);

    assert.throws(
      () => loadConfigFile({ path: file, base: defaultConfig(), language: "" }),
      /format.reason must be a string/,
    );
  });

  it("appends the reason to the default diagnostic message", async () => {
    const { dir, file } = writeConfig(`version: 1
rules:
  max-function-parameters:
    reason: " wrap related values in an object "
  format:
    enabled: false
`);
    const source = path.join(dir, "sample.ts");
    fs.writeFileSync(
      source,
      "export function rejected(left: number, right: number) {}\n",
      "utf8",
    );
    let stdout = "";
    const code = await run({
      args: ["--config", file, "--format", "json", source],
      stdout: { write: (chunk: string) => void (stdout += chunk) },
      stderr: { write: () => undefined },
      cwd: dir,
    });

    assert.equal(code, 1);
    assert.deepEqual(
      JSON.parse(stdout).diagnostics.map(
        (item: { message: string }) => item.message,
      ),
      [
        "rejected has 2 parameters; maximum allowed is 1 (reason: wrap related values in an object)",
      ],
    );
  });
});
