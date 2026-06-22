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
            case ",":
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 {
                    count += 1
                }
            default:
                break
            }
        }

        return hasContent ? count : 0
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

    private static func maskNonCode(_ source: String) -> String {
        var characters = Array(source)
        var cursor = 0

        while cursor < characters.count {
            if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "//")) {
                let result = maskLine(MaskRequest(characters: characters, offset: cursor))
                characters = result.characters
                cursor = result.offset
            } else if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "/*")) {
                let result = maskBlock(MaskRequest(characters: characters, offset: cursor))
                characters = result.characters
                cursor = result.offset
            } else if characters[cursor] == "\"" {
                let result = maskString(MaskRequest(characters: characters, offset: cursor))
                characters = result.characters
                cursor = result.offset
            } else {
                cursor += 1
            }
        }

        return String(characters)
    }

    private static func maskLine(_ request: MaskRequest) -> MaskResult {
        var characters = request.characters
        var cursor = request.offset
        while cursor < characters.count && characters[cursor] != "\n" {
            characters[cursor] = " "
            cursor += 1
        }
        return MaskResult(characters: characters, offset: cursor)
    }

    private static func maskBlock(_ request: MaskRequest) -> MaskResult {
        var characters = request.characters
        var cursor = request.offset
        while cursor + 1 < characters.count {
            if characters[cursor] == "*" && characters[cursor + 1] == "/" {
                characters[cursor] = " "
                characters[cursor + 1] = " "
                return MaskResult(characters: characters, offset: cursor + 2)
            }
            if characters[cursor] != "\n" {
                characters[cursor] = " "
            }
            cursor += 1
        }

        while cursor < characters.count {
            if characters[cursor] != "\n" {
                characters[cursor] = " "
            }
            cursor += 1
        }
        return MaskResult(characters: characters, offset: cursor)
    }

    private static func maskString(_ request: MaskRequest) -> MaskResult {
        var characters = request.characters
        var cursor = request.offset
        var escaped = false

        while cursor < characters.count {
            let character = characters[cursor]
            if character != "\n" {
                characters[cursor] = " "
            }

            if character == "\"" && !escaped && cursor > request.offset {
                return MaskResult(characters: characters, offset: cursor + 1)
            }

            escaped = character == "\\" && !escaped
            if character != "\\" {
                escaped = false
            }
            cursor += 1
        }

        return MaskResult(characters: characters, offset: cursor)
    }
}

func functionBodyLineCount(_ request: FunctionBodyLineCountRequest) -> Int {
    let start = SourceLocations.location(LocationRequest(source: request.source, offset: request.bodyStart)).line
    let end = SourceLocations.location(LocationRequest(source: request.source, offset: request.bodyEnd)).line
    return max(0, end - start - 1)
}

func hasFunctionDocstring(_ request: DocstringLookupRequest) -> Bool {
    var cursor = request.offset - 1
    while cursor >= 0 {
        let character = request.characters[cursor]
        if character == " " || character == "\t" || character == "\r" || character == "\n" {
            cursor -= 1
        } else {
            break
        }
    }

    if cursor < 0 {
        return false
    }

    let prefix = String(request.characters[0...cursor]).trimmingCharacters(in: .whitespacesAndNewlines)
    if prefix.hasSuffix("*/") {
        return prefix.contains("/**")
    }

    var lineStart = cursor
    while lineStart > 0 && request.characters[lineStart - 1] != "\n" {
        lineStart -= 1
    }

    let line = String(request.characters[lineStart...cursor]).trimmingCharacters(in: .whitespaces)
    return line.hasPrefix("///")
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

struct MaskRequest {
    let characters: [Character]
    let offset: Int
}

struct MaskResult {
    let characters: [Character]
    let offset: Int
}
