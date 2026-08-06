import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Analyzer } from "../src/analysis.js";
import { defaultConfig } from "../src/config.js";
import type { FormatRunner } from "../src/format.js";

const identityFormat: FormatRunner = {
  async format(source: string) {
    return source;
  },
};

describe("Analyzer structural rules", () => {
  it("reports functions with too many parameters (VET001)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export function accepted(value: number) {}\nexport function rejected(left: number, right: number) {}\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET001");
    assert.match(diagnostics[0]!.message, /rejected has 2 parameters/);
  });

  it("honors disabled max-function-parameters", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.maxFunctionParameters.enabled = false;
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export function accepted(left: number, right: number) {}\n`,
    });
    assert.equal(diagnostics.length, 0);
  });

  it("reports missing required header (VET002)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.sourceFileHeader.required = true;
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export const x = 1;\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET002");
  });

  it("reports source file above maximum lines (VET005)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.sourceFileLines.max = 2;
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `// one\n// two\n// three\nexport const x = 1;\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET005");
  });

  it("reports function body line overflow (VET006)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.functionBodyLines.max = 1;
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export function rejected() {\n  const a = 1;\n  const b = 2;\n}\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET006");
  });

  it("reports missing mandatory docstring (VET007)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.functionDocstring.policy = "mandatory";
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export function missing() {}\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET007");
  });

  it("reports casing violations when enabled (VET010)", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.casing.enabled = true;
    cfg.casing.functions = "camelCase";
    const diagnostics = await new Analyzer(cfg).analyzeFile({
      path: "sample.ts",
      source: `export function Bad_Name() {}\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET010");
  });
});

describe("Analyzer format (VET008)", () => {
  it("reports unformatted source via format runner", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = true;
    cfg.maxFunctionParameters.enabled = false;
    const runner: FormatRunner = {
      async format(source: string) {
        return source.replaceAll("  ", " ");
      },
    };
    const diagnostics = await new Analyzer(cfg, runner).analyzeFile({
      path: "sample.ts",
      source: `export function ok()  {\n  return 1;\n}\n`,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET008");
    assert.match(diagnostics[0]!.message, /prettier-formatted/);
  });

  it("accepts already-formatted source", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = true;
    cfg.maxFunctionParameters.enabled = false;
    const source = `export function ok() {\n  return 1;\n}\n`;
    const diagnostics = await new Analyzer(cfg, identityFormat).analyzeFile({
      path: "sample.ts",
      source,
    });
    assert.equal(diagnostics.length, 0);
  });

  it("honors disabled format check", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = false;
    cfg.maxFunctionParameters.enabled = false;
    const runner: FormatRunner = {
      async format() {
        throw new Error("should not run");
      },
    };
    const diagnostics = await new Analyzer(cfg, runner).analyzeFile({
      path: "sample.ts",
      source: `export function ok()  {\n  return 1;\n}\n`,
    });
    assert.equal(diagnostics.length, 0);
  });

  it("errors clearly when prettier runner reports missing tool", async () => {
    const cfg = defaultConfig();
    cfg.format.enabled = true;
    const runner: FormatRunner = {
      async format() {
        throw new Error("prettier not available; install prettier to enforce source-format (VET008)");
      },
    };
    await assert.rejects(
      () =>
        new Analyzer(cfg, runner).analyzeFile({
          path: "sample.ts",
          source: `export const x = 1;\n`,
        }),
      /prettier not available/,
    );
  });
});
