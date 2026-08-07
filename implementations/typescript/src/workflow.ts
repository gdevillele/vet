import { parseDocument, isMap, isSeq, isScalar } from "yaml";
import { diagnostic, type Diagnostic } from "./diagnostic.js";

export const RULE_GITHUB_ACTIONS_PINNED = "VET014";

const FULL_SHA = /^[0-9a-fA-F]{40}$/;

function lineOf(node: { range?: [number, number, number] | null } | null | undefined, source: string): number {
  if (!node?.range) {
    return 1;
  }
  const offset = node.range[0];
  let line = 1;
  for (let i = 0; i < offset && i < source.length; i++) {
    if (source[i] === "\n") {
      line++;
    }
  }
  return line;
}

function columnOf(node: { range?: [number, number, number] | null } | null | undefined, source: string): number {
  if (!node?.range) {
    return 1;
  }
  const offset = node.range[0];
  let column = 1;
  for (let i = offset - 1; i >= 0; i--) {
    if (source[i] === "\n") {
      break;
    }
    column++;
  }
  return column;
}

export function analyzeWorkflowFile(options: {
  path: string;
  source: string;
  enabled: boolean;
}): Diagnostic[] {
  if (!options.enabled) {
    return [];
  }

  const doc = parseDocument(options.source, { keepSourceTokens: true });
  if (!isMap(doc.contents)) {
    return [];
  }

  const jobs = doc.contents.get("jobs", true);
  if (!isMap(jobs)) {
    return [];
  }

  const diagnostics: Diagnostic[] = [];

  for (const jobItem of jobs.items) {
    if (!isMap(jobItem.value)) {
      continue;
    }
    const steps = jobItem.value.get("steps", true);
    if (!isSeq(steps)) {
      continue;
    }
    for (const step of steps.items) {
      if (!isMap(step)) {
        continue;
      }
      const usesNode = step.get("uses", true);
      if (!isScalar(usesNode) || typeof usesNode.value !== "string") {
        continue;
      }
      const uses = usesNode.value;
      if (uses.startsWith("./") || uses.startsWith("docker://")) {
        continue;
      }
      const at = uses.lastIndexOf("@");
      const ref = at >= 0 ? uses.slice(at + 1) : "";
      if (FULL_SHA.test(ref)) {
        continue;
      }
      diagnostics.push(
        diagnostic(
          RULE_GITHUB_ACTIONS_PINNED,
          `GitHub action ${JSON.stringify(uses)} must be pinned to a full-length commit SHA`,
          options.path,
          lineOf(usesNode, options.source),
          columnOf(usesNode, options.source),
        ),
      );
    }
  }

  return diagnostics;
}
