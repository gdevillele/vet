import prettier from "prettier";
import { diagnostic, type Diagnostic } from "./diagnostic.js";

export const RULE_SOURCE_FORMAT = "VET008";

export type FormatRunner = {
  format: (source: string, filePath: string) => Promise<string>;
};

/**
 * Industry-standard Prettier formatter. Using the prettier package API matches
 * go/format for Go: same tool semantics, no silent skip when the dependency is
 * installed with the runner.
 */
export const prettierRunner: FormatRunner = {
  async format(source: string, filePath: string): Promise<string> {
    try {
      return await prettier.format(source, {
        filepath: filePath,
        // Prefer project config when present; fall back to Prettier defaults.
        ...(await prettier.resolveConfig(filePath).catch(() => null)),
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (
        message.includes("Cannot find module") ||
        message.includes("Cannot find package") ||
        (/prettier/i.test(message) && /not found|ENOENT/i.test(message))
      ) {
        throw new Error(
          "prettier not available; install prettier to enforce source-format (VET008)",
        );
      }
      throw err;
    }
  },
};

export async function checkFormat(options: {
  path: string;
  source: string;
  enabled: boolean;
  runner?: FormatRunner;
}): Promise<Diagnostic[]> {
  if (!options.enabled) {
    return [];
  }

  const runner = options.runner ?? prettierRunner;
  let formatted: string;
  try {
    formatted = await runner.format(options.source, options.path);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (/prettier not available|Cannot find module ['"]prettier['"]/i.test(message)) {
      throw new Error(
        "prettier not available; install prettier to enforce source-format (VET008)",
      );
    }
    // Unparseable sources are not format violations; structural parse handles syntax.
    return [];
  }

  if (formatted === options.source) {
    return [];
  }

  return [
    diagnostic(
      RULE_SOURCE_FORMAT,
      "file is not prettier-formatted",
      options.path,
      1,
      1,
    ),
  ];
}
