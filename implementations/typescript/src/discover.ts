import fs from "node:fs";
import path from "node:path";

const TS_EXTENSIONS = new Set([".ts", ".tsx", ".mts", ".cts"]);

export function isTypeScriptFile(filePath: string): boolean {
  const ext = path.extname(filePath);
  if (!TS_EXTENSIONS.has(ext)) {
    return false;
  }
  // Skip declaration files.
  if (filePath.endsWith(".d.ts") || filePath.endsWith(".d.mts") || filePath.endsWith(".d.cts")) {
    return false;
  }
  return true;
}

function matchesExclude(filePath: string, patterns: string[]): boolean {
  const normalized = filePath.replaceAll("\\", "/");
  const base = path.basename(normalized);
  for (const pattern of patterns) {
    const p = pattern.replaceAll("\\", "/");
    if (p.startsWith("**/")) {
      const suffix = p.slice(3);
      if (base === suffix || normalized.endsWith(`/${suffix}`) || minimatchSuffix(normalized, suffix)) {
        return true;
      }
      if (normalized.includes(`/${suffix.replace(/^\*\//, "")}`)) {
        // fall through to simple checks
      }
    }
    if (p.includes("**")) {
      // directory prefix: vendor/**
      const prefix = p.replace(/\/\*\*$/, "").replace(/\*\*$/, "");
      if (
        normalized === prefix ||
        normalized.startsWith(`${prefix}/`) ||
        normalized.includes(`/${prefix}/`)
      ) {
        return true;
      }
    }
    if (p.includes("*") && !p.includes("/")) {
      // basename glob: *.test.ts
      const re = new RegExp(
        `^${p.replace(/[.+^${}()|[\]\\]/g, "\\$&").replaceAll("*", ".*")}$`,
      );
      if (re.test(base)) {
        return true;
      }
    }
    if (normalized === p || normalized.endsWith(`/${p}`) || base === p) {
      return true;
    }
  }
  return false;
}

function minimatchSuffix(normalized: string, suffix: string): boolean {
  if (!suffix.includes("*")) {
    return normalized.endsWith(suffix);
  }
  const re = new RegExp(
    `${suffix.replace(/[.+^${}()|[\]\\]/g, "\\$&").replaceAll("*", ".*")}$`,
  );
  return re.test(normalized);
}

function walkDir(dir: string, recursive: boolean, out: string[]): void {
  let entries: fs.Dirent[];
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === "node_modules" || entry.name === "dist" || entry.name === ".git") {
        continue;
      }
      if (recursive) {
        walkDir(full, true, out);
      }
      continue;
    }
    if (entry.isFile() && isTypeScriptFile(full)) {
      out.push(full);
    }
  }
}

export function collectTypeScriptFiles(options: {
  paths: string[];
  exclude: string[];
}): string[] {
  const seen = new Set<string>();
  const files: string[] = [];

  for (const raw of options.paths) {
    const target = raw.endsWith("/...")
      ? raw.slice(0, -4)
      : raw.endsWith("...")
        ? raw.slice(0, -3)
        : raw;
    const recursive = raw.endsWith("...") || raw.endsWith("/...");

    if (target.includes("*")) {
      // simple glob: only support basename-style under a directory prefix
      const dir = path.dirname(target);
      const basePattern = path.basename(target);
      const re = new RegExp(
        `^${basePattern.replace(/[.+^${}()|[\]\\]/g, "\\$&").replaceAll("*", ".*")}$`,
      );
      if (fs.existsSync(dir)) {
        for (const name of fs.readdirSync(dir)) {
          if (re.test(name) && isTypeScriptFile(name)) {
            const full = path.join(dir, name);
            if (!seen.has(full) && !matchesExclude(full, options.exclude)) {
              seen.add(full);
              files.push(full);
            }
          }
        }
      }
      continue;
    }

    if (!fs.existsSync(target)) {
      throw new Error(`path does not exist: ${raw}`);
    }
    const stat = fs.statSync(target);
    if (stat.isFile()) {
      if (isTypeScriptFile(target) && !seen.has(target) && !matchesExclude(target, options.exclude)) {
        seen.add(target);
        files.push(target);
      }
      continue;
    }
    if (stat.isDirectory()) {
      const found: string[] = [];
      walkDir(target, recursive || true, found);
      for (const full of found) {
        if (!seen.has(full) && !matchesExclude(full, options.exclude)) {
          seen.add(full);
          files.push(full);
        }
      }
    }
  }

  return files.sort();
}

function isWorkflowYaml(filePath: string): boolean {
  const base = path.basename(filePath);
  return base.endsWith(".yml") || base.endsWith(".yaml");
}

function addWorkflowDir(
  dir: string,
  seen: Set<string>,
  files: string[],
): void {
  if (!fs.existsSync(dir) || !fs.statSync(dir).isDirectory()) {
    return;
  }
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    if (fs.statSync(full).isDirectory()) {
      continue;
    }
    if (isWorkflowYaml(full) && !seen.has(full)) {
      seen.add(full);
      files.push(full);
    }
  }
}

/**
 * Collect GitHub Actions workflow files.
 * Matches Go/Rust runners:
 * - non-explicit: `<cwd>/.github/workflows/*.{yml,yaml}`
 * - explicit file: include if it is a workflow YAML
 * - explicit directory: prefer `<dir>/.github/workflows`, else treat as a workflow dir
 */
export function collectWorkflowFiles(options: {
  paths: string[];
  explicit: boolean;
  /** Working directory used for the default `.github/workflows` scan. */
  cwd?: string;
}): string[] {
  const seen = new Set<string>();
  const files: string[] = [];
  const cwd = options.cwd ?? process.cwd();

  if (options.explicit && options.paths.length > 0) {
    for (const raw of options.paths) {
      if (!fs.existsSync(raw)) {
        continue;
      }
      const stat = fs.statSync(raw);
      if (stat.isFile()) {
        if (isWorkflowYaml(raw) && !seen.has(raw)) {
          seen.add(raw);
          files.push(raw);
        }
        continue;
      }
      if (stat.isDirectory()) {
        const nested = path.join(raw, ".github", "workflows");
        if (fs.existsSync(nested) && fs.statSync(nested).isDirectory()) {
          addWorkflowDir(nested, seen, files);
        } else {
          // Path may itself be a workflows directory (or contain workflow files).
          addWorkflowDir(raw, seen, files);
        }
      }
    }
    return files.sort();
  }

  addWorkflowDir(path.join(cwd, ".github", "workflows"), seen, files);
  return files.sort();
}
