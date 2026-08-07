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
