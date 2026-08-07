import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import {
  defaultConfig,
  loadConfigFile,
  validate,
} from "../src/config.js";

describe("config", () => {
  it("loads rule config and language overrides for typescript", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-config-"));
    const file = path.join(dir, "vet.yaml");
    fs.writeFileSync(
      file,
      `version: 1
rules:
  max-function-parameters:
    enabled: true
    max: 3
  format:
    enabled: false
  github-actions-pinned:
    enabled: true
languages:
  typescript:
    files:
      - src/**
    exclude:
      - "**/*.test.ts"
    rules:
      max-function-parameters:
        max: 2
      format:
        enabled: true
`,
      "utf8",
    );

    const cfg = loadConfigFile({
      path: file,
      base: defaultConfig(),
      language: "typescript",
    });
    assert.equal(cfg.maxFunctionParameters.max, 2);
    assert.equal(cfg.format.enabled, true);
    assert.equal(cfg.githubActionsPinned.enabled, true);
    assert.deepEqual(cfg.fileSelection.files, ["src/**"]);
    assert.deepEqual(cfg.fileSelection.exclude, ["**/*.test.ts"]);
  });

  it("rejects invalid docstring policy", () => {
    const cfg = defaultConfig();
    // @ts-expect-error intentional invalid
    cfg.functionDocstring.policy = "sometimes";
    assert.throws(() => validate(cfg), /function-docstring.policy/);
  });
});
