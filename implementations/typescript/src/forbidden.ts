import type { Config } from "./config.js";
import { diagnostic, type Diagnostic } from "./diagnostic.js";
import { matchesExclude } from "./discover.js";

export const RULE_FORBIDDEN_FILES = "VET016";

export function analyzeForbiddenFile(options: {
  path: string;
  relativePath: string;
  rule: Config["forbiddenFiles"];
}): Diagnostic[] {
  const { rule, relativePath } = options;
  if (!rule.enabled || matchesExclude(relativePath, rule.exclude)) {
    return [];
  }

  const pattern = rule.patterns.find((item) =>
    matchesExclude(relativePath, [item]),
  );
  if (pattern === undefined) {
    return [];
  }

  return [
    diagnostic(
      RULE_FORBIDDEN_FILES,
      `file type is forbidden (matches "${pattern}")`,
      options.path,
      1,
      1,
    ),
  ];
}
