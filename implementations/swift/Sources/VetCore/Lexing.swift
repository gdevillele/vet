import Foundation

struct StartsWithRequest {
    let characters: [Character]
    let offset: Int
    let text: String
}

func startsWith(_ request: StartsWithRequest) -> Bool {
    let target = Array(request.text)
    guard request.offset + target.count <= request.characters.count else {
        return false
    }

    for index in 0..<target.count {
        if request.characters[request.offset + index] != target[index] {
            return false
        }
    }

    return true
}

func isKeyword(_ request: KeywordRequest) -> Bool {
    guard startsWith(StartsWithRequest(
        characters: request.characters,
        offset: request.offset,
        text: request.keyword
    )) else {
        return false
    }

    let before = request.offset == 0 ? nil : request.characters[request.offset - 1]
    let afterOffset = request.offset + request.keyword.count
    let after = afterOffset >= request.characters.count ? nil : request.characters[afterOffset]
    return !isIdentifierPart(before) && !isIdentifierPart(after)
}

func isIdentifierPart(_ character: Character?) -> Bool {
    guard let character else {
        return false
    }

    return character.isLetter || character.isNumber || character == "_"
}

func maskNonCode(_ source: String) -> String {
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

private func maskLine(_ request: MaskRequest) -> MaskResult {
    var characters = request.characters
    var cursor = request.offset
    while cursor < characters.count && characters[cursor] != "\n" {
        characters[cursor] = " "
        cursor += 1
    }
    return MaskResult(characters: characters, offset: cursor)
}

private func maskBlock(_ request: MaskRequest) -> MaskResult {
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

private func maskString(_ request: MaskRequest) -> MaskResult {
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

struct DiagnosticBuildRequest {
    let ruleID: String
    let message: String
    let path: String
    let source: String
    let offset: Int
}

private struct MaskRequest {
    let characters: [Character]
    let offset: Int
}

private struct MaskResult {
    let characters: [Character]
    let offset: Int
}

func makeDiagnostic(_ request: DiagnosticBuildRequest) -> Diagnostic {
    Diagnostic(DiagnosticRequest(DiagnosticSource(DiagnosticSourceRequest(
        ruleID: request.ruleID,
        severity: .error,
        message: request.message,
        file: request.path,
        location: SourceLocations.location(LocationRequest(source: request.source, offset: request.offset))
    ))))
}
