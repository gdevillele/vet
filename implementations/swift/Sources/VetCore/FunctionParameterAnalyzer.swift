import Foundation

struct FunctionAnalyzeRequest {
    let path: String
    let source: String
    let maxParameters: Int?
    let maxBodyLines: Int
    let docstringPolicy: FunctionDocstringPolicy
}

struct FunctionCandidate {
    let name: String
    let nameOffset: Int
    let parameterStart: Int
    let parameterEnd: Int
    let bodyStart: Int?
    let bodyEnd: Int?
    let docstring: Bool
}

enum FunctionParameterAnalyzer {
    static func analyze(_ request: FunctionAnalyzeRequest) -> [Diagnostic] {
        let characters = Array(maskNonCode(request.source))
        let originalCharacters = Array(request.source)
        var diagnostics: [Diagnostic] = []
        var cursor = 0

        while cursor < characters.count {
            guard isKeyword(KeywordRequest(characters: characters, offset: cursor, keyword: "func")) else {
                cursor += 1
                continue
            }

            if let candidate = readFunction(FunctionReadRequest(
                characters: characters,
                originalCharacters: originalCharacters,
                offset: cursor
            )) {
                if let max = request.maxParameters {
                    let count = parameterCount(ParameterCountRequest(
                        characters: characters,
                        start: candidate.parameterStart,
                        end: candidate.parameterEnd
                    ))
                    if count > max {
                        diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                            ruleID: RuleID.maxFunctionParameters,
                            message: "\(candidate.name) has \(count) parameters; maximum allowed is \(max)",
                            path: request.path,
                            source: request.source,
                            offset: candidate.nameOffset
                        )))
                    }
                }

