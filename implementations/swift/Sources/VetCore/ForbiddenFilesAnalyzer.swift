import Foundation

public struct AnalyzeForbiddenFileRequest {
    public let path: String
    public let relativePath: String

    public init(path: String, relativePath: String) {
        self.path = path
        self.relativePath = relativePath
    }
}

public struct ForbiddenFilesAnalyzer {
    private let config: VetConfig

    public init(config: VetConfig) {
        self.config = config
    }

    public func analyzeFile(_ request: AnalyzeForbiddenFileRequest) -> [Diagnostic] {
        guard config.forbiddenFiles.enabled else {
            return []
        }

        if config.forbiddenFiles.exclude.contains(where: { patternMatches($0, request.relativePath) }) {
            return []
        }

        guard let pattern = config.forbiddenFiles.patterns.first(where: { patternMatches($0, request.relativePath) }) else {
            return []
        }

        return [Diagnostic(DiagnosticRequest(DiagnosticSource(DiagnosticSourceRequest(
            ruleID: RuleID.forbiddenFiles,
            severity: .error,
            message: "file type is forbidden (matches \"\(pattern)\")",
            file: request.path,
            location: SourceLocation(line: 1, column: 1)
        ))))]
    }
}
