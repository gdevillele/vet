import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import {
  applyVersionToPackageJson,
  versionFromTag,
} from "../scripts/version-from-tag.mjs";

const scriptPath = fileURLToPath(
  new URL("../scripts/version-from-tag.mjs", import.meta.url),
);

describe("versionFromTag", () => {
  it("maps v0.1.0 to 0.1.0", () => {
    assert.equal(versionFromTag("v0.1.0"), "0.1.0");
  });

  it("strips refs/tags/ prefix", () => {
    assert.equal(versionFromTag("refs/tags/v1.2.3"), "1.2.3");
  });

  it("allows prerelease and build metadata", () => {
    assert.equal(versionFromTag("v1.0.0-alpha.1"), "1.0.0-alpha.1");
    assert.equal(versionFromTag("v2.0.0+build.9"), "2.0.0+build.9");
  });

  it("rejects tags without leading v", () => {
    assert.throws(() => versionFromTag("0.1.0"), /must start with 'v'/);
  });

  it("rejects non-semver after the v", () => {
    assert.throws(() => versionFromTag("vnot-a-version"), /invalid semver/);
    assert.throws(() => versionFromTag("v1.2"), /invalid semver/);
  });
});

describe("applyVersionToPackageJson", () => {
  it("updates the version field in a package.json file", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-version-"));
    const pkgPath = path.join(dir, "package.json");
    fs.writeFileSync(
      pkgPath,
      JSON.stringify({ name: "@vet/typescript", version: "0.0.0" }, null, 2) +
        "\n",
      "utf8",
    );
    const result = applyVersionToPackageJson(pkgPath, "0.1.0");
    assert.equal(result.previous, "0.0.0");
    assert.equal(result.version, "0.1.0");
    const written = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
    assert.equal(written.version, "0.1.0");
    assert.equal(written.name, "@vet/typescript");
  });
});

describe("version-from-tag CLI", () => {
  it("prints version for a tag (real script entrypoint)", () => {
    const result = spawnSync(process.execPath, [scriptPath, "v0.1.0"], {
      encoding: "utf8",
    });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout.trim(), "0.1.0");
  });

  it("exits non-zero for invalid tags", () => {
    const result = spawnSync(process.execPath, [scriptPath, "vbad"], {
      encoding: "utf8",
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /invalid semver|must start/);
  });

  it("writes package.json when --write is set (real CLI entrypoint)", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vet-version-cli-"));
    const pkgPath = path.join(dir, "package.json");
    fs.writeFileSync(
      pkgPath,
      JSON.stringify({ name: "@vet/typescript", version: "0.0.0" }, null, 2) +
        "\n",
      "utf8",
    );
    const result = spawnSync(
      process.execPath,
      [scriptPath, "--write", "--package-json", pkgPath, "v0.1.0"],
      { encoding: "utf8" },
    );
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /0\.0\.0 -> 0\.1\.0/);
    const written = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
    assert.equal(written.version, "0.1.0");
  });
});
