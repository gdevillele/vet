import Foundation
import Yams

public struct VetConfig: Equatable {
    public var maxFunctionParameters: MaxFunctionParametersRule
    public var sourceFileHeader: SourceFileHeaderRule
    public var sourceFileLines: SourceFileLinesRule
    public var functionBodyLines: FunctionBodyLinesRule
    public var functionDocstring: FunctionDocstringRule
    public var indent: IndentRule
    public var casing: CasingRule

    public static func `default`() -> VetConfig {
        VetConfig(
            maxFunctionParameters: MaxFunctionParametersRule(enabled: true, max: 1),
            sourceFileHeader: SourceFileHeaderRule(required: false, minLength: 0, maxLength: 0),
            sourceFileLines: SourceFileLinesRule(max: 0),
            functionBodyLines: FunctionBodyLinesRule(max: 0),
            functionDocstring: FunctionDocstringRule(policy: .optional),
            indent: IndentRule(type: .languageDefault, width: 0),
            casing: CasingRule(
                enabled: false,
                functions: .languageDefault,
                variables: .languageDefault,
                types: .languageDefault,
                constants: .languageDefault,
                ignoreNames: [],
                ignorePatterns: []
            )
        )
    }
}

public struct MaxFunctionParametersRule: Equatable {
    public var enabled: Bool
    public var max: Int
}

public struct SourceFileHeaderRule: Equatable {
    public var required: Bool
    public var minLength: Int
    public var maxLength: Int
}

public struct SourceFileLinesRule: Equatable {
    public var max: Int
}

public struct FunctionBodyLinesRule: Equatable {
    public var max: Int
}

public enum FunctionDocstringPolicy: String, Codable, Equatable {
    case forbidden
    case optional
    case mandatory
}

public struct FunctionDocstringRule: Equatable {
    public var policy: FunctionDocstringPolicy
}

public enum IndentType: String, Codable, Equatable {
    case tabs
    case spaces
    case languageDefault = "language-default"
}

public struct IndentRule: Equatable {
    public var type: IndentType
    public var width: Int
}

public enum CasingStyle: String, Codable, Equatable {
    case off
    case languageDefault = "language-default"
    case camelCase
    case upperCamelCase = "UpperCamelCase"
    case snakeCase = "snake_case"
    case snakeUpperCase = "SNAKE_CASE_FULL_CAPS"
}

public struct CasingRule: Equatable {
    public var enabled: Bool
    public var functions: CasingStyle
    public var variables: CasingStyle
    public var types: CasingStyle
    public var constants: CasingStyle
    public var ignoreNames: [String]
    public var ignorePatterns: [String]
}

public struct ConfigLoadRequest {
    public let path: String
    public let base: VetConfig

    public init(path: String, base: VetConfig) {
        self.path = path
        self.base = base
    }
}

struct ConfigFile: Decodable {
    let version: Int?
    let rules: RulesFile?
}

struct RulesFile: Decodable {
    let maxFunctionParameters: MaxFunctionParametersFile?
    let sourceFileHeader: SourceFileHeaderFile?
    let sourceFileLines: SourceFileLinesFile?
    let functionBodyLines: FunctionBodyLinesFile?
    let functionDocstring: FunctionDocstringFile?
    let indent: IndentFile?
    let casing: CasingFile?

    enum CodingKeys: String, CodingKey {
        case maxFunctionParameters = "max-function-parameters"
        case sourceFileHeader = "source-file-header"
        case sourceFileLines = "max-source-file-lines"
        case functionBodyLines = "max-function-body-lines"
        case functionDocstring = "function-docstring"
        case indent
        case casing
    }
}

struct MaxFunctionParametersFile: Decodable {
    let enabled: Bool?
    let max: Int?
}

struct SourceFileHeaderFile: Decodable {
    let required: Bool?
    let minLength: Int?
    let maxLength: Int?

    enum CodingKeys: String, CodingKey {
        case required
        case minLength = "min-length"
        case maxLength = "max-length"
    }
}

struct SourceFileLinesFile: Decodable {
    let max: Int?
}

struct FunctionBodyLinesFile: Decodable {
    let max: Int?
}

struct FunctionDocstringFile: Decodable {
    let policy: FunctionDocstringPolicy?
}

struct IndentFile: Decodable {
    let type: IndentType?
    let width: Int?
}

