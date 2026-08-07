import Foundation

public enum RuleID {
    public static let maxFunctionParameters = "VET001"
    public static let sourceFileHeaderRequired = "VET002"
    public static let sourceFileHeaderMin = "VET003"
    public static let sourceFileHeaderMax = "VET004"
    public static let sourceFileLines = "VET005"
    public static let functionBodyLines = "VET006"
    public static let functionDocstring = "VET007"
    public static let sourceFormat = "VET008"
    public static let functionCasing = "VET010"
    public static let variableCasing = "VET011"
    public static let typeCasing = "VET012"
    public static let constantCasing = "VET013"
    public static let githubActionsPinned = "VET014"
}

public struct AnalyzeFileRequest {
    public let path: String
    public let source: String

    public init(path: String, source: String) {
        self.path = path
        self.source = source
    }
}

public struct AnalyzeWorkflowFileRequest {
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

    public func analyzeFile(_ request: AnalyzeFileRequest) throws -> [Diagnostic] {
        var diagnostics: [Diagnostic] = []
        diagnostics.append(contentsOf: SourceFileLineAnalyzer.analyze(SourceFileLineAnalyzeRequest(
            path: request.path,
            source: request.source,
            rule: config.sourceFileLines
        )))
        diagnostics.append(contentsOf: try FormatAnalyzer.analyze(FormatAnalyzeRequest(
            path: request.path,
            source: request.source,
            rule: config.format
        )))
        diagnostics.append(contentsOf: SourceFileHeaderAnalyzer.analyze(HeaderAnalyzeRequest(
            path: request.path,
            source: request.source,
            rule: config.sourceFileHeader
        )))
        return diagnostics
    }
}
