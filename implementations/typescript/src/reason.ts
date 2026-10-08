import {
  RULE_CONSTANT_CASING,
  RULE_FUNCTION_BODY_LINES,
  RULE_FUNCTION_CASING,
  RULE_FUNCTION_DOCSTRING,
  RULE_MAX_FUNCTION_PARAMETERS,
  RULE_NO_COMMENTS,
  RULE_SOURCE_FILE_HEADER_MAX,
  RULE_SOURCE_FILE_HEADER_MIN,
  RULE_SOURCE_FILE_HEADER_REQUIRED,
  RULE_SOURCE_FILE_LINES,
  RULE_TYPE_CASING,
  RULE_VARIABLE_CASING,
} from "./analysis.js";
import type { Config } from "./config.js";
import type { Diagnostic } from "./diagnostic.js";
import { RULE_FORBIDDEN_FILES } from "./forbidden.js";
import { RULE_SOURCE_FORMAT } from "./format.js";
import { RULE_GITHUB_ACTIONS_PINNED } from "./workflow.js";

/** Appends each rule's configured reason so developers know why the rule is enforced. */
export function withReasons(cfg: Config, items: Diagnostic[]): Diagnostic[] {
  return items.map((item) => {
    const reason = reasonFor(cfg, item.rule_id)?.trim();
    return reason
      ? { ...item, message: `${item.message} (reason: ${reason})` }
      : item;
  });
}

function reasonFor(cfg: Config, ruleId: string): string | undefined {
  switch (ruleId) {
    case RULE_MAX_FUNCTION_PARAMETERS:
      return cfg.maxFunctionParameters.reason;
    case RULE_SOURCE_FILE_HEADER_REQUIRED:
    case RULE_SOURCE_FILE_HEADER_MIN:
    case RULE_SOURCE_FILE_HEADER_MAX:
      return cfg.sourceFileHeader.reason;
    case RULE_SOURCE_FILE_LINES:
      return cfg.sourceFileLines.reason;
    case RULE_FUNCTION_BODY_LINES:
      return cfg.functionBodyLines.reason;
    case RULE_FUNCTION_DOCSTRING:
      return cfg.functionDocstring.reason;
    case RULE_SOURCE_FORMAT:
      return cfg.format.reason;
    case RULE_FUNCTION_CASING:
    case RULE_VARIABLE_CASING:
    case RULE_TYPE_CASING:
    case RULE_CONSTANT_CASING:
      return cfg.casing.reason;
    case RULE_GITHUB_ACTIONS_PINNED:
      return cfg.githubActionsPinned.reason;
    case RULE_NO_COMMENTS:
      return cfg.noComments.reason;
    case RULE_FORBIDDEN_FILES:
      return cfg.forbiddenFiles.reason;
    default:
      return undefined;
  }
}
