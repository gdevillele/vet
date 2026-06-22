import Foundation

public enum RuleID {
    public static let maxFunctionParameters = "VET001"
    public static let sourceFileHeaderRequired = "VET002"
    public static let sourceFileHeaderMin = "VET003"
    public static let sourceFileHeaderMax = "VET004"
}

public struct AnalyzeFileRequest {
    public let path: String
    public let source: String

    public init(path: String, source: String) {
        self.path = path
        self.source = source
    }
}

public struct SwiftAnalyzer {
    private let config: VetConfig

    public init(config: VetConfig) {
        self.config = config
    }

    public func analyzeFile(_ request: AnalyzeFileRequest) -> [Diagnostic] {
        var diagnostics: [Diagnostic] = []
        diagnostics.append(contentsOf: SourceFileHeaderAnalyzer.analyze(HeaderAnalyzeRequest(
            path: request.path,
            source: request.source,
            rule: config.sourceFileHeader
        )))

        if config.maxFunctionParameters.enabled {
            diagnostics.append(contentsOf: FunctionParameterAnalyzer.analyze(FunctionAnalyzeRequest(
                path: request.path,
                source: request.source,
                max: config.maxFunctionParameters.max
            )))
        }

        return diagnostics
    }
}
