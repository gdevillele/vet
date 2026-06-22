import Foundation
import Yams

public struct VetConfig: Equatable {
    public var maxFunctionParameters: MaxFunctionParametersRule
    public var sourceFileHeader: SourceFileHeaderRule

    public static func `default`() -> VetConfig {
        VetConfig(
            maxFunctionParameters: MaxFunctionParametersRule(enabled: true, max: 1),
            sourceFileHeader: SourceFileHeaderRule(required: false, minLength: 0, maxLength: 0)
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

    enum CodingKeys: String, CodingKey {
        case maxFunctionParameters = "max-function-parameters"
        case sourceFileHeader = "source-file-header"
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
    }
}
