import Foundation
import XCTest
@testable import VetCore

final class CLITests: XCTestCase {
    func testRunReportsDiagnostics() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try """
        func rejected(_ left: Int, _ right: Int) {}
        """.write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: [directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET001"))
        XCTAssertEqual(stderr, "")
    }

    func testRunAllowsConfiguredParameterLimit() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try """
        func accepted(_ left: Int, _ right: Int) {}
        """.write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--max-function-parameters", "2", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunReportsMissingRequiredHeader() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try "func accepted(_ value: Int) {}\n".write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--require-file-header", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET002"))
        XCTAssertEqual(stderr, "")
    }

    func testRunReadsLongConfigFlag() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try "func accepted(_ value: Int) {}\n".write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          source-file-header:
            required: true
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET002"))
        XCTAssertEqual(stderr, "")
    }

    func testRunReadsDefaultConfigFile() throws {
        let directory = temporaryDirectory()
        let originalDirectory = FileManager.default.currentDirectoryPath
        defer {
            _ = FileManager.default.changeCurrentDirectoryPath(originalDirectory)
        }
        XCTAssertTrue(FileManager.default.changeCurrentDirectoryPath(directory.path))

        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try "func accepted(_ value: Int) {}\n".write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          source-file-header:
            required: true
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["."],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET002"))
        XCTAssertEqual(stderr, "")
    }

    func testRunAppliesSwiftLanguageConfigOverride() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try """
        func accepted(_ left: Int, _ right: Int) {}
        """.write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          max-function-parameters:
            max: 1
        languages:
          go:
            rules:
              max-function-parameters:
                max: 1
          swift:
            rules:
              max-function-parameters:
                max: 2
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunUsesSwiftLanguageFileSelectionFromConfig() throws {
        let directory = temporaryDirectory()
        let sourceDirectory = directory.appendingPathComponent("source")
        try FileManager.default.createDirectory(at: sourceDirectory, withIntermediateDirectories: true)

        let included = sourceDirectory.appendingPathComponent("Included.swift")
        try """
        func rejected(_ left: Int, _ right: Int) {}
        """.write(to: included, atomically: true, encoding: .utf8)

        let excluded = sourceDirectory.appendingPathComponent("IgnoredTests.swift")
        try """
        func ignored(_ left: Int, _ right: Int) {}
        """.write(to: excluded, atomically: true, encoding: .utf8)

        let config = directory.appendingPathComponent("vet.yaml")
        try """
        version: 1
        languages:
          swift:
            files:
              - \(sourceDirectory.path)/*.swift
            exclude:
              - "**/*Tests.swift"
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("Included.swift"))
        XCTAssertFalse(stdout.contains("IgnoredTests.swift"))
        XCTAssertEqual(stderr, "")
    }

    func testRunExplicitPathsOverrideConfigFileSelection() throws {
        let directory = temporaryDirectory()
        let configured = directory.appendingPathComponent("Configured.swift")
        try """
        func rejected(_ left: Int, _ right: Int) {}
        """.write(to: configured, atomically: true, encoding: .utf8)

        let explicit = directory.appendingPathComponent("Explicit.swift")
        try """
        func accepted(_ value: Int) {}
        """.write(to: explicit, atomically: true, encoding: .utf8)

        let config = directory.appendingPathComponent("vet.yaml")
        try """
        version: 1
        languages:
          swift:
            files:
              - \(configured.path)
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, explicit.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunReadsShortConfigFlag() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try "func accepted(_ value: Int) {}\n".write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          source-file-header:
            required: true
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["-c", config.path, directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET002"))
        XCTAssertEqual(stderr, "")
    }

    func testRunFlagsOverrideConfigFile() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try """
        // Tiny
        func accepted(_ value: Int) {}
        """.write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          source-file-header:
            min-length: 10
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, "--min-file-header-length", "4", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunRejectsSingleDashLongConfigFlag() {
        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["-config", "vet.yaml"],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 2)
        XCTAssertTrue(stderr.contains("use -c or --config"))
    }

    func testRunReportsNewRuleDiagnostics() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try """
        func missing() {
            print("one")
            print("two")
        }
        """.write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: [
                "--max-source-file-lines", "2",
                "--max-function-body-lines", "1",
                "--function-docstring-policy", "mandatory",
                directory.path,
            ],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        let lines = stdout.trimmingCharacters(in: .whitespacesAndNewlines).split(separator: "\n")
        XCTAssertEqual(lines.count, 1)
        XCTAssertTrue(lines[0].contains("VET005"))
        XCTAssertFalse(stdout.contains("VET006"))
        XCTAssertFalse(stdout.contains("VET007"))
        XCTAssertEqual(stderr, "")
    }

    func testRunReportsAllDiagnosticsAsJSON() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try """
        func missing() {
            print("one")
            print("two")
        }
        """.write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: [
                "--format", "json",
                "--max-source-file-lines", "2",
                "--max-function-body-lines", "1",
                "--function-docstring-policy", "mandatory",
                directory.path,
            ],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertEqual(stderr, "")

        let data = try XCTUnwrap(stdout.data(using: .utf8))
        let object = try JSONSerialization.jsonObject(with: data)
        let payload = try XCTUnwrap(object as? [String: Any])
        let diagnostics = try XCTUnwrap(payload["diagnostics"] as? [[String: Any]])
        let ruleIDs = diagnostics.compactMap { $0["rule_id"] as? String }
        XCTAssertEqual(ruleIDs, ["VET005", "VET006", "VET007"])
    }

    func testRunReportsIndentDiagnostics() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try "func rejected() {\n  print(\"one\")\n}\n".write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--indent-type", "spaces", "--indent-width", "4", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET009"))
        XCTAssertEqual(stderr, "")
    }

    func testRunReportsCasingDiagnostics() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try "func Rejected() {}\n".write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--function-casing", "camelCase", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET010"))
        XCTAssertEqual(stderr, "")
    }

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
