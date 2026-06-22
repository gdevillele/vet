import Foundation

public enum Severity: String, Codable, Equatable {
    case error
}

public struct Diagnostic: Codable, Equatable {
    public let ruleID: String
    public let severity: Severity
    public let message: String
    public let file: String
    public let line: Int
    public let column: Int

    public init(_ request: DiagnosticRequest) {
        ruleID = request.ruleID
        severity = request.severity
        message = request.message
        file = request.file
        line = request.line
        column = request.column
    }

    enum CodingKeys: String, CodingKey {
        case ruleID = "rule_id"
        case severity
        case message
        case file
        case line
        case column
    }
}

public struct DiagnosticRequest {
    let ruleID: String
    let severity: Severity
    let message: String
    let file: String
    let line: Int
    let column: Int

    public init(_ source: DiagnosticSource) {
        ruleID = source.ruleID
        severity = source.severity
        message = source.message
        file = source.file
        line = source.location.line
        column = source.location.column
    }
}

public struct DiagnosticSource {
    let ruleID: String
    let severity: Severity
    let message: String
    let file: String
    let location: SourceLocation

    init(_ request: DiagnosticSourceRequest) {
        ruleID = request.ruleID
        severity = request.severity
        message = request.message
        file = request.file
        location = request.location
    }
}

struct DiagnosticSourceRequest {
    let ruleID: String
    let severity: Severity
    let message: String
    let file: String
    let location: SourceLocation
}
