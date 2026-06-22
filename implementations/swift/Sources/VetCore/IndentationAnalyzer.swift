import Foundation

struct IndentAnalyzeRequest {
    let path: String
    let source: String
    let rule: IndentRule
}

enum IndentationAnalyzer {
    static func analyze(_ request: IndentAnalyzeRequest) -> [Diagnostic] {
        let effectiveType = request.rule.type == .languageDefault ? IndentType.spaces : request.rule.type
        let lines = request.source.split(separator: "\n", omittingEmptySubsequences: false)
        var diagnostics: [Diagnostic] = []
        var offset = 0

        for item in lines.enumerated() {
            let line = String(item.element)
            defer {
                offset += line.count + 1
            }

            if line.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                continue
            }

            let leading = leadingIndent(line)
            if leading.isEmpty {
                continue
            }

            switch effectiveType {
            case .spaces:
                if let column = leading.firstIndex(of: "\t") {
                    diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                        ruleID: RuleID.indentType,
                        message: "line indentation uses tabs; expected spaces",
                        path: request.path,
                        source: request.source,
                        offset: offset + line.distance(from: line.startIndex, to: column)
                    )))
                    continue
                }

                if request.rule.width > 0 && leading.count % request.rule.width != 0 {
                    diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                        ruleID: RuleID.indentWidth,
                        message: "line indentation has \(leading.count) spaces; expected a multiple of \(request.rule.width)",
                        path: request.path,
                        source: request.source,
                        offset: offset
                    )))
                }
            case .tabs:
                if let column = leading.firstIndex(of: " ") {
                    diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                        ruleID: RuleID.indentType,
                        message: "line indentation uses spaces; expected tabs",
                        path: request.path,
                        source: request.source,
                        offset: offset + line.distance(from: line.startIndex, to: column)
                    )))
                }
            case .languageDefault:
                break
            }
        }

        return diagnostics
    }

    private static func leadingIndent(_ line: String) -> String {
        var result = ""
        for character in line {
            if character != " " && character != "\t" {
                break
            }
            result.append(character)
        }
        return result
    }
}
