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
        let codeLines = maskNonCode(request.source).split(separator: "\n", omittingEmptySubsequences: false)
        var diagnostics: [Diagnostic] = []
        var continuationState = ContinuationState()
        var offset = 0

        for item in lines.enumerated() {
            let line = String(item.element)
            let isContinuation = continuationState.consume(String(codeLines[item.offset]))
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

                if request.rule.width > 0 &&
                    leading.count % request.rule.width != 0 &&
                    !isContinuation {
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

private struct OpenDelimiter {
    let character: Character
    let blockDepth: Int
}

private struct ContinuationState {
    private var blockDepth = 0
    private var delimiters: [OpenDelimiter] = []
    private var commaBlockDepth: Int?

    mutating func consume(_ line: String) -> Bool {
        let code = line.trimmingCharacters(in: .whitespacesAndNewlines)
        let lineBlockDepth = max(0, blockDepth - leadingClosingBraceCount(code))
        let isContinuation = delimiters.contains { $0.blockDepth == lineBlockDepth } ||
            commaBlockDepth == lineBlockDepth ||
            code.hasPrefix(".")

        updateDepths(code)
        if !code.isEmpty {
            commaBlockDepth = code.last == "," ? blockDepth : nil
        }

        return isContinuation
    }

    private func leadingClosingBraceCount(_ code: String) -> Int {
        code.prefix(while: { $0 == "}" }).count
    }

    private mutating func updateDepths(_ code: String) {
        for character in code {
            switch character {
            case "{":
                blockDepth += 1
            case "}":
                blockDepth = max(0, blockDepth - 1)
            case "(", "[":
                delimiters.append(OpenDelimiter(character: character, blockDepth: blockDepth))
            case ")":
                closeDelimiter("(")
            case "]":
                closeDelimiter("[")
            default:
                break
            }
        }
    }

    private mutating func closeDelimiter(_ character: Character) {
        guard let index = delimiters.lastIndex(where: { $0.character == character }) else {
            return
        }
        delimiters.remove(at: index)
    }
}
