import SwiftParser
import SwiftSyntax

enum CommentAnalyzer {
    static func analyze(_ request: AnalyzeFileRequest, rule: NoCommentsRule) -> [Diagnostic] {
        guard rule.enabled else { return [] }
        let header = SourceFileHeaderAnalyzer.parseHeader(request.source)
        let characters = Array(request.source)
        let headerStart = String(characters.prefix(header.offset)).utf8.count
        let headerEnd = String(characters.prefix(header.firstCodeOffset)).utf8.count
        let tree = Parser.parse(source: request.source)
        let locations = SourceLocationConverter(fileName: request.path, tree: tree)
        var diagnostics: [Diagnostic] = []
        for token in tree.tokens(viewMode: .all) {
            for (trivia, start) in [
                (token.leadingTrivia, token.position), (token.trailingTrivia, token.endPositionBeforeTrailingTrivia),
            ] {
                var position = start
                for piece in trivia {
                    defer { position = position.advanced(by: piece.sourceLength.utf8Length) }
                    switch piece {
                    case .lineComment, .blockComment, .docLineComment, .docBlockComment:
                        if rule.allowHeader && header.present && position.utf8Offset >= headerStart
                            && position.utf8Offset < headerEnd
                        {
                            continue
                        }
                        let location = locations.location(for: position)
                        diagnostics.append(
                            Diagnostic(
                                DiagnosticRequest(
                                    DiagnosticSource(
                                        DiagnosticSourceRequest(
                                            ruleID: RuleID.noComments, severity: .error,
                                            message: "comment is not allowed", file: request.path,
                                            location: SourceLocation(line: location.line, column: location.column)
                                        )))))
                    default: break
                    }
                }
            }
        }
        return diagnostics
    }
}
