import fs from "node:fs";
import { parse as parseYaml } from "yaml";

export const DEFAULT_MAX_FUNCTION_PARAMETERS = 1;

export type FunctionDocstringPolicy = "forbidden" | "optional" | "mandatory";
export type CasingStyle =
  | "off"
  | "language-default"
  | "camelCase"
  | "UpperCamelCase"
  | "snake_case"
  | "SNAKE_CASE_FULL_CAPS";

export interface Config {
  noComments: { enabled: boolean; reason?: string };
  maxFunctionParameters: { enabled: boolean; max: number; reason?: string };
  sourceFileHeader: {
    required: boolean;
    minLength: number;
    maxLength: number;
    reason?: string;
  };
  sourceFileLines: { max: number; reason?: string };
  functionBodyLines: { max: number; reason?: string };
  functionDocstring: { policy: FunctionDocstringPolicy; reason?: string };
  format: { enabled: boolean; reason?: string };
  casing: {
    enabled: boolean;
    functions: CasingStyle;
    variables: CasingStyle;
    types: CasingStyle;
    constants: CasingStyle;
    ignoreNames: string[];
    ignorePatterns: string[];
    reason?: string;
  };
  githubActionsPinned: { enabled: boolean; reason?: string };
  forbiddenFiles: {
    enabled: boolean;
    patterns: string[];
    exclude: string[];
    reason?: string;
  };
  fileSelection: { files: string[]; exclude: string[] };
}

export function defaultConfig(): Config {
  return {
    maxFunctionParameters: {
      enabled: true,
      max: DEFAULT_MAX_FUNCTION_PARAMETERS,
    },
    sourceFileHeader: {
      required: false,
      minLength: 0,
      maxLength: 0,
    },
    noComments: { enabled: false },
    sourceFileLines: { max: 0 },
    functionBodyLines: { max: 0 },
    functionDocstring: { policy: "optional" },
    format: { enabled: true },
    casing: {
      enabled: false,
      functions: "language-default",
      variables: "language-default",
      types: "language-default",
      constants: "language-default",
      ignoreNames: [],
      ignorePatterns: [],
    },
    githubActionsPinned: { enabled: false },
    forbiddenFiles: { enabled: false, patterns: [], exclude: [] },
    fileSelection: { files: [], exclude: [] },
  };
}

interface RulesFile {
  "no-comments"?: { enabled?: boolean; reason?: string };
  "max-function-parameters"?: {
    enabled?: boolean;
    max?: number;
    reason?: string;
  };
  "source-file-header"?: {
    required?: boolean;
    "min-length"?: number;
    "max-length"?: number;
    reason?: string;
  };
  "max-source-file-lines"?: { max?: number; reason?: string };
  "max-function-body-lines"?: { max?: number; reason?: string };
  "function-docstring"?: {
    policy?: FunctionDocstringPolicy;
    reason?: string;
  };
  format?: { enabled?: boolean; reason?: string };
  casing?: {
    enabled?: boolean;
    functions?: CasingStyle;
    variables?: CasingStyle;
    types?: CasingStyle;
    constants?: CasingStyle;
    "ignore-names"?: string[];
    "ignore-patterns"?: string[];
    reason?: string;
  };
  "github-actions-pinned"?: { enabled?: boolean; reason?: string };
  "forbidden-files"?: {
    enabled?: boolean;
    patterns?: string[];
    exclude?: string[];
    reason?: string;
  };
}

const RULE_FIELDS: Record<
  keyof RulesFile,
  Exclude<keyof Config, "fileSelection">
> = {
  "no-comments": "noComments",
  "max-function-parameters": "maxFunctionParameters",
  "source-file-header": "sourceFileHeader",
  "max-source-file-lines": "sourceFileLines",
  "max-function-body-lines": "functionBodyLines",
  "function-docstring": "functionDocstring",
  format: "format",
  casing: "casing",
  "github-actions-pinned": "githubActionsPinned",
  "forbidden-files": "forbiddenFiles",
};

interface FileConfig {
  version?: number;
  rules?: RulesFile;
  languages?: Record<
    string,
    { files?: string[]; exclude?: string[]; rules?: RulesFile }
  >;
}

