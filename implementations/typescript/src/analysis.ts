import ts from "typescript";
import type { Config, CasingStyle } from "./config.js";
import { diagnostic, type Diagnostic } from "./diagnostic.js";
import { checkFormat, type FormatRunner } from "./format.js";

export const RULE_MAX_FUNCTION_PARAMETERS = "VET001";
export const RULE_SOURCE_FILE_HEADER_REQUIRED = "VET002";
export const RULE_SOURCE_FILE_HEADER_MIN = "VET003";
export const RULE_SOURCE_FILE_HEADER_MAX = "VET004";
export const RULE_SOURCE_FILE_LINES = "VET005";
export const RULE_FUNCTION_BODY_LINES = "VET006";
export const RULE_FUNCTION_DOCSTRING = "VET007";
export const RULE_FUNCTION_CASING = "VET010";
export const RULE_VARIABLE_CASING = "VET011";
export const RULE_TYPE_CASING = "VET012";
export const RULE_CONSTANT_CASING = "VET013";

export class Analyzer {
  constructor(
    private readonly config: Config,
    private readonly formatRunner?: FormatRunner,
  ) {}

  async analyzeFile(options: {
    path: string;
    source: string;
  }): Promise<Diagnostic[]> {
    const diagnostics: Diagnostic[] = [];
    diagnostics.push(...this.checkSourceFileLines(options.path, options.source));
    diagnostics.push(
      ...(await checkFormat({
        path: options.path,
        source: options.source,
        enabled: this.config.format.enabled,
        runner: this.formatRunner,
      })),
    );
    diagnostics.push(...this.checkFileHeader(options.path, options.source));

    const kind = options.path.endsWith(".tsx")
      ? ts.ScriptKind.TSX
      : ts.ScriptKind.TS;
    const sourceFile = ts.createSourceFile(
      options.path,
      options.source,
      ts.ScriptTarget.Latest,
      true,
      kind,
    );

    diagnostics.push(...this.checkCasing(sourceFile, options.path));
    this.visitFunctions(sourceFile, options.path, diagnostics);

    return diagnostics;
  }

  private checkSourceFileLines(path: string, source: string): Diagnostic[] {
    const max = this.config.sourceFileLines.max;
    if (max <= 0) {
      return [];
    }
    const count = sourceLineCount(source);
    if (count <= max) {
      return [];
    }
    return [
      diagnostic(
        RULE_SOURCE_FILE_LINES,
        `source file has ${count} lines; maximum allowed is ${max}`,
        path,
        1,
        1,
      ),
    ];
  }

  private checkFileHeader(path: string, source: string): Diagnostic[] {
    const rule = this.config.sourceFileHeader;
    const header = findSourceFileHeader(source);

    if (!header.present) {
      if (!rule.required) {
        return [];
      }
      const { line, column } = offsetToLineColumn(source, header.firstCodeOffset);
      return [
        diagnostic(
          RULE_SOURCE_FILE_HEADER_REQUIRED,
          "source file has no header",
          path,
          line,
          column,
        ),
      ];
    }

    const length = [...header.text].length;
    const { line, column } = offsetToLineColumn(source, header.offset);
    const out: Diagnostic[] = [];
    if (rule.minLength > 0 && length < rule.minLength) {
      out.push(
        diagnostic(
          RULE_SOURCE_FILE_HEADER_MIN,
          `file header has ${length} characters; minimum allowed is ${rule.minLength}`,
          path,
          line,
          column,
        ),
      );
    }
    if (rule.maxLength > 0 && length > rule.maxLength) {
      out.push(
        diagnostic(
          RULE_SOURCE_FILE_HEADER_MAX,
          `file header has ${length} characters; maximum allowed is ${rule.maxLength}`,
          path,
          line,
          column,
        ),
      );
    }
    return out;
  }

