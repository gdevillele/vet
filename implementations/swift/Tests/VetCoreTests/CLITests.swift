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

    func testRunHandlesDirectorySymlinkCycle() throws {
        let root = temporaryDirectory()
        let target = temporaryDirectory()
        let file = target.appendingPathComponent("sample.swift")
        try """
        func rejected(_ left: Int, _ right: Int) {}
        """.write(to: file, atomically: true, encoding: .utf8)

        let linkedDirectory = root.appendingPathComponent("source")
        try FileManager.default.createSymbolicLink(at: linkedDirectory, withDestinationURL: target)
        try FileManager.default.createSymbolicLink(
            at: target.appendingPathComponent("loop"),
            withDestinationURL: root
        )

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: [root.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains(linkedDirectory.path))
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

    func testRunRejectsConflictingConfigAliases() throws {
        let directory = temporaryDirectory()
        let first = directory.appendingPathComponent("first.yaml")
        let second = directory.appendingPathComponent("second.yaml")

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["-c", first.path, "--config", second.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 2)
        XCTAssertEqual(stdout, "")
        XCTAssertTrue(stderr.contains("-c and --config cannot point to different files"))
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

    func testRunBooleanFalseFlagOverridesConfigFile() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        let config = directory.appendingPathComponent("vet.yaml")
        try "func accepted() {}\n".write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          source-file-header:
            required: true
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, "--require-file-header=false", directory.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunGithubActionsPinnedFalseFlagOverridesConfigFile() throws {
        let directory = temporaryDirectory()
        let workflow = directory.appendingPathComponent("build.yml")
        let config = directory.appendingPathComponent("vet.yaml")
        try """
        name: test
        jobs:
          build:
            runs-on: ubuntu-latest
            steps:
              - uses: actions/checkout@main
        """.write(to: workflow, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          github-actions-pinned:
            enabled: true
        """.write(to: config, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--config", config.path, "--github-actions-pinned=false", workflow.path],
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

    func testRunAcceptsInlineValueFlags() throws {
        let directory = temporaryDirectory()
        let file = directory.appendingPathComponent("sample.swift")
        try """
        let one = 1
        let two = 2
        """.write(to: file, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: [
                "--format=json",
                "--max-source-file-lines=1",
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
        XCTAssertEqual(diagnostics.count, 1)
        XCTAssertEqual(diagnostics[0]["rule_id"] as? String, "VET005")
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

    func testRunGithubActionsPinnedScansDefaultWorkflows() throws {
        let directory = temporaryDirectory()
        let workflowDirectory = directory.appendingPathComponent(".github/workflows")
        try FileManager.default.createDirectory(at: workflowDirectory, withIntermediateDirectories: true)
        let workflow = workflowDirectory.appendingPathComponent("build.yml")
        try """
        name: test
        jobs:
          build:
            runs-on: ubuntu-latest
            steps:
              - uses: actions/checkout@v5
        """.write(to: workflow, atomically: true, encoding: .utf8)

        let originalDirectory = FileManager.default.currentDirectoryPath
        defer {
            _ = FileManager.default.changeCurrentDirectoryPath(originalDirectory)
        }
        XCTAssertTrue(FileManager.default.changeCurrentDirectoryPath(directory.path))

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--github-actions-pinned"],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET014"))
        XCTAssertTrue(stdout.contains(".github/workflows/build.yml"))
        XCTAssertEqual(stderr, "")
    }

    func testRunGithubActionsPinnedHandlesMissingDefaultWorkflowDirectory() throws {
        let directory = temporaryDirectory()
        let originalDirectory = FileManager.default.currentDirectoryPath
        defer {
            _ = FileManager.default.changeCurrentDirectoryPath(originalDirectory)
        }
        XCTAssertTrue(FileManager.default.changeCurrentDirectoryPath(directory.path))

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--github-actions-pinned"],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 0)
        XCTAssertEqual(stdout, "")
        XCTAssertEqual(stderr, "")
    }

    func testRunGithubActionsPinnedAcceptsExplicitWorkflowPath() throws {
        let directory = temporaryDirectory()
        let workflow = directory.appendingPathComponent("build.yaml")
        try """
        name: test
        jobs:
          build:
            runs-on: ubuntu-latest
            steps:
              - uses: actions/checkout
        """.write(to: workflow, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--github-actions-pinned", workflow.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertTrue(stdout.contains("VET014"))
        XCTAssertEqual(stderr, "")
    }

    func testRunGithubActionsPinnedReportsSortedJSONDiagnostics() throws {
        let directory = temporaryDirectory()
        let first = directory.appendingPathComponent("a.yml")
        let second = directory.appendingPathComponent("z.yml")
        let workflow = """
        name: test
        jobs:
          build:
            runs-on: ubuntu-latest
            steps:
              - uses: actions/checkout@main
        """
        try workflow.write(to: first, atomically: true, encoding: .utf8)
        try workflow.write(to: second, atomically: true, encoding: .utf8)

        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(
            arguments: ["--format", "json", "--github-actions-pinned", second.path, first.path],
            stdout: { stdout += $0 },
            stderr: { stderr += $0 }
        ))

        XCTAssertEqual(code, 1)
        XCTAssertEqual(stderr, "")

        let data = try XCTUnwrap(stdout.data(using: .utf8))
        let object = try JSONSerialization.jsonObject(with: data)
        let payload = try XCTUnwrap(object as? [String: Any])
        let diagnostics = try XCTUnwrap(payload["diagnostics"] as? [[String: Any]])
        XCTAssertEqual(diagnostics.count, 2)
        XCTAssertEqual(diagnostics[0]["rule_id"] as? String, "VET014")
        XCTAssertEqual(diagnostics[1]["rule_id"] as? String, "VET014")
        XCTAssertEqual(diagnostics[0]["file"] as? String, first.path)
        XCTAssertEqual(diagnostics[1]["file"] as? String, second.path)
    }

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
