import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { run } from "../src/cli.js";
import { defaultConfig, loadConfigFile } from "../src/config.js";
import { analyzeForbiddenFile } from "../src/forbidden.js";
import { withReasons } from "../src/reason.js";

const fixture = new URL(
  "../../../spec/conformance/forbidden-files/",
  import.meta.url,
);

function capture() {
  const out = { stdout: "", stderr: "" };
  return {
    out,
    stdout: { write: (chunk: string) => void (out.stdout += chunk) },
    stderr: { write: (chunk: string) => void (out.stderr += chunk) },
  };
}

async function vet(cwd: string, args: string[]) {
  const cap = capture();
  const code = await run({
    args,
    stdout: cap.stdout,
    stderr: cap.stderr,
    cwd,
  });
  return { code, ...cap.out };
}

function buildLayout(root: string): void {
  const layout = fs.readFileSync(new URL("layout.txt", fixture), "utf8");
  for (const line of layout.split("\n")) {
    const fields = line.trim().split(/\s+/);
    if (fields[0] === "" || fields[0]!.startsWith("#")) {
      continue;
    }
    const target = path.join(root, fields[1]!);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    if (fields[0] === "file" && fields.length === 2) {
      fs.writeFileSync(target, "");
    } else if (fields[0] === "symlink" && fields.length === 3) {
      fs.symlinkSync(fields[2]!, target);
    } else {
      throw new Error(`invalid layout line ${JSON.stringify(line)}`);
    }
  }
}

function writeFiles(root: string, files: string[]): void {
  for (const file of files) {
    fs.mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
    fs.writeFileSync(path.join(root, file), "");
  }
}

function writeConfig(dir: string, name: string, rule: string): string {
  const file = path.join(dir, name);
  fs.writeFileSync(
    file,
    `version: 1\nrules:\n  forbidden-files:\n    ${rule}\n`,
    "utf8",
  );
  return file;
}