  private visitFunctions(
    sourceFile: ts.SourceFile,
    path: string,
    diagnostics: Diagnostic[],
  ): void {
    const visit = (node: ts.Node): void => {
      if (
        ts.isFunctionDeclaration(node) ||
        ts.isMethodDeclaration(node) ||
        ts.isFunctionExpression(node) ||
        ts.isArrowFunction(node)
      ) {
        const name = functionName(node);
        const isLiteral =
          ts.isFunctionExpression(node) || ts.isArrowFunction(node);
        const pos = namePos(node, sourceFile);
        const params = parameterCount(node);
        const bodyLines = bodyLineCount(node, sourceFile);
        const hasDoc = hasJSDoc(node, sourceFile);

        if (this.config.maxFunctionParameters.enabled) {
          const max = this.config.maxFunctionParameters.max;
          if (params > max) {
            diagnostics.push(
              diagnostic(
                RULE_MAX_FUNCTION_PARAMETERS,
                `${name} has ${params} parameters; maximum allowed is ${max}`,
                path,
                pos.line,
                pos.column,
              ),
            );
          }
        }

        if (this.config.functionBodyLines.max > 0 && bodyLines !== null) {
          const max = this.config.functionBodyLines.max;
          if (bodyLines > max) {
            diagnostics.push(
              diagnostic(
                RULE_FUNCTION_BODY_LINES,
                `${name} body has ${bodyLines} lines; maximum allowed is ${max}`,
                path,
                pos.line,
                pos.column,
              ),
            );
          }
        }

        if (!isLiteral || ts.isFunctionDeclaration(node) || ts.isMethodDeclaration(node)) {
          // Docstring applies to named declarations (not pure function literals).
          if (
            (ts.isFunctionDeclaration(node) && node.name) ||
            ts.isMethodDeclaration(node)
          ) {
            diagnostics.push(
              ...this.checkDocstring(name, hasDoc, path, pos.line, pos.column),
            );
          }
        }
      }
      ts.forEachChild(node, visit);
    };
    visit(sourceFile);
  }

  private checkDocstring(
    name: string,
    hasDoc: boolean,
    path: string,
    line: number,
    column: number,
  ): Diagnostic[] {
    const policy = this.config.functionDocstring.policy;
    if (policy === "optional") {
      return [];
    }
    if (policy === "mandatory" && hasDoc) {
      return [];
    }
    if (policy === "forbidden" && !hasDoc) {
      return [];
    }
    const message =
      policy === "mandatory"
        ? `${name} must have a docstring`
        : `${name} must not have a docstring`;
    return [diagnostic(RULE_FUNCTION_DOCSTRING, message, path, line, column)];
  }

