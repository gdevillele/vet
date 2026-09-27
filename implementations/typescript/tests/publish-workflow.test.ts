import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { versionFromTag } from "../scripts/version-from-tag.mjs";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../..",
);
const workflowPath = path.join(
  repoRoot,
  ".github/workflows/publish-typescript.yml",
);

/**
 * Approximate GitHub Actions filter matching for the subset we use in tags:
 * `*` = any run of non-slash chars, `[0-9]` = one digit, other chars literal.
 * This is intentionally simpler than GHA's full language; it exists so we can
 * prove the workflow glob matches real release tags like `v0.1.0` (unlike a
 * regex-style `[0-9]+` pattern under plain fnmatch, where `+` is literal).
 */
function matchGhaStyleGlob(pattern: string, value: string): boolean {
  let re = "";
  for (let i = 0; i < pattern.length; i++) {
    const ch = pattern[i];
    if (ch === "*") {
      re += "[^/]*";
      continue;
    }
    if (ch === "[" && pattern.startsWith("[0-9]", i)) {
      re += "[0-9]";
      i += 4; // skip "0-9]"
      continue;
    }
    if ("\\.^$+?(){}|".includes(ch)) {
      re += `\\${ch}`;
      continue;
    }
    re += ch;
  }
  return new RegExp(`^${re}$`).test(value);
}

function extractTagPatterns(yaml: string): string[] {
  const lines = yaml.split("\n");
  const patterns: string[] = [];
  let inTags = false;
  for (const line of lines) {
    if (/^\s*tags:\s*$/.test(line)) {
      inTags = true;
      continue;
    }
    if (inTags) {
      const m = line.match(/^\s*-\s*["']?([^"'#]+?)["']?\s*(?:#.*)?$/);
      if (m) {
        patterns.push(m[1].trim());
        continue;
      }
      if (/^\s*#/.test(line) || line.trim() === "") {
        continue;
      }
      // Left the tags list (next key at same/less indent as tags)
      if (/^\S/.test(line) || /^\s{0,3}\w/.test(line)) {
        break;
      }
    }
  }
  return patterns;
}

describe("publish-typescript workflow tag filter", () => {
  it("uses a GHA filter glob that matches v0.1.0 (not regex + quantifiers)", () => {
    const yaml = fs.readFileSync(workflowPath, "utf8");
    assert.match(yaml, /environment:\s*production/);
    assert.match(yaml, /secrets\.NPM_TOKEN/);
    assert.match(yaml, /version-from-tag\.mjs/);
    assert.match(yaml, /npm publish/);

    const patterns = extractTagPatterns(yaml);
    assert.ok(patterns.length >= 1, "expected at least one tags: pattern");

    // Reject the common mistake: regex-style + after [0-9] (broken under fnmatch
    // and easy to misread). Prefer * quantifiers in the committed workflow.
    for (const p of patterns) {
      assert.doesNotMatch(
        p,
        /\[0-9\]\+/,
        `tag filter must not use regex-style [0-9]+ (got ${p})`,
      );
    }

    const releaseTags = ["v0.1.0", "v1.2.3", "v1.2.3-alpha.1", "v10.20.30"];
    for (const tag of releaseTags) {
      const matched = patterns.some((p) => matchGhaStyleGlob(p, tag));
      assert.ok(
        matched,
        `tag ${tag} must match workflow filter ${JSON.stringify(patterns)}`,
      );
      // Real semver gate used by the job
      assert.equal(versionFromTag(tag), tag.slice(1));
    }
  });

  it("lets version-from-tag reject loose tags the glob might over-match", () => {
    // Glob v[0-9]*.[0-9]*.[0-9]* can match non-semver strings; the script is
    // the hard gate before npm publish.
    assert.throws(() => versionFromTag("v0.1.x"), /invalid semver/);
    assert.throws(() => versionFromTag("0.1.0"), /must start with 'v'/);
  });
});
