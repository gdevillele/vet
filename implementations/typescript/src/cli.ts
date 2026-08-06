import fs from "node:fs";
import path from "node:path";
import {
  DEFAULT_MAX_FUNCTION_PARAMETERS,
  defaultConfig,
  loadConfigFile,
  validate,
  type Config,
  type CasingStyle,
  type FunctionDocstringPolicy,
} from "./config.js";
import { Analyzer } from "./analysis.js";
import { collectTypeScriptFiles, collectWorkflowFiles } from "./discover.js";
import {
  renderJSON,
  renderText,
  sortDiagnostics,
  type Diagnostic,
} from "./diagnostic.js";
import { analyzeWorkflowFile } from "./workflow.js";

export const VERSION = "0.1.0-dev";
const DEFAULT_CONFIG_FILENAME = "vet.yaml";

export interface Invocation {
  args: string[];
  stdout: { write(chunk: string): void };
  stderr: { write(chunk: string): void };
  cwd?: string;
}

type FlagMap = Record<string, string | boolean | undefined>;

function parseArgs(args: string[]): {
  flags: FlagMap;
  visited: Set<string>;
  positionals: string[];
} {
  const flags: FlagMap = {};
  const visited = new Set<string>();
  const positionals: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i]!;
    if (arg === "--") {
      positionals.push(...args.slice(i + 1));
      break;
    }
    if (!arg.startsWith("-")) {
      positionals.push(arg);
      continue;
    }
    if (arg === "-config" || arg.startsWith("-config=")) {
      throw new Error("use -c or --config, not -config");
    }

    let name: string;
    let value: string | boolean | undefined;
    if (arg.startsWith("--")) {
      const eq = arg.indexOf("=");
      if (eq >= 0) {
        name = arg.slice(2, eq);
        value = arg.slice(eq + 1);
      } else {
        name = arg.slice(2);
        const next = args[i + 1];
        if (next !== undefined && !next.startsWith("-")) {
          // boolean flags may be bare
          if (
            name === "check-format" ||
            name === "casing" ||
            name === "require-file-header" ||
            name === "github-actions-pinned" ||
            name === "version"
          ) {
            // peek: if next looks like value for non-bool, leave it
            if (name === "check-format" && (next === "true" || next === "false")) {
              value = next;
              i++;
            } else {
              value = true;
            }
          } else {
            value = next;
            i++;
          }
        } else {
          value = true;
        }
      }
    } else {
      // single-dash long names used by other runners (-c, -format)
      const eq = arg.indexOf("=");
      if (eq >= 0) {
        name = arg.slice(1, eq);
        value = arg.slice(eq + 1);
      } else {
        name = arg.slice(1);
        const next = args[i + 1];
        if (
          next !== undefined &&
          !next.startsWith("-") &&
          name !== "casing" &&
          name !== "require-file-header" &&
          name !== "github-actions-pinned" &&
          name !== "version"
        ) {
          if (name === "check-format" && next !== "true" && next !== "false") {
            value = true;
          } else {
            value = next;
            i++;
          }
        } else {
          value = true;
        }
      }
    }

    // normalize aliases
    if (name === "c") {
      name = "config";
    }
    visited.add(name);
    flags[name] = value;
  }

  return { flags, visited, positionals };
}

function asBool(value: string | boolean | undefined, flag: string): boolean {
  if (value === true || value === undefined) {
    return true;
  }
  if (value === false) {
    return false;
  }
  if (value === "true") {
    return true;
  }
  if (value === "false") {
    return false;
  }
  throw new Error(`${flag} must be true or false`);
}

