import Foundation
import XCTest
@testable import VetCore

final class ConfigTests: XCTestCase {
    func testLoadFileAppliesRuleConfig() throws {
        let directory = temporaryDirectory()
        let configPath = directory.appendingPathComponent("vet.yaml")
        let yaml = """
        version: 1
        rules:
          max-function-parameters:
            enabled: false
            max: 3
          source-file-header:
            required: true
            min-length: 10
            max-length: 80
          max-source-file-lines:
            max: 100
          max-function-body-lines:
            max: 12
          function-docstring:
            policy: mandatory
          indent:
            type: spaces
            width: 4
          casing:
            enabled: true
            functions: camelCase
            variables: snake_case
            types: UpperCamelCase
            constants: SNAKE_CASE_FULL_CAPS
            ignore-names:
              - generated_name
            ignore-patterns:
              - "^Test[A-Z]"
        """

        try yaml.write(to: configPath, atomically: true, encoding: .utf8)

        let config = try ConfigLoader.load(ConfigLoadRequest(
            path: configPath.path,
            base: .default()
        ))

        XCTAssertFalse(config.maxFunctionParameters.enabled)
        XCTAssertEqual(config.maxFunctionParameters.max, 3)
        XCTAssertTrue(config.sourceFileHeader.required)
        XCTAssertEqual(config.sourceFileHeader.minLength, 10)
        XCTAssertEqual(config.sourceFileHeader.maxLength, 80)
        XCTAssertEqual(config.sourceFileLines.max, 100)
        XCTAssertEqual(config.functionBodyLines.max, 12)
        XCTAssertEqual(config.functionDocstring.policy, .mandatory)
        XCTAssertEqual(config.indent.type, .spaces)
        XCTAssertEqual(config.indent.width, 4)
        XCTAssertTrue(config.casing.enabled)
        XCTAssertEqual(config.casing.functions, .camelCase)
        XCTAssertEqual(config.casing.variables, .snakeCase)
        XCTAssertEqual(config.casing.types, .upperCamelCase)
        XCTAssertEqual(config.casing.constants, .snakeUpperCase)
        XCTAssertEqual(config.casing.ignoreNames, ["generated_name"])
        XCTAssertEqual(config.casing.ignorePatterns, ["^Test[A-Z]"])
    }

    func testLoadFileAppliesLanguageOverrides() throws {
        let directory = temporaryDirectory()
        let configPath = directory.appendingPathComponent("vet.yaml")
        let yaml = """
        version: 1
        rules:
          max-function-parameters:
            enabled: true
            max: 3
          indent:
            type: tabs
            width: 0
          casing:
            enabled: false
            functions: language-default
        languages:
          go:
            rules:
              max-function-parameters:
                max: 2
              casing:
                enabled: true
                functions: camelCase
          swift:
            files:
              - Sources/**/*.swift
            exclude:
              - "**/*Tests.swift"
            rules:
              max-function-parameters:
                max: 5
              indent:
                type: spaces
                width: 4
        """

        try yaml.write(to: configPath, atomically: true, encoding: .utf8)

        let config = try ConfigLoader.load(ConfigLoadRequest(
            path: configPath.path,
            base: .default(),
            language: "swift"
        ))

        XCTAssertEqual(config.maxFunctionParameters.max, 5)
        XCTAssertEqual(config.fileSelection.files, ["Sources/**/*.swift"])
        XCTAssertEqual(config.fileSelection.exclude, ["**/*Tests.swift"])
        XCTAssertEqual(config.indent.type, .spaces)
        XCTAssertEqual(config.indent.width, 4)
        XCTAssertFalse(config.casing.enabled)
        XCTAssertEqual(config.casing.functions, .languageDefault)
    }

    func testLoadFileIgnoresLanguageOverridesWithoutLanguage() throws {
        let directory = temporaryDirectory()
        let configPath = directory.appendingPathComponent("vet.yaml")
        let yaml = """
        version: 1
        rules:
          max-function-parameters:
            max: 3
        languages:
          swift:
            rules:
              max-function-parameters:
                max: 5
        """

        try yaml.write(to: configPath, atomically: true, encoding: .utf8)

        let config = try ConfigLoader.load(ConfigLoadRequest(
            path: configPath.path,
            base: .default()
        ))

        XCTAssertEqual(config.maxFunctionParameters.max, 3)
    }

    func testValidateRejectsInvalidHeaderBounds() {
        var config = VetConfig.default()
        config.sourceFileHeader.minLength = 20
        config.sourceFileHeader.maxLength = 10

        XCTAssertThrowsError(try ConfigLoader.validate(config))
    }

    func testValidateRejectsInvalidLineBounds() {
        var config = VetConfig.default()
        config.sourceFileLines.max = -1
        XCTAssertThrowsError(try ConfigLoader.validate(config))

        config = .default()
        config.functionBodyLines.max = -1
        XCTAssertThrowsError(try ConfigLoader.validate(config))
    }

    func testValidateRejectsInvalidIndentWidth() {
        var config = VetConfig.default()
        config.indent.width = -1

        XCTAssertThrowsError(try ConfigLoader.validate(config))
    }

    func testValidateRejectsInvalidCasingIgnorePattern() {
        var config = VetConfig.default()
        config.casing.ignorePatterns = ["["]

        XCTAssertThrowsError(try ConfigLoader.validate(config))
    }

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
