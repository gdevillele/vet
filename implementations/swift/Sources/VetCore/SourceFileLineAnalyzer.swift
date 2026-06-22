import Foundation

struct SourceFileLineAnalyzeRequest {
    let path: String
    let source: String
    let rule: SourceFileLinesRule
}

enum SourceFileLineAnalyzer {
    static func analyze(_ request: SourceFileLineAnalyzeRequest) -> [Diagnostic] {
        guard request.rule.max > 0 else {
            return []
        }

        let count = sourceLineCount(request.source)
        guard count > request.rule.max else {
            return []
        }

        return [makeDiagnostic(DiagnosticBuildRequest(
            ruleID: RuleID.sourceFileLines,
            message: "source file has \(count) lines; maximum allowed is \(request.rule.max)",
            path: request.path,
            source: request.source,
            offset: 0
        ))]
    }
}

func sourceLineCount(_ source: String) -> Int {
    if source.isEmpty {
        return 0
    }

    var count = 1
    for character in source where character == "\n" {
        count += 1
    }

    if source.last == "\n" {
        count -= 1
    }
    return count
}