  private checkCasing(sourceFile: ts.SourceFile, path: string): Diagnostic[] {
    if (!this.config.casing.enabled) {
      return [];
    }
    const diagnostics: Diagnostic[] = [];
    const ignoreNames = new Set(this.config.casing.ignoreNames);
    const ignorePatterns = this.config.casing.ignorePatterns.map(
      (pattern) => new RegExp(pattern),
    );
    const ignored = (name: string): boolean => {
      if (ignoreNames.has(name)) {
        return true;
      }
      return ignorePatterns.some((re) => re.test(name));
    };

    const check = (
      name: string,
      style: CasingStyle,
      languageDefault: CasingStyle,
      ruleId: string,
      kind: string,
      node: ts.Node,
    ): void => {
      if (ignored(name) || name === "_") {
        return;
      }
      const effective =
        style === "language-default" ? languageDefault : style;
      if (effective === "off") {
        return;
      }
      if (matchesCasing(name, effective)) {
        return;
      }
      const { line, character } = sourceFile.getLineAndCharacterOfPosition(
        node.getStart(sourceFile),
      );
      diagnostics.push(
        diagnostic(
          ruleId,
          `${kind} ${JSON.stringify(name)} must use ${effective}`,
          path,
          line + 1,
          character + 1,
        ),
      );
    };

    const visit = (node: ts.Node): void => {
      if (ts.isFunctionDeclaration(node) && node.name) {
        check(
          node.name.text,
          this.config.casing.functions,
          "camelCase",
          RULE_FUNCTION_CASING,
          "function",
          node.name,
        );
      }
      if (ts.isMethodDeclaration(node) && node.name && ts.isIdentifier(node.name)) {
        check(
          node.name.text,
          this.config.casing.functions,
          "camelCase",
          RULE_FUNCTION_CASING,
          "function",
          node.name,
        );
      }
      if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name)) {
        const isConst =
          node.parent &&
          ts.isVariableDeclarationList(node.parent) &&
          (node.parent.flags & ts.NodeFlags.Const) !== 0;
        if (isConst) {
          check(
            node.name.text,
            this.config.casing.constants,
            "camelCase",
            RULE_CONSTANT_CASING,
            "constant",
            node.name,
          );
        } else {
          check(
            node.name.text,
            this.config.casing.variables,
            "camelCase",
            RULE_VARIABLE_CASING,
            "variable",
            node.name,
          );
        }
      }
      if (ts.isClassDeclaration(node) && node.name) {
        check(
          node.name.text,
          this.config.casing.types,
          "UpperCamelCase",
          RULE_TYPE_CASING,
          "type",
          node.name,
        );
      }
      if (ts.isInterfaceDeclaration(node)) {
        check(
          node.name.text,
          this.config.casing.types,
          "UpperCamelCase",
          RULE_TYPE_CASING,
          "type",
          node.name,
        );
      }
      if (ts.isTypeAliasDeclaration(node)) {
        check(
          node.name.text,
          this.config.casing.types,
          "UpperCamelCase",
          RULE_TYPE_CASING,
          "type",
          node.name,
        );
      }
      if (ts.isEnumDeclaration(node)) {
        check(
          node.name.text,
          this.config.casing.types,
          "UpperCamelCase",
          RULE_TYPE_CASING,
          "type",
          node.name,
        );
      }
      ts.forEachChild(node, visit);
    };
    visit(sourceFile);
    return diagnostics;
  }
}

function functionName(
  node:
    | ts.FunctionDeclaration
    | ts.MethodDeclaration
    | ts.FunctionExpression
    | ts.ArrowFunction,
): string {
  if (ts.isFunctionDeclaration(node) && node.name) {
    return node.name.text;
  }
  if (ts.isMethodDeclaration(node) && ts.isIdentifier(node.name)) {
    return node.name.text;
  }
  if (ts.isFunctionExpression(node) && node.name) {
    return node.name.text;
  }
  return "function literal";
}

function namePos(
  node:
    | ts.FunctionDeclaration
    | ts.MethodDeclaration
    | ts.FunctionExpression
    | ts.ArrowFunction,
  sourceFile: ts.SourceFile,
): { line: number; column: number } {
  let start = node.getStart(sourceFile);
  if (ts.isFunctionDeclaration(node) && node.name) {
    start = node.name.getStart(sourceFile);
  } else if (ts.isMethodDeclaration(node) && ts.isIdentifier(node.name)) {
    start = node.name.getStart(sourceFile);
  } else if (ts.isFunctionExpression(node) && node.name) {
    start = node.name.getStart(sourceFile);
  }
  const { line, character } = sourceFile.getLineAndCharacterOfPosition(start);
  return { line: line + 1, column: character + 1 };
}

function parameterCount(
  node:
    | ts.FunctionDeclaration
    | ts.MethodDeclaration
    | ts.FunctionExpression
    | ts.ArrowFunction,
): number {
  // Do not count TypeScript `this` parameters.
  return node.parameters.filter((p) => p.name.getText() !== "this").length;
}

function bodyLineCount(
  node:
    | ts.FunctionDeclaration
    | ts.MethodDeclaration
    | ts.FunctionExpression
    | ts.ArrowFunction,
  sourceFile: ts.SourceFile,
): number | null {
  if (!node.body) {
    return null;
  }
  if (!ts.isBlock(node.body)) {
    // Concise arrow body: single expression — count as 1 physical line of body.
    return 1;
  }
  const start = sourceFile.getLineAndCharacterOfPosition(node.body.getStart(sourceFile)).line;
  const end = sourceFile.getLineAndCharacterOfPosition(node.body.end).line;
  // Exclude opening and closing brace lines.
  const count = end - start - 1;
  return count < 0 ? 0 : count;
}

