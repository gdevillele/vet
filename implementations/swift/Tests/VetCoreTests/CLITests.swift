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

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