describe("forbidden-files", () => {
  it("matches the conformance fixture", async () => {
    const expected = fs
      .readFileSync(new URL("expected.txt", fixture), "utf8")
      .trim()
      .split("\n");
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-forbidden-"));
    buildLayout(root);
    const config = new URL("vet.yaml", fixture).pathname;

    const json = await vet(root, ["--config", config, "--format", "json"]);
    assert.equal(json.code, 1, json.stderr);
    assert.equal(json.stderr, "");
    const diagnostics = JSON.parse(json.stdout).diagnostics as Array<
      Record<string, string | number>
    >;
    assert.deepEqual(
      diagnostics.map(
        (d) => `${d.file}:${d.line}:${d.column}: ${d.rule_id}: ${d.message}`,
      ),
      expected,
    );
    assert.ok(diagnostics.every((d) => d.severity === "error"));

    const text = await vet(root, ["--config", config]);
    assert.equal(text.code, 1, text.stderr);
    assert.equal(text.stdout, `${expected[0]}\n`);
  });

  it("applies config, flags, and explicit paths", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-forbidden-"));
    writeFiles(root, [
      "foo/bar.py",
      "other/tool.py",
      "docs/notes.md",
      "design/archive/old.py",
      ".git/hooks/hook.py",
      ".tools/README.md",
    ]);
    fs.symlinkSync("missing", path.join(root, ".tools", "broken.py"));
    const configs = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-forbidden-cfg-"));
    const enabled = writeConfig(
      configs,
      "enabled.yaml",
      'enabled: true\n    patterns: ["**/*.py"]\n    exclude: ["design/archive/**"]',
    );
    const disabled = writeConfig(
      configs,
      "disabled.yaml",
      'enabled: false\n    patterns: ["**/*.py"]',
    );
    const empty = writeConfig(configs, "empty.yaml", "enabled: true");
    const all = [".tools/broken.py", "foo/bar.py", "other/tool.py"];

    const cases: Array<[string[], number, string[]]> = [
      [["--config", enabled], 1, all],
      [["--config", enabled, "foo"], 1, ["foo/bar.py"]],
      [["--config", enabled, "other/tool.py", "foo/..."], 1, ["foo/bar.py", "other/tool.py"]],
      [["--config", enabled, "design", "docs", ".git"], 0, []],
      [["--config", enabled, "--forbidden-files=false"], 0, []],
      [["--config", disabled], 0, []],
      [
        ["--config", disabled, "--forbidden-files"],
        1,
        [".tools/broken.py", "design/archive/old.py", "foo/bar.py", "other/tool.py"],
      ],
      [["--config", empty], 0, []],
      [["--forbidden-files"], 0, []],
      [["--config", enabled, "--forbidden-files=invalid"], 2, []],
      [["--config", enabled, "missing"], 2, []],
    ];
    for (const [args, code, files] of cases) {
      const result = await vet(root, [
        "--format",
        "json",
        "--check-format=false",
        ...args,
      ]);
      assert.equal(result.code, code, `${args}: ${result.stdout}${result.stderr}`);
      if (code === 2) {
        continue;
      }
      const diagnostics = JSON.parse(result.stdout).diagnostics as Array<
        Record<string, string | number>
      >;
      for (const d of diagnostics) {
        assert.equal(d.rule_id, "VET016");
        assert.equal(d.line, 1);
        assert.equal(d.column, 1);
      }
      assert.deepEqual(diagnostics.map((d) => d.file), files, String(args));
    }
  });

  it("reports the first matching pattern with the reason", () => {
    const cfg = defaultConfig();
    cfg.forbiddenFiles = {
      enabled: true,
      patterns: ["**/*.py", "**/pyproject.toml", "**/requirements*.txt"],
      exclude: ["design/archive/**"],
      reason: " Use Go for tooling and checks\n",
    };
    const check = (relativePath: string) =>
      withReasons(
        cfg,
        analyzeForbiddenFile({ path: relativePath, relativePath, rule: cfg.forbiddenFiles }),
      ).map((d) => d.message);

    assert.deepEqual(check("foo/bar.py"), [
      'file type is forbidden (matches "**/*.py") (reason: Use Go for tooling and checks)',
    ]);
    assert.deepEqual(check("pyproject.toml"), [
      'file type is forbidden (matches "**/pyproject.toml") (reason: Use Go for tooling and checks)',
    ]);
    assert.deepEqual(check("tools/requirements-dev.txt"), [
      'file type is forbidden (matches "**/requirements*.txt") (reason: Use Go for tooling and checks)',
    ]);
    assert.deepEqual(check("design/archive/old/tool.py"), []);
    assert.deepEqual(check("foo/bar.py.txt"), []);
    cfg.forbiddenFiles.enabled = false;
    assert.deepEqual(check("foo/bar.py"), []);
  });

  it("loads top-level config and rejects invalid config", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-forbidden-cfg-"));
    assert.deepEqual(defaultConfig().forbiddenFiles, {
      enabled: false,
      patterns: [],
      exclude: [],
    });

    const valid = writeConfig(
      dir,
      "valid.yaml",
      'enabled: true\n    patterns: ["**/*.py"]\n    exclude: ["design/archive/**"]\n    reason: Use Go for tooling and checks',
    );
    const cfg = loadConfigFile({ path: valid, base: defaultConfig(), language: "typescript" });
    assert.deepEqual(cfg.forbiddenFiles, {
      enabled: true,
      patterns: ["**/*.py"],
      exclude: ["design/archive/**"],
      reason: "Use Go for tooling and checks",
    });

    const invalid: Array<[string, RegExp]> = [
      [
        "version: 1\nlanguages:\n  rust:\n    rules:\n      forbidden-files:\n        enabled: true\n",
        /languages\.rust\.rules\.forbidden-files is not supported; forbidden-files is repo-wide and must be set under top-level rules/,
      ],
      ['version: 1\nrules:\n  forbidden-files:\n    patterns: [""]\n', /forbidden-files\.patterns must not contain empty patterns/],
      ['version: 1\nrules:\n  forbidden-files:\n    exclude: [" "]\n', /forbidden-files\.exclude must not contain empty patterns/],
    ];
    for (const [data, error] of invalid) {
      const file = path.join(dir, "invalid.yaml");
      fs.writeFileSync(file, data, "utf8");
      assert.throws(
        () => loadConfigFile({ path: file, base: defaultConfig(), language: "typescript" }),
        error,
      );
    }
  });

  it("fails the CLI on a language override", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-ts-forbidden-cfg-"));
    fs.writeFileSync(
      path.join(dir, "vet.yaml"),
      "version: 1\nlanguages:\n  typescript:\n    rules:\n      forbidden-files:\n        enabled: true\n",
    );
    const result = await vet(dir, []);
    assert.equal(result.code, 2);
    assert.match(result.stderr, /languages\.typescript\.rules\.forbidden-files is not supported/);
  });
});
