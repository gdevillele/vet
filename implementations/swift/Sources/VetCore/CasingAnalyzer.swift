import Foundation

struct CasingAnalyzeRequest {
    let path: String
    let source: String
    let rule: CasingRule
}

private struct CasingCandidate {
    let name: String
    let offset: Int
    let end: Int
}

private struct CasingKind {
    let ruleID: String
    let name: String
    let style: CasingStyle
    let languageDefault: CasingStyle
}

enum CasingAnalyzer {
    static func analyze(_ request: CasingAnalyzeRequest) -> [Diagnostic] {
        guard request.rule.enabled else {
            return []
        }

        let characters = Array(maskNonCode(request.source))
        var diagnostics: [Diagnostic] = []
        var cursor = 0

        while cursor < characters.count {
            if isKeyword(KeywordRequest(characters: characters, offset: cursor, keyword: "func")) {
                if let candidate = readIdentifier(characters, offset: cursor + 4) {
                    diagnostics.append(contentsOf: checkCandidate(
                        candidate,
                        request: request,
                        kind: CasingKind(
                            ruleID: RuleID.functionCasing,
                            name: "function",
                            style: request.rule.functions,
                            languageDefault: .camelCase
                        )
                    ))
                }
                cursor += 4
                continue
            }

            if isKeyword(KeywordRequest(characters: characters, offset: cursor, keyword: "let")) {
                for candidate in readBindingCandidates(characters, offset: cursor + 3) {
                    diagnostics.append(contentsOf: checkCandidate(
                        candidate,
                        request: request,
                        kind: CasingKind(
                            ruleID: RuleID.constantCasing,
                            name: "constant",
                            style: request.rule.constants,
                            languageDefault: .camelCase
                        )
                    ))
                }
                cursor += 3
                continue
            }

            if isKeyword(KeywordRequest(characters: characters, offset: cursor, keyword: "var")) {
                for candidate in readBindingCandidates(characters, offset: cursor + 3) {
                    diagnostics.append(contentsOf: checkCandidate(
                        candidate,
                        request: request,
                        kind: CasingKind(
                            ruleID: RuleID.variableCasing,
                            name: "variable",
                            style: request.rule.variables,
                            languageDefault: .camelCase
                        )
                    ))
                }
                cursor += 3
                continue
            }

            if let typeKeywordLength = typeKeywordLength(characters, offset: cursor) {
                if let candidate = readIdentifier(characters, offset: cursor + typeKeywordLength) {
                    diagnostics.append(contentsOf: checkCandidate(
                        candidate,
                        request: request,
                        kind: CasingKind(
                            ruleID: RuleID.typeCasing,
                            name: "type",
                            style: request.rule.types,
                            languageDefault: .upperCamelCase
                        )
                    ))
                }
                cursor += typeKeywordLength
                continue
            }

            cursor += 1
        }

        return diagnostics
    }

    private static func checkCandidate(
        _ candidate: CasingCandidate,
        request: CasingAnalyzeRequest,
        kind: CasingKind
    ) -> [Diagnostic] {
        if shouldIgnore(request.rule, name: candidate.name) {
            return []
        }

        let style = kind.style == .languageDefault ? kind.languageDefault : kind.style
        if style == .off || matches(style, name: candidate.name) {
            return []
        }

        return [makeDiagnostic(DiagnosticBuildRequest(
            ruleID: kind.ruleID,
            message: "\(kind.name) \"\(candidate.name)\" must use \(style.rawValue)",
            path: request.path,
            source: request.source,
            offset: candidate.offset
        ))]
    }

    private static func typeKeywordLength(_ characters: [Character], offset: Int) -> Int? {
        for keyword in ["struct", "class", "enum", "protocol", "actor"] {
            if isKeyword(KeywordRequest(characters: characters, offset: offset, keyword: keyword)) {
                return keyword.count
            }
        }
        return nil
    }

