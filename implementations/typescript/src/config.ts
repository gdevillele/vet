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
  maxFunctionParameters: { enabled: boolean; max: number };
  sourceFileHeader: { required: boolean; minLength: number; maxLength: number };
  sourceFileLines: { max: number };
  functionBodyLines: { max: number };
  functionDocstring: { policy: FunctionDocstringPolicy };
  format: { enabled: boolean };
  casing: {
    enabled: boolean;
    functions: CasingStyle;
    variables: CasingStyle;
    types: CasingStyle;
    constants: CasingStyle;
    ignoreNames: string[];
    ignorePatterns: string[];
  };
  githubActionsPinned: { enabled: boolean };
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
    fileSelection: { files: [], exclude: [] },
  };
}

interface RulesFile {
  "max-function-parameters"?: { enabled?: boolean; max?: number };
  "source-file-header"?: {
    required?: boolean;
    "min-length"?: number;
    "max-length"?: number;
  };
  "max-source-file-lines"?: { max?: number };
  "max-function-body-lines"?: { max?: number };
  "function-docstring"?: { policy?: FunctionDocstringPolicy };
  format?: { enabled?: boolean };
  casing?: {
    enabled?: boolean;
    functions?: CasingStyle;
    variables?: CasingStyle;
    types?: CasingStyle;
    constants?: CasingStyle;
    "ignore-names"?: string[];
    "ignore-patterns"?: string[];
  };
  "github-actions-pinned"?: { enabled?: boolean };
}

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

  return result;
}

export function validate(cfg: Config): void {
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