struct CasingFile: Decodable {
    let enabled: Bool?
    let functions: CasingStyle?
    let variables: CasingStyle?
    let types: CasingStyle?
    let constants: CasingStyle?
    let ignoreNames: [String]?
    let ignorePatterns: [String]?

    enum CodingKeys: String, CodingKey {
        case enabled
        case functions
        case variables
        case types
        case constants
        case ignoreNames = "ignore-names"
        case ignorePatterns = "ignore-patterns"
    }
}

public enum ConfigError: Error, CustomStringConvertible, Equatable {
    case unsupportedVersion(Int)
    case invalid(String)

    public var description: String {
        switch self {
        case let .unsupportedVersion(version):
            "unsupported config version \(version)"
        case let .invalid(message):
            message
        }
    }
}

public enum ConfigLoader {
    public static func load(_ request: ConfigLoadRequest) throws -> VetConfig {
        let yaml = try String(contentsOfFile: request.path, encoding: .utf8)
        let document = try YAMLDecoder().decode(ConfigFile.self, from: yaml)

        if let version = document.version, version != 1 {
            throw ConfigError.unsupportedVersion(version)
        }

        var result = request.base
        if let rule = document.rules?.maxFunctionParameters {
            if let enabled = rule.enabled {
                result.maxFunctionParameters.enabled = enabled
            }
            if let max = rule.max {
                result.maxFunctionParameters.max = max
            }
        }

        if let rule = document.rules?.sourceFileHeader {
            if let required = rule.required {
                result.sourceFileHeader.required = required
            }
            if let minLength = rule.minLength {
                result.sourceFileHeader.minLength = minLength
            }
            if let maxLength = rule.maxLength {
                result.sourceFileHeader.maxLength = maxLength
            }
        }

        if let rule = document.rules?.sourceFileLines {
            if let max = rule.max {
                result.sourceFileLines.max = max
            }
        }

        if let rule = document.rules?.functionBodyLines {
            if let max = rule.max {
                result.functionBodyLines.max = max
            }
        }

        if let rule = document.rules?.functionDocstring {
            if let policy = rule.policy {
                result.functionDocstring.policy = policy
            }
        }

        if let rule = document.rules?.indent {
            if let type = rule.type {
                result.indent.type = type
            }
            if let width = rule.width {
                result.indent.width = width
            }
        }

        if let rule = document.rules?.casing {
            if let enabled = rule.enabled {
                result.casing.enabled = enabled
            }
            if let functions = rule.functions {
                result.casing.functions = functions
            }
            if let variables = rule.variables {
                result.casing.variables = variables
            }
            if let types = rule.types {
                result.casing.types = types
            }
            if let constants = rule.constants {
                result.casing.constants = constants
            }
            if let ignoreNames = rule.ignoreNames {
                result.casing.ignoreNames = ignoreNames
            }
            if let ignorePatterns = rule.ignorePatterns {
                result.casing.ignorePatterns = ignorePatterns
            }
        }

        try validate(result)
        return result
    }

    public static func validate(_ config: VetConfig) throws {
        if config.maxFunctionParameters.max < 0 {
            throw ConfigError.invalid("max-function-parameters.max must be zero or greater")
        }
        if config.sourceFileHeader.minLength < 0 {
            throw ConfigError.invalid("source-file-header.min-length must be zero or greater")
        }
        if config.sourceFileHeader.maxLength < 0 {
            throw ConfigError.invalid("source-file-header.max-length must be zero or greater")
        }
        if config.sourceFileHeader.minLength > 0 &&
            config.sourceFileHeader.maxLength > 0 &&
            config.sourceFileHeader.maxLength < config.sourceFileHeader.minLength {
            throw ConfigError.invalid("source-file-header.max-length must be greater than or equal to source-file-header.min-length")
        }
        if config.sourceFileLines.max < 0 {
            throw ConfigError.invalid("max-source-file-lines.max must be zero or greater")
        }
        if config.functionBodyLines.max < 0 {
            throw ConfigError.invalid("max-function-body-lines.max must be zero or greater")
        }
        if config.indent.width < 0 {
            throw ConfigError.invalid("indent.width must be zero or greater")
        }
        for pattern in config.casing.ignorePatterns {
            do {
                _ = try NSRegularExpression(pattern: pattern)
            } catch {
                throw ConfigError.invalid("casing.ignore-patterns contains invalid regex \(pattern)")
            }
        }
    }
}