    private static func readIdentifier(_ characters: [Character], offset: Int) -> CasingCandidate? {
        var cursor = skipSpaces(characters, offset: offset)
        if cursor >= characters.count {
            return nil
        }

        if characters[cursor] == "`" {
            let nameOffset = cursor + 1
            cursor = nameOffset
            while cursor < characters.count && characters[cursor] != "`" {
                cursor += 1
            }
            if cursor <= nameOffset {
                return nil
            }
            return CasingCandidate(name: String(characters[nameOffset..<cursor]), offset: nameOffset, end: cursor + 1)
        }

        guard isIdentifierStart(characters[cursor]) else {
            return nil
        }

        let nameOffset = cursor
        cursor += 1
        while cursor < characters.count && isIdentifierPart(characters[cursor]) {
            cursor += 1
        }

        return CasingCandidate(name: String(characters[nameOffset..<cursor]), offset: nameOffset, end: cursor)
    }

    private static func readBindingCandidates(_ characters: [Character], offset: Int) -> [CasingCandidate] {
        var candidates: [CasingCandidate] = []
        var cursor = skipSpaces(characters, offset: offset)

        while cursor < characters.count {
            guard let candidate = readIdentifier(characters, offset: cursor) else {
                return candidates
            }

            candidates.append(candidate)
            cursor = skipBindingRest(characters, offset: candidate.end)
            if cursor >= characters.count || characters[cursor] != "," {
                return candidates
            }
            cursor = skipSpaces(characters, offset: cursor + 1)
        }

        return candidates
    }

    private static func skipBindingRest(_ characters: [Character], offset: Int) -> Int {
        var cursor = offset
        var parenDepth = 0
        var squareDepth = 0
        var braceDepth = 0

        while cursor < characters.count {
            let character = characters[cursor]
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
                    return cursor
                }
            case "\n", ";":
                if parenDepth == 0 && squareDepth == 0 && braceDepth == 0 {
                    return cursor
                }
            default:
                break
            }
            cursor += 1
        }

        return cursor
    }

    private static func skipSpaces(_ characters: [Character], offset: Int) -> Int {
        var cursor = offset
        while cursor < characters.count {
            let character = characters[cursor]
            if character == " " || character == "\t" || character == "\n" || character == "\r" {
                cursor += 1
            } else {
                break
            }
        }
        return cursor
    }

    private static func isIdentifierStart(_ character: Character) -> Bool {
        character.isLetter || character == "_"
    }

    private static func matches(_ style: CasingStyle, name: String) -> Bool {
        let pattern: String
        switch style {
        case .camelCase:
            pattern = #"^[a-z][A-Za-z0-9]*$"#
        case .upperCamelCase:
            pattern = #"^[A-Z][A-Za-z0-9]*$"#
        case .snakeCase:
            pattern = #"^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$"#
        case .snakeUpperCase:
            pattern = #"^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$"#
        case .off:
            return true
        case .languageDefault:
            return false
        }

        return name.range(of: pattern, options: .regularExpression) != nil
    }

    private static func shouldIgnore(_ rule: CasingRule, name: String) -> Bool {
        if name == "_" || rule.ignoreNames.contains(name) {
            return true
        }

        for pattern in rule.ignorePatterns {
            if name.range(of: pattern, options: .regularExpression) != nil {
                return true
            }
        }

        return false
    }

    private static func maskNonCode(_ source: String) -> String {
        var characters = Array(source)
        var cursor = 0

        while cursor < characters.count {
            if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "//")) {
                let result = maskLine(characters, offset: cursor)
                characters = result.characters
                cursor = result.offset
            } else if startsWith(StartsWithRequest(characters: characters, offset: cursor, text: "/*")) {
                let result = maskBlock(characters, offset: cursor)
                characters = result.characters
                cursor = result.offset
            } else if characters[cursor] == "\"" {
                let result = maskString(characters, offset: cursor)
                characters = result.characters
                cursor = result.offset
            } else {
                cursor += 1
            }
        }

        return String(characters)
    }

    private static func maskLine(_ characters: [Character], offset: Int) -> MaskResult {
        var characters = characters
        var cursor = offset
        while cursor < characters.count && characters[cursor] != "\n" {
            characters[cursor] = " "
            cursor += 1
        }
        return MaskResult(characters: characters, offset: cursor)
    }

    private static func maskBlock(_ characters: [Character], offset: Int) -> MaskResult {
        var characters = characters
        var cursor = offset
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

    private static func maskString(_ characters: [Character], offset: Int) -> MaskResult {
        var characters = characters
        var cursor = offset
        var escaped = false

        while cursor < characters.count {
            let character = characters[cursor]
            if character != "\n" {
                characters[cursor] = " "
            }

            if character == "\"" && !escaped && cursor > offset {
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