function hasJSDoc(node: ts.Node, sourceFile: ts.SourceFile): boolean {
  const ranges = ts.getLeadingCommentRanges(sourceFile.text, node.pos) ?? [];
  for (const range of ranges) {
    const text = sourceFile.text.slice(range.pos, range.end).trim();
    if (text.startsWith("/**") && text.endsWith("*/") && text.length > 5) {
      return true;
    }
  }
  return false;
}

function matchesCasing(name: string, style: CasingStyle): boolean {
  switch (style) {
    case "camelCase":
      return /^[a-z][a-zA-Z0-9]*$/.test(name);
    case "UpperCamelCase":
      return /^[A-Z][a-zA-Z0-9]*$/.test(name);
    case "snake_case":
      return /^[a-z][a-z0-9]*(_[a-z0-9]+)*$/.test(name);
    case "SNAKE_CASE_FULL_CAPS":
      return /^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$/.test(name);
    default:
      return true;
  }
}

export function sourceLineCount(source: string): number {
  if (source.length === 0) {
    return 0;
  }
  let count = 1;
  for (let i = 0; i < source.length; i++) {
    if (source[i] === "\n") {
      count++;
    }
  }
  if (source.endsWith("\n")) {
    count--;
  }
  return count;
}

function findSourceFileHeader(source: string): {
  present: boolean;
  text: string;
  offset: number;
  firstCodeOffset: number;
} {
  let i = 0;
  // Skip BOM / shebang
  if (source.startsWith("#!")) {
    const nl = source.indexOf("\n");
    i = nl === -1 ? source.length : nl + 1;
  }

  const chunks: string[] = [];
  let headerOffset = i;
  let sawHeader = false;

  while (i < source.length) {
    while (i < source.length && /\s/.test(source[i]!)) {
      i++;
    }
    if (source.startsWith("//", i)) {
      if (!sawHeader) {
        headerOffset = i;
        sawHeader = true;
      }
      const end = source.indexOf("\n", i);
      const lineEnd = end === -1 ? source.length : end;
      const line = source.slice(i + 2, lineEnd);
      const normalized = normalizeHeaderLine(line);
      if (!shouldIgnoreHeaderLine(normalized) && normalized !== "") {
        chunks.push(normalized);
      }
      i = end === -1 ? source.length : end + 1;
      continue;
    }
    if (source.startsWith("/*", i)) {
      if (!sawHeader) {
        headerOffset = i;
        sawHeader = true;
      }
      const end = source.indexOf("*/", i + 2);
      const blockEnd = end === -1 ? source.length : end + 2;
      const body = source.slice(i + 2, end === -1 ? source.length : end);
      for (const raw of body.split("\n")) {
        const normalized = normalizeHeaderLine(raw);
        if (!shouldIgnoreHeaderLine(normalized) && normalized !== "") {
          chunks.push(normalized);
        }
      }
      i = blockEnd;
      continue;
    }
    break;
  }

  const text = chunks.join("\n").trim();
  return {
    present: text.length > 0,
    text,
    offset: headerOffset,
    firstCodeOffset: i,
  };
}

function normalizeHeaderLine(line: string): string {
  let result = line.trim();
  if (result.startsWith("*")) {
    result = result.slice(1).trim();
  }
  return result;
}

function shouldIgnoreHeaderLine(line: string): boolean {
  if (line === "") {
    return true;
  }
  if (line.startsWith("Code generated ") && line.includes("DO NOT EDIT.")) {
    return true;
  }
  return false;
}

function offsetToLineColumn(
  source: string,
  offset: number,
): { line: number; column: number } {
  let line = 1;
  let column = 1;
  const limit = Math.min(offset, source.length);
  for (let i = 0; i < limit; i++) {
    if (source[i] === "\n") {
      line++;
      column = 1;
    } else {
      column++;
    }
  }
  return { line, column };
}