function asInt(value: string | boolean | undefined, flag: string): number {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${flag} requires an integer value`);
  }
  const n = Number(value);
  if (!Number.isInteger(n)) {
    throw new Error(`${flag} must be an integer`);
  }
  return n;
}

function defaultConfigPath(cwd: string): string | "" {
  const candidate = path.join(cwd, DEFAULT_CONFIG_FILENAME);
  try {
    const st = fs.statSync(candidate);
    if (st.isFile()) {
      return candidate;
    }
  } catch {
    // missing is fine
  }
  return "";
}

export async function run(invocation: Invocation): Promise<number> {
  const cwd = invocation.cwd ?? process.cwd();
  let parsed;
  try {
    parsed = parseArgs(invocation.args);
  } catch (err) {
    invocation.stderr.write(`vet: ${err instanceof Error ? err.message : String(err)}\n`);
    return 2;
  }

  const { flags, visited, positionals } = parsed;

  if (visited.has("version")) {
    invocation.stdout.write(`${VERSION}\n`);
    return 0;
  }

  let cfg: Config = defaultConfig();
  let configPath = "";
  if (visited.has("config")) {
    configPath = String(flags.config ?? "");
  } else {
    configPath = defaultConfigPath(cwd);
  }

  try {
    if (configPath) {
      cfg = loadConfigFile({
        path: path.isAbsolute(configPath)
          ? configPath
          : path.join(cwd, configPath),
        base: cfg,
        language: "typescript",
      });
    }

    if (visited.has("max-function-parameters")) {
      cfg.maxFunctionParameters.max = asInt(
        flags["max-function-parameters"],
        "--max-function-parameters",
      );
    }
    if (visited.has("require-file-header")) {
      cfg.sourceFileHeader.required = asBool(
        flags["require-file-header"],
        "--require-file-header",
      );
    }
    if (visited.has("min-file-header-length")) {
      cfg.sourceFileHeader.minLength = asInt(
        flags["min-file-header-length"],
        "--min-file-header-length",
      );
    }
    if (visited.has("max-file-header-length")) {
      cfg.sourceFileHeader.maxLength = asInt(
        flags["max-file-header-length"],
        "--max-file-header-length",
      );
    }
    if (visited.has("max-source-file-lines")) {
      cfg.sourceFileLines.max = asInt(
        flags["max-source-file-lines"],
        "--max-source-file-lines",
      );
    }
    if (visited.has("max-function-body-lines")) {
      cfg.functionBodyLines.max = asInt(
        flags["max-function-body-lines"],
        "--max-function-body-lines",
      );
    }
    if (visited.has("function-docstring-policy")) {
      cfg.functionDocstring.policy = String(
        flags["function-docstring-policy"],
      ) as FunctionDocstringPolicy;
    }
    if (visited.has("check-format")) {
      cfg.format.enabled = asBool(flags["check-format"], "--check-format");
    }
    if (visited.has("casing")) {
      cfg.casing.enabled = asBool(flags.casing, "--casing");
    }
    if (visited.has("function-casing")) {
      cfg.casing.enabled = true;
      cfg.casing.functions = String(flags["function-casing"]) as CasingStyle;
    }
    if (visited.has("variable-casing")) {
      cfg.casing.enabled = true;
      cfg.casing.variables = String(flags["variable-casing"]) as CasingStyle;
    }
    if (visited.has("type-casing")) {
      cfg.casing.enabled = true;
      cfg.casing.types = String(flags["type-casing"]) as CasingStyle;
    }
    if (visited.has("constant-casing")) {
      cfg.casing.enabled = true;
      cfg.casing.constants = String(flags["constant-casing"]) as CasingStyle;
    }
    if (visited.has("github-actions-pinned")) {
      cfg.githubActionsPinned.enabled = asBool(
        flags["github-actions-pinned"],
        "--github-actions-pinned",
      );
    }

    validate(cfg);
  } catch (err) {
    invocation.stderr.write(
      `vet: ${err instanceof Error ? err.message : String(err)}\n`,
    );
    return 2;
  }

  let paths = positionals;
  let exclude: string[] = [];
  if (paths.length === 0) {
    paths = cfg.fileSelection.files;
    exclude = cfg.fileSelection.exclude;
  }
  if (paths.length === 0) {
    paths = ["."];
  }

  // Resolve relative to cwd
  const resolvedPaths = paths.map((p) =>
    path.isAbsolute(p) ? p : path.join(cwd, p),
  );

  let files: string[];
  try {
    files = collectTypeScriptFiles({ paths: resolvedPaths, exclude });
  } catch (err) {
    invocation.stderr.write(
      `vet: ${err instanceof Error ? err.message : String(err)}\n`,
    );
    return 2;
  }

  const analyzer = new Analyzer(cfg);
  const diagnostics: Diagnostic[] = [];

  for (const file of files) {
    let source: string;
    try {
      source = fs.readFileSync(file, "utf8");
    } catch (err) {
      invocation.stderr.write(
        `vet: ${file}: ${err instanceof Error ? err.message : String(err)}\n`,
      );
      return 2;
    }
    try {
      const fileDiagnostics = await analyzer.analyzeFile({ path: file, source });
      diagnostics.push(...fileDiagnostics);
    } catch (err) {
      invocation.stderr.write(
        `vet: ${file}: ${err instanceof Error ? err.message : String(err)}\n`,
      );
      return 2;
    }
  }

  if (cfg.githubActionsPinned.enabled) {
    let workflowFiles: string[];
    try {
      // Honor Invocation.cwd for the default `.github/workflows` scan, and for
      // explicit directory paths look under `<dir>/.github/workflows` (Go/Rust).
      workflowFiles = collectWorkflowFiles({
        paths: positionals.length > 0 ? resolvedPaths : [],
        explicit: positionals.length > 0,
        cwd,
      });
    } catch (err) {
      invocation.stderr.write(
        `vet: ${err instanceof Error ? err.message : String(err)}\n`,
      );
      return 2;
    }
    for (const file of workflowFiles) {
      let source: string;
      try {
        source = fs.readFileSync(file, "utf8");
      } catch (err) {
        invocation.stderr.write(
          `vet: ${file}: ${err instanceof Error ? err.message : String(err)}\n`,
        );
        return 2;
      }
      diagnostics.push(
        ...analyzeWorkflowFile({
          path: file,
          source,
          enabled: true,
        }),
      );
    }
  }

  const sorted = sortDiagnostics(diagnostics);
  const format = visited.has("format")
    ? String(flags.format ?? "text")
    : "text";

  if (format === "text") {
    invocation.stdout.write(renderText(sorted));
  } else if (format === "json") {
    invocation.stdout.write(renderJSON(sorted));
  } else {
    invocation.stderr.write(`vet: unsupported format ${JSON.stringify(format)}\n`);
    return 2;
  }

  return sorted.length > 0 ? 1 : 0;
}

async function main(): Promise<void> {
  const code = await run({
    args: process.argv.slice(2),
    stdout: process.stdout,
    stderr: process.stderr,
    cwd: process.cwd(),
  });
  process.exit(code);
}

// Only auto-run when executed as the CLI entry point.
const entry = process.argv[1] ? path.resolve(process.argv[1]) : "";
if (
  entry.endsWith(`${path.sep}cli.ts`) ||
  entry.endsWith(`${path.sep}cli.js`) ||
  entry.endsWith(`${path.sep}vet.js`)
) {
  void main();
}
