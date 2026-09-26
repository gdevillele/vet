import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { run } from "../src/cli.js";
import { it } from "node:test";
import { Analyzer } from "../src/analysis.js";
import { defaultConfig } from "../src/config.js";

it("no-comments conformance", async () => {
  const fixture = new URL(
    "../../../spec/conformance/no-comments/typescript/",
    import.meta.url,
  );
  const source = fs.readFileSync(new URL("comments.tsx", fixture), "utf8");
  const expected = JSON.parse(
    fs.readFileSync(new URL("expected.json", fixture), "utf8"),
  );
  for (const enabled of [false, true]) {
    for (const headerRequired of [false, true]) {
      const cfg = defaultConfig();
      cfg.format.enabled = false;
      cfg.sourceFileHeader.required = headerRequired;
      cfg.noComments = { enabled };
      const diagnostics = await new Analyzer(cfg).analyzeFile({
        path: "comments.tsx",
        source,
      });
      const want = headerRequired
        ? expected
        : [{ line: 1, column: 1 }, ...expected];
      assert.deepEqual(
        diagnostics.map((d) => ({ line: d.line, column: d.column })),
        enabled ? want : [],
      );
      for (const d of diagnostics) {
        assert.equal(d.rule_id, "VET015");
        assert.equal(d.message, "comment is not allowed");
      }
    }
  }
});

it("no-comments config and CLI", async (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-comments-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const file = path.join(dir, "sample.ts");
  fs.writeFileSync(file, "// Header\nconst value = 1;\n");
  fs.writeFileSync(
    path.join(dir, "vet.yaml"),
    `version: 1
rules:
  format: { enabled: false }
  source-file-header: { required: false }
  no-comments: { enabled: false }
languages:
  typescript:
    rules:
      no-comments: { enabled: true }
      source-file-header: { required: true }
`,
  );
  for (const [flags, expected] of [
    [[], 0],
    [["--require-file-header=false"], 1],
    [["--require-file-header=false", "--no-comments=false"], 0],
    [["--require-file-header=false", "--no-comments"], 1],
    [["--no-comments=invalid"], 2],
    [["--require-file-header=invalid"], 2],
  ] as const) {
    let stdout = "";
    let stderr = "";
    const code = await run({
      args: ["--format", "json", ...flags, file],
      cwd: dir,
      stdout: {
        write: (text) => {
          stdout += text;
        },
      },
      stderr: {
        write: (text) => {
          stderr += text;
        },
      },
    });
    assert.equal(code, expected, stdout + stderr);
    if (code === 1) {
      const diagnostics = JSON.parse(stdout).diagnostics;
      assert.equal(diagnostics.length, 1);
      assert.equal(diagnostics[0].rule_id, "VET015");
      assert.equal(diagnostics[0].line, 1);
      assert.equal(diagnostics[0].column, 1);
    }
  }
});