                if request.maxBodyLines > 0,
                   let bodyStart = candidate.bodyStart,
                   let bodyEnd = candidate.bodyEnd {
                    let count = functionBodyLineCount(FunctionBodyLineCountRequest(
                        source: request.source,
                        bodyStart: bodyStart,
                        bodyEnd: bodyEnd
                    ))
                    if count > request.maxBodyLines {
                        diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                            ruleID: RuleID.functionBodyLines,
                            message: "\(candidate.name) body has \(count) lines; maximum allowed is \(request.maxBodyLines)",
                            path: request.path,
                            source: request.source,
                            offset: candidate.nameOffset
                        )))
                    }
                }

                if request.docstringPolicy == .mandatory && !candidate.docstring {
                    diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                        ruleID: RuleID.functionDocstring,
                        message: "\(candidate.name) must have a docstring",
                        path: request.path,
                        source: request.source,
                        offset: candidate.nameOffset
                    )))
                }

                if request.docstringPolicy == .forbidden && candidate.docstring {
                    diagnostics.append(makeDiagnostic(DiagnosticBuildRequest(
                        ruleID: RuleID.functionDocstring,
                        message: "\(candidate.name) must not have a docstring",
                        path: request.path,
                        source: request.source,
                        offset: candidate.nameOffset
                    )))
                }

                cursor = candidate.parameterEnd + 1
            } else {
                cursor += 4
            }
        }

        return diagnostics
    }

    private static func readFunction(_ request: FunctionReadRequest) -> FunctionCandidate? {
        var cursor = request.offset + 4
        cursor = skipSpaces(SpaceSkipRequest(characters: request.characters, offset: cursor))
        let nameOffset = cursor

        while cursor < request.characters.count {
            let character = request.characters[cursor]
            if character == "(" || character == "<" || character == " " || character == "\t" || character == "\n" {
                break
            }
            cursor += 1
        }

        let name = String(request.characters[nameOffset..<cursor]).trimmingCharacters(in: .whitespacesAndNewlines)
        while cursor < request.characters.count && request.characters[cursor] != "(" {
            if request.characters[cursor] == "{" {
                return nil
            }
            cursor += 1
        }

        guard cursor < request.characters.count else {
            return nil
        }

        let parameterStart = cursor + 1
        guard let parameterEnd = findClosingParen(ParenFindRequest(characters: request.characters, offset: cursor)) else {
            return nil
        }
        let bodyStart = findBodyStart(BodyStartFindRequest(characters: request.characters, offset: parameterEnd + 1))
        let bodyEnd = bodyStart.flatMap { start in
            findClosingBrace(BraceFindRequest(characters: request.characters, offset: start))
        }

        return FunctionCandidate(
            name: name.isEmpty ? "function" : name,
            nameOffset: nameOffset,
            parameterStart: parameterStart,
            parameterEnd: parameterEnd,
            bodyStart: bodyStart,
            bodyEnd: bodyEnd,
            docstring: hasFunctionDocstring(DocstringLookupRequest(characters: request.originalCharacters, offset: request.offset))
        )
    }

    private static func parameterCount(_ request: ParameterCountRequest) -> Int {
        var hasContent = false
        var count = 1
        var parenDepth = 0
        var squareDepth = 0
        var braceDepth = 0
        var angleDepth = 0
        var inDefaultValue = false

        for index in request.start..<request.end {
            let character = request.characters[index]
            if character == " " || character == "\t" || character == "\n" || character == "\r" {
                continue
            }

            hasContent = true
            switch character {
            case "(":
                parenDepth += 1
            case ")":
                parenDepth = max(0, parenDepth - 1)
            case "[":
                squareDepth += 1
            case "]":
                squareDepth = max(0, squareDepth - 1)
            case "{":
                braceDepth += 1
            case "}":
                braceDepth = max(0, braceDepth - 1)
            case "<":
                angleDepth += 1
            case ">":
                angleDepth = max(0, angleDepth - 1)
            case "=":
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 && angleDepth == 0 {
                    inDefaultValue = true
                }
            case ",":
                let startsNextParameter = inDefaultValue && looksLikeParameterStart(ParameterStartRequest(
                    characters: request.characters,
                    start: index + 1,
                    end: request.end
                ))
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 && (angleDepth == 0 || startsNextParameter) {
                    count += 1
                    angleDepth = 0
                    inDefaultValue = false
                }
            default:
                break
            }
        }

        return hasContent ? count : 0
    }

    private static func looksLikeParameterStart(_ request: ParameterStartRequest) -> Bool {
        var parenDepth = 0
        var squareDepth = 0
        var braceDepth = 0
        var angleDepth = 0

        for index in request.start..<request.end {
            let character = request.characters[index]
            switch character {
            case "(":
                parenDepth += 1
            case ")":
                if parenDepth == 0 {
                    return false
                }
                parenDepth -= 1
            case "[":
                squareDepth += 1
            case "]":
                squareDepth = max(0, squareDepth - 1)
            case "{":
                braceDepth += 1
            case "}":
                braceDepth = max(0, braceDepth - 1)
            case "<":
                angleDepth += 1
            case ">":
                angleDepth = max(0, angleDepth - 1)
            case ":":
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 && angleDepth == 0 {
                    return true
                }
            case ",", "=":
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 && angleDepth == 0 {
                    return false
                }
            default:
                break
            }
        }

        return false
    }

    private static func findClosingParen(_ request: ParenFindRequest) -> Int? {
        var depth = 0
        var cursor = request.offset

        while cursor < request.characters.count {
            if request.characters[cursor] == "(" {
                depth += 1
            } else if request.characters[cursor] == ")" {
                depth -= 1
                if depth == 0 {
                    return cursor
                }
            }
            cursor += 1
        }

        return nil
    }

    private static func findBodyStart(_ request: BodyStartFindRequest) -> Int? {
        var cursor = request.offset
        while cursor < request.characters.count {
            let character = request.characters[cursor]
            if character == "{" {
                return cursor
            }
            if character == "=" || character == ";" {
                return nil
            }
            cursor += 1
        }

        return nil
    }

    private static func findClosingBrace(_ request: BraceFindRequest) -> Int? {
        var depth = 0
        var cursor = request.offset

        while cursor < request.characters.count {
            if request.characters[cursor] == "{" {
                depth += 1
            } else if request.characters[cursor] == "}" {
                depth -= 1
                if depth == 0 {
                    return cursor
                }
            }
            cursor += 1
        }

        return nil
    }

    private static func skipSpaces(_ request: SpaceSkipRequest) -> Int {
        var cursor = request.offset
        while cursor < request.characters.count {
            let character = request.characters[cursor]
            if character == " " || character == "\t" || character == "\n" || character == "\r" {
                cursor += 1
            } else {
                break
            }
        }
        return cursor
    }
}

func functionBodyLineCount(_ request: FunctionBodyLineCountRequest) -> Int {
    let start = SourceLocations.location(LocationRequest(source: request.source, offset: request.bodyStart)).line
    let end = SourceLocations.location(LocationRequest(source: request.source, offset: request.bodyEnd)).line
    return max(0, end - start - 1)
}

