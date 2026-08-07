import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { analyzeWorkflowFile } from "../src/workflow.js";

describe("GitHub Actions pin (VET014)", () => {
  it("reports unpinned actions", () => {
    const diagnostics = analyzeWorkflowFile({
      path: "workflow.yml",
      source: `name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`,
      enabled: true,
    });
    assert.equal(diagnostics.length, 1);
    assert.equal(diagnostics[0]!.rule_id, "VET014");
  });

  it("accepts full SHA pins", () => {
    const diagnostics = analyzeWorkflowFile({
      path: "workflow.yml",
      source: `name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@f43a0e5ff2bd294095638e18286ca9a3d1956744
`,
      enabled: true,
    });
    assert.equal(diagnostics.length, 0);
  });
});
