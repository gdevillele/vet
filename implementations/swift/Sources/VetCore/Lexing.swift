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

struct DiagnosticBuildRequest {
    let ruleID: String
    let message: String
    let path: String
    let source: String
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