func hasFunctionDocstring(_ request: DocstringLookupRequest) -> Bool {
    let characters = request.characters
    var cursor = skipWhitespaceBackward(characters, from: request.offset - 1)

    while cursor >= 0 {
        if let start = trailingDeclarationModifierStart(characters, end: cursor)
            ?? trailingAttributeStart(characters, end: cursor) {
            cursor = skipWhitespaceBackward(characters, from: start - 1)
            continue
        }
        break
    }

    guard cursor >= 0 else {
        return false
    }

    if isBlockDocCommentEnding(characters, end: cursor) {
        return true
    }

    let lineStart = lineStartIndex(characters, containing: cursor)
    let line = String(characters[lineStart...cursor]).trimmingCharacters(in: .whitespaces)
    return line.hasPrefix("///")
}

/// Declaration modifiers that may appear between a docstring and `func`.
private let functionDeclarationModifiers: Set<String> = [
    "public", "private", "internal", "open", "fileprivate", "package",
    "static", "class", "final", "override", "required",
    "mutating", "nonmutating",
    "optional", "dynamic", "indirect",
    "nonisolated", "distributed",
]

private func skipWhitespaceBackward(_ characters: [Character], from offset: Int) -> Int {
    var cursor = offset
    while cursor >= 0 {
        let character = characters[cursor]
        if character == " " || character == "\t" || character == "\r" || character == "\n" {
            cursor -= 1
        } else {
            break
        }
    }
    return cursor
}

private func lineStartIndex(_ characters: [Character], containing offset: Int) -> Int {
    var lineStart = offset
    while lineStart > 0 && characters[lineStart - 1] != "\n" {
        lineStart -= 1
    }
    return lineStart
}

private func trailingIdentifierStart(_ characters: [Character], end: Int) -> Int? {
    guard end >= 0, isIdentifierPart(characters[end]) else {
        return nil
    }

    var start = end
    while start > 0 && isIdentifierPart(characters[start - 1]) {
        start -= 1
    }
    return start
}

private func trailingDeclarationModifierStart(_ characters: [Character], end: Int) -> Int? {
    guard let start = trailingIdentifierStart(characters, end: end) else {
        return nil
    }
    let name = String(characters[start...end])
    return functionDeclarationModifiers.contains(name) ? start : nil
}

/// Returns the index of `@` when `end` is the last character of a trailing attribute.
private func trailingAttributeStart(_ characters: [Character], end: Int) -> Int? {
    var cursor = end

    if characters[cursor] == ")" {
        var depth = 0
        while cursor >= 0 {
            let character = characters[cursor]
            if character == ")" {
                depth += 1
            } else if character == "(" {
                depth -= 1
                if depth == 0 {
                    break
                }
            }
            cursor -= 1
        }
        guard cursor >= 0 else {
            return nil
        }
        cursor = skipWhitespaceBackward(characters, from: cursor - 1)
    }

    guard let nameStart = trailingIdentifierStart(characters, end: cursor) else {
        return nil
    }
    let atIndex = nameStart - 1
    guard atIndex >= 0, characters[atIndex] == "@" else {
        return nil
    }
    return atIndex
}

private func isBlockDocCommentEnding(_ characters: [Character], end: Int) -> Bool {
    guard end >= 1, characters[end] == "/", characters[end - 1] == "*" else {
        return false
    }

    var cursor = end - 2
    while cursor >= 1 {
        if characters[cursor - 1] == "/" && characters[cursor] == "*" {
            // Doc block comments start with /** (third character is '*').
            return cursor + 1 <= end && characters[cursor + 1] == "*"
        }
        cursor -= 1
    }
    return false
}

struct FunctionReadRequest {
    let characters: [Character]
    let originalCharacters: [Character]
    let offset: Int
}

struct ParameterCountRequest {
    let characters: [Character]
    let start: Int
    let end: Int
}

struct ParameterStartRequest {
    let characters: [Character]
    let start: Int
    let end: Int
}

struct ParenFindRequest {
    let characters: [Character]
    let offset: Int
}

struct BodyStartFindRequest {
    let characters: [Character]
    let offset: Int
}

struct BraceFindRequest {
    let characters: [Character]
    let offset: Int
}

struct SpaceSkipRequest {
    let characters: [Character]
    let offset: Int
}

struct FunctionBodyLineCountRequest {
    let source: String
    let bodyStart: Int
    let bodyEnd: Int
}

struct DocstringLookupRequest {
    let characters: [Character]
    let offset: Int
}

struct KeywordRequest {
    let characters: [Character]
    let offset: Int
    let keyword: String
}
