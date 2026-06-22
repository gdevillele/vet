import Foundation

struct HeaderAnalyzeRequest {
    let path: String
    let source: String
    let rule: SourceFileHeaderRule
}

struct HeaderParseResult {
    let present: Bool
    let text: String
    let offset: Int
    let firstCodeOffset: Int
}

enum SourceFileHeaderAnalyzer {
    static func analyze(_ request: HeaderAnalyzeRequest) -> [Diagnostic] {
        let header = parseHeader(request.source)

        if !header.present {
            guard request.rule.required else {
                return []
            }

            return [makeDiagnostic(DiagnosticBuildRequest(
                ruleID: RuleID.sourceFileHeaderRequired,
                message: "source file has no header",
                path: request.path,
                source: request.source,
                offset: header.firstCodeOffset
            ))]
        }

        let length = header.text.count
        var diagnostics: [Diagnostic] = []
        if request.rule.minLength > 0 && length < request.rule.minLength {
            diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                ruleID: RuleID.sourceFileHeaderMin,
                message: "file header has \(length) characters; minimum allowed is \(request.rule.minLength)",
                path: request.path,
                source: request.source,
                offset: header.offset
            )))
        }

        if request.rule.maxLength > 0 && length > request.rule.maxLength {
            diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                ruleID: RuleID.sourceFileHeaderMax,
                message: "file header has \(length) characters; maximum allowed is \(request.rule.maxLength)",
                path: request.path,
                source: request.source,
                offset: header.offset
            )))
        }

        return diagnostics
    }

    static func parseHeader(_ source: String) -> HeaderParseResult {
        let characters = Array(source)
        var cursor = 0

        while cursor < characters.count {
            cursor = skipWhitespace(WhitespaceSkipRequest(characters: characters, offset: cursor))
            if cursor >= characters.count {
                return HeaderParseResult(present: false, text: "", offset: 0, firstCodeOffset: cursor)
            }

            if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "#!")) {
                cursor = skipLine(LineSkipRequest(characters: characters, offset: cursor))
                continue
            }

            if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "//")) {
                let group = readLineCommentGroup(CommentReadRequest(characters: characters, offset: cursor))
                let text = normalizedHeaderText(group.lines)
                if !text.isEmpty {
                    return HeaderParseResult(present: true, text: text, offset: cursor, firstCodeOffset: group.endOffset)
                }
                cursor = group.endOffset
                continue
            }

            if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "/*")) {
                let group = readBlockComment(CommentReadRequest(characters: characters, offset: cursor))
                let text = normalizedHeaderText(group.lines)
                if !text.isEmpty {
                    return HeaderParseResult(present: true, text: text, offset: cursor, firstCodeOffset: group.endOffset)
                }
                cursor = group.endOffset
                continue
            }

            return HeaderParseResult(present: false, text: "", offset: 0, firstCodeOffset: cursor)
        }

        return HeaderParseResult(present: false, text: "", offset: 0, firstCodeOffset: cursor)
    }

    private static func skipWhitespace(_ request: WhitespaceSkipRequest) -> Int {
        var cursor = request.offset
        while cursor < request.characters.count {
            let character = request.characters[cursor]
            if character == " " || character == "\t" || character == "\r" || character == "\n" {
                cursor += 1
            } else {
                break
            }
        }
        return cursor
    }

    private static func skipLine(_ request: LineSkipRequest) -> Int {
        var cursor = request.offset
        while cursor < request.characters.count && request.characters[cursor] != "\n" {
            cursor += 1
        }
        if cursor < request.characters.count {
            cursor += 1
        }
        return cursor
    }

    private static func readLineCommentGroup(_ request: CommentReadRequest) -> CommentGroup {
        var cursor = request.offset
        var lines: [String] = []

        while cursor < request.characters.count &&
            startsWith(StartsWithRequest(characters: request.characters, offset: cursor, text: "//")) {
            cursor += 2
            let lineStart = cursor
            cursor = skipLine(LineSkipRequest(characters: request.characters, offset: cursor))
            lines.append(String(request.characters[lineStart..<trimLineEnd(LineEndTrimRequest(
                characters: request.characters,
                start: lineStart,
                end: cursor
            ))]))
            cursor = skipWhitespace(WhitespaceSkipRequest(characters: request.characters, offset: cursor))
        }

        return CommentGroup(lines: lines, endOffset: cursor)
    }

    private static func readBlockComment(_ request: CommentReadRequest) -> CommentGroup {
        var cursor = request.offset + 2
        let start = cursor

        while cursor + 1 < request.characters.count {
            if request.characters[cursor] == "*" && request.characters[cursor + 1] == "/" {
                let body = String(request.characters[start..<cursor])
                return CommentGroup(lines: body.components(separatedBy: "\n"), endOffset: cursor + 2)
            }
            cursor += 1
        }

        let body = String(request.characters[start..<request.characters.count])
        return CommentGroup(lines: body.components(separatedBy: "\n"), endOffset: request.characters.count)
    }

    private static func trimLineEnd(_ request: LineEndTrimRequest) -> Int {
        var end = request.end
        while end > request.start {
            let character = request.characters[end - 1]
            if character == "\n" || character == "\r" {
                end -= 1
            } else {
                break
            }
        }
        return end
    }

    private static func normalizedHeaderText(_ lines: [String]) -> String {
        let normalized = lines.compactMap { line -> String? in
            let trimmed = normalizeHeaderLine(line)
            if shouldIgnoreHeaderLine(trimmed) {
                return nil
            }
            return trimmed
        }

        return normalized.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private static func normalizeHeaderLine(_ line: String) -> String {
        var trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.hasPrefix("*") {
            trimmed.removeFirst()
        }
        return trimmed.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private static func shouldIgnoreHeaderLine(_ line: String) -> Bool {
        if line.isEmpty {
            return true
        }
        if line.hasPrefix("swift-tools-version:") {
            return true
        }
        return line.hasPrefix("Code generated ") && line.contains("DO NOT EDIT.")
    }
}

struct WhitespaceSkipRequest {
    let characters: [Character]
    let offset: Int
}

struct LineSkipRequest {
    let characters: [Character]
    let offset: Int
}

struct CommentReadRequest {
    let characters: [Character]
    let offset: Int
}

struct LineEndTrimRequest {
    let characters: [Character]
    let start: Int
    let end: Int
}

struct CommentGroup {
    let lines: [String]
    let endOffset: Int
}
