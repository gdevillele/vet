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
    }

    func testValidateRejectsInvalidHeaderBounds() {
        var config = VetConfig.default()
        config.sourceFileHeader.minLength = 20
        config.sourceFileHeader.maxLength = 10

        XCTAssertThrowsError(try ConfigLoader.validate(config))
    }

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
