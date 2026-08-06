export type Severity = "error";

export interface Diagnostic {
  rule_id: string;
  severity: Severity;
  message: string;
  file: string;
  line: number;
  column: number;
}

export function diagnostic(
  ruleId: string,
  message: string,
  file: string,
  line: number,
  column: number,
): Diagnostic {
  return {
    rule_id: ruleId,
    severity: "error",
    message,
    file,
    line,
    column,
  };
}

export function sortDiagnostics(items: Diagnostic[]): Diagnostic[] {
  return [...items].sort((left, right) => {
    if (left.file !== right.file) {
      return left.file < right.file ? -1 : 1;
    }
    if (left.line !== right.line) {
      return left.line - right.line;
    }
    if (left.column !== right.column) {
      return left.column - right.column;
    }
    if (left.rule_id !== right.rule_id) {
      return left.rule_id < right.rule_id ? -1 : 1;
    }
    return 0;
  });
}

export function renderText(items: Diagnostic[]): string {
  if (items.length === 0) {
    return "";
  }
  const first = items[0];
  return `${first.file}:${first.line}:${first.column}: ${first.rule_id}: ${first.message}\n`;
}

export function renderJSON(items: Diagnostic[]): string {
  return `${JSON.stringify({ diagnostics: items }, null, 2)}\n`;
}