function applyRules(cfg: Config, rules: RulesFile | undefined): Config {
  if (!rules) {
    return cfg;
  }
  const result: Config = structuredClone(cfg);

  const maxParams = rules["max-function-parameters"];
  if (maxParams) {
    if (maxParams.enabled !== undefined) {
      result.maxFunctionParameters.enabled = maxParams.enabled;
    }
    if (maxParams.max !== undefined) {
      result.maxFunctionParameters.max = maxParams.max;
    }
  }

  const comments = rules["no-comments"];
  if (comments?.enabled !== undefined)
    result.noComments.enabled = comments.enabled;

  const header = rules["source-file-header"];
  if (header) {
    if (header.required !== undefined) {
      result.sourceFileHeader.required = header.required;
    }
    if (header["min-length"] !== undefined) {
      result.sourceFileHeader.minLength = header["min-length"];
    }
    if (header["max-length"] !== undefined) {
      result.sourceFileHeader.maxLength = header["max-length"];
    }
  }

  if (rules["max-source-file-lines"]?.max !== undefined) {
    result.sourceFileLines.max = rules["max-source-file-lines"].max;
  }
  if (rules["max-function-body-lines"]?.max !== undefined) {
    result.functionBodyLines.max = rules["max-function-body-lines"].max;
  }
  if (rules["function-docstring"]?.policy !== undefined) {
    result.functionDocstring.policy = rules["function-docstring"].policy;
  }
  if (rules.format?.enabled !== undefined) {
    result.format.enabled = rules.format.enabled;
  }

  const casing = rules.casing;
  if (casing) {
    if (casing.enabled !== undefined) {
      result.casing.enabled = casing.enabled;
    }
    if (casing.functions !== undefined) {
      result.casing.functions = casing.functions;
    }
    if (casing.variables !== undefined) {
      result.casing.variables = casing.variables;
    }
    if (casing.types !== undefined) {
      result.casing.types = casing.types;
    }
    if (casing.constants !== undefined) {
      result.casing.constants = casing.constants;
    }
    if (casing["ignore-names"] !== undefined) {
      result.casing.ignoreNames = casing["ignore-names"];
    }
    if (casing["ignore-patterns"] !== undefined) {
      result.casing.ignorePatterns = casing["ignore-patterns"];
    }
  }

  if (rules["github-actions-pinned"]?.enabled !== undefined) {
    result.githubActionsPinned.enabled = rules["github-actions-pinned"].enabled;
  }

  const forbidden = rules["forbidden-files"];
  if (forbidden) {
    if (forbidden.enabled !== undefined) {
      result.forbiddenFiles.enabled = forbidden.enabled;
    }
    if (forbidden.patterns !== undefined) {
      result.forbiddenFiles.patterns = forbidden.patterns;
    }
    if (forbidden.exclude !== undefined) {
      result.forbiddenFiles.exclude = forbidden.exclude;
    }
  }

  for (const [key, field] of Object.entries(RULE_FIELDS)) {
    const reason = rules[key as keyof RulesFile]?.reason;
    if (reason !== undefined) {
      if (typeof reason !== "string") {
        throw new Error(`${key}.reason must be a string`);
      }
      result[field].reason = reason;
    }
  }

  return result;
}

export function validate(cfg: Config): void {
  if (typeof cfg.noComments.enabled !== "boolean") {
    throw new Error("no-comments.enabled must be a boolean");
  }
  if (cfg.maxFunctionParameters.max < 0) {
    throw new Error("max-function-parameters.max must be zero or greater");
  }
  if (cfg.sourceFileHeader.minLength < 0) {
    throw new Error("source-file-header.min-length must be zero or greater");
  }
  if (cfg.sourceFileHeader.maxLength < 0) {
    throw new Error("source-file-header.max-length must be zero or greater");
  }
  if (
    cfg.sourceFileHeader.minLength > 0 &&
    cfg.sourceFileHeader.maxLength > 0 &&
    cfg.sourceFileHeader.maxLength < cfg.sourceFileHeader.minLength
  ) {
    throw new Error(
      "source-file-header.max-length must be greater than or equal to source-file-header.min-length",
    );
  }
  if (cfg.sourceFileLines.max < 0) {
    throw new Error("max-source-file-lines.max must be zero or greater");
  }
  if (cfg.functionBodyLines.max < 0) {
    throw new Error("max-function-body-lines.max must be zero or greater");
  }
  const policies: FunctionDocstringPolicy[] = [
    "forbidden",
    "optional",
    "mandatory",
  ];
  if (!policies.includes(cfg.functionDocstring.policy)) {
    throw new Error(
      "function-docstring.policy must be forbidden, optional, or mandatory",
    );
  }
  const styles: CasingStyle[] = [
    "off",
    "language-default",
    "camelCase",
    "UpperCamelCase",
    "snake_case",
    "SNAKE_CASE_FULL_CAPS",
  ];
  for (const [field, style] of [
    ["casing.functions", cfg.casing.functions],
    ["casing.variables", cfg.casing.variables],
    ["casing.types", cfg.casing.types],
    ["casing.constants", cfg.casing.constants],
  ] as const) {
    if (!styles.includes(style)) {
      throw new Error(
        `${field} must be off, language-default, camelCase, UpperCamelCase, snake_case, or SNAKE_CASE_FULL_CAPS`,
      );
    }
  }
  for (const [field, patterns] of [
    ["forbidden-files.patterns", cfg.forbiddenFiles.patterns],
    ["forbidden-files.exclude", cfg.forbiddenFiles.exclude],
  ] as const) {
    if (!Array.isArray(patterns)) {
      throw new Error(`${field} must be a list of strings`);
    }
    for (const pattern of patterns) {
      if (typeof pattern !== "string") {
        throw new Error(`${field} must be a list of strings`);
      }
      if (pattern.trim() === "") {
        throw new Error(`${field} must not contain empty patterns`);
      }
    }
  }
  for (const pattern of cfg.casing.ignorePatterns) {
    try {
      new RegExp(pattern);
    } catch (err) {
      throw new Error(
        `casing.ignore-patterns contains invalid regex ${JSON.stringify(pattern)}: ${err}`,
      );
    }
  }
}

export function loadConfigFile(options: {
  path: string;
  base: Config;
  language: string;
}): Config {
  const data = fs.readFileSync(options.path, "utf8");
  const document = parseYaml(data) as FileConfig;
  if (document.version !== undefined && document.version !== 1) {
    throw new Error(
      `config ${JSON.stringify(options.path)} uses unsupported version ${document.version}`,
    );
  }

  const overridden = Object.entries(document.languages ?? {})
    .filter(([, language]) => language?.rules?.["forbidden-files"] !== undefined)
    .map(([name]) => name)
    .sort();
  if (overridden.length > 0) {
    throw new Error(
      `config ${JSON.stringify(options.path)}: languages.${overridden[0]}.rules.forbidden-files is not supported; forbidden-files is repo-wide and must be set under top-level rules`,
    );
  }

  let result = applyRules(options.base, document.rules);
  const language = document.languages?.[options.language];
  if (language) {
    result = applyRules(result, language.rules);
    result.fileSelection = {
      files: language.files ? [...language.files] : [],
      exclude: language.exclude ? [...language.exclude] : [],
    };
  }
  validate(result);
  return result;
}
