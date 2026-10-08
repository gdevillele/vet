import Foundation
import XCTest

@testable import VetCore

final class ForbiddenFilesTests: XCTestCase {
    private let fixture = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        .appendingPathComponent("../../spec/conformance/forbidden-files")
        .standardizedFileURL

    func testLoadFileAppliesForbiddenFilesConfig() throws {
        let defaults = VetConfig.default().forbiddenFiles
        XCTAssertFalse(defaults.enabled)
        XCTAssertEqual(defaults.patterns, [])
        XCTAssertEqual(defaults.exclude, [])

        let config = try load("""
        version: 1
        rules:
          forbidden-files:
            enabled: true
            patterns: ["**/*.py", "**/pyproject.toml"]
            exclude: ["design/archive/**"]
            reason: Use Go for tooling and checks
        """)

        XCTAssertEqual(config.forbiddenFiles, ForbiddenFilesRule(
            enabled: true,
            patterns: ["**/*.py", "**/pyproject.toml"],
            exclude: ["design/archive/**"],
            reason: "Use Go for tooling and checks"
        ))
    }

    func testLoadFileRejectsInvalidForbiddenFilesConfig() throws {
        let cases: [(String, String)] = [
            ("languages:\n  rust:\n    rules:\n      forbidden-files:\n        enabled: true\n  go:\n    rules:\n      forbidden-files:\n        enabled: true\n", "languages.go.rules.forbidden-files is not supported; forbidden-files is repo-wide and must be set under top-level rules"),
            ("rules:\n  forbidden-files:\n    paths: [\"**/*.py\"]\n", "unknown field paths"),
            ("rules:\n  forbidden-files:\n    patterns: [\"\"]\n", "forbidden-files.patterns must not contain empty patterns"),
            ("rules:\n  forbidden-files:\n    exclude: [\" \"]\n", "forbidden-files.exclude must not contain empty patterns"),
        ]
        for (yaml, message) in cases {
            XCTAssertThrowsError(try load("version: 1\n" + yaml)) { error in
                XCTAssertTrue(String(describing: error).contains(message), "\(error)")
            }
        }
    }

    func testAnalyzerMatchesFirstPatternAfterExclude() {
        var config = VetConfig.default()
        config.forbiddenFiles = ForbiddenFilesRule(
            enabled: true,
            patterns: ["**/*.py", "**/pyproject.toml", "**/requirements*.txt"],
            exclude: ["design/archive/**"]
        )
        let analyzer = ForbiddenFilesAnalyzer(config: config)
        let cases: [(String, String?)] = [
            ("foo/bar.py", "**/*.py"),
            ("pyproject.toml", "**/pyproject.toml"),
            ("tools/requirements-dev.txt", "**/requirements*.txt"),
            ("design/archive/old/tool.py", nil),
            ("foo/bar.py.txt", nil),
        ]
        for (path, pattern) in cases {
            let diagnostics = analyzer.analyzeFile(AnalyzeForbiddenFileRequest(path: "./" + path, relativePath: path))
            guard let pattern else {
                XCTAssertEqual(diagnostics, [], path)
                continue
            }
            XCTAssertEqual(diagnostics.count, 1, path)
            XCTAssertEqual(diagnostics.first?.ruleID, "VET016")
            XCTAssertEqual(diagnostics.first?.severity, .error)
            XCTAssertEqual(diagnostics.first?.message, "file type is forbidden (matches \"\(pattern)\")")
            XCTAssertEqual(diagnostics.first?.file, "./" + path)
            XCTAssertEqual(diagnostics.first?.line, 1)
            XCTAssertEqual(diagnostics.first?.column, 1)
        }

        config.forbiddenFiles.enabled = false
        XCTAssertEqual(ForbiddenFilesAnalyzer(config: config).analyzeFile(
            AnalyzeForbiddenFileRequest(path: "foo/bar.py", relativePath: "foo/bar.py")), [])
    }

    func testConformance() throws {
        let expected = try String(contentsOf: fixture.appendingPathComponent("expected.txt"), encoding: .utf8)
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .components(separatedBy: "\n")
        let root = temporaryDirectory()
        try buildLayout(root)
        let config = fixture.appendingPathComponent("vet.yaml").path

        try inDirectory(root) {
            let json = run(["--config", config, "--format", "json"])
            XCTAssertEqual(json.code, 1, json.stdout + json.stderr)
            XCTAssertEqual(json.stderr, "")
            XCTAssertEqual(try renderedDiagnostics(json.stdout), expected)

            let text = run(["--config", config])
            XCTAssertEqual(text.code, 1, text.stderr)
            XCTAssertEqual(text.stdout, expected[0] + "\n")
        }
    }

    func testConfigAndCLI() throws {
        let root = temporaryDirectory()
        for file in ["foo/bar.py", "other/tool.py", "docs/notes.md", "design/archive/old.py", ".git/hooks/hook.py", ".tools/README.md"] {
            try writeFile(root.appendingPathComponent(file))
        }
        try FileManager.default.createSymbolicLink(
            atPath: root.appendingPathComponent(".tools/broken.py").path,
            withDestinationPath: "missing"
        )

        let configs = temporaryDirectory()
        let enabled = try writeConfig(configs, "enabled.yaml", "enabled: true\n    patterns: [\"**/*.py\"]\n    exclude: [\"design/archive/**\"]")
        let disabled = try writeConfig(configs, "disabled.yaml", "enabled: false\n    patterns: [\"**/*.py\"]")
        let empty = try writeConfig(configs, "empty.yaml", "enabled: true")
        let languageOverride = configs.appendingPathComponent("language.yaml").path
        try "version: 1\nlanguages:\n  swift:\n    rules:\n      forbidden-files:\n        enabled: true\n"
            .write(toFile: languageOverride, atomically: true, encoding: .utf8)

        let cases: [([String], Int, [String])] = [
            (["--config", enabled], 1, [".tools/broken.py", "foo/bar.py", "other/tool.py"]),
            (["--config", enabled, "foo"], 1, ["foo/bar.py"]),
            (["--config", enabled, "other/tool.py", "foo/..."], 1, ["foo/bar.py", "other/tool.py"]),
            (["--config", enabled, "design", "docs", ".git"], 0, []),
            (["--config", enabled, "--forbidden-files=false"], 0, []),
            (["--config", disabled], 0, []),
            (["--config", disabled, "--forbidden-files"], 1, [".tools/broken.py", "design/archive/old.py", "foo/bar.py", "other/tool.py"]),
            (["--config", empty], 0, []),
            (["--forbidden-files"], 0, []),
            (["--config", enabled, "--forbidden-files=invalid"], 2, []),
            (["--config", enabled, "missing"], 2, []),
            (["--config", languageOverride], 2, []),
        ]
        try inDirectory(root) {
            for (arguments, code, files) in cases {
                let result = run(["--format", "json", "--check-format=false"] + arguments)
                XCTAssertEqual(result.code, code, "\(arguments): \(result.stdout)\(result.stderr)")
                guard result.code != 2 else {
                    continue
                }

                let payload = try JSONSerialization.jsonObject(with: Data(result.stdout.utf8)) as! [String: Any]
                let diagnostics = payload["diagnostics"] as! [[String: Any]]
                XCTAssertEqual(diagnostics.map { $0["file"] as? String ?? "" }, files, "\(arguments)")
                for diagnostic in diagnostics {
                    XCTAssertEqual(diagnostic["rule_id"] as? String, "VET016")
                    XCTAssertEqual(diagnostic["line"] as? Int, 1)
                    XCTAssertEqual(diagnostic["column"] as? Int, 1)
                }
            }

            let override = run(["--config", languageOverride])
            XCTAssertTrue(override.stderr.contains("languages.swift.rules.forbidden-files is not supported"), override.stderr)
        }
    }

    func testRelativeSlashPath() {
        XCTAssertEqual(relativeSlashPath(workingDirectory: "/repo", path: "/repo/foo/bar.py"), "foo/bar.py")
        XCTAssertEqual(relativeSlashPath(workingDirectory: "/repo/sub", path: "/repo/foo/bar.py"), "../foo/bar.py")
        XCTAssertEqual(relativeSlashPath(workingDirectory: "/repo", path: "foo/bar.py"), "foo/bar.py")
        XCTAssertEqual(joinPath(".", "foo"), "foo")
        XCTAssertEqual(joinPath("./foo/", "bar.py"), "foo/bar.py")
    }

    private func load(_ yaml: String) throws -> VetConfig {
        let path = temporaryDirectory().appendingPathComponent("vet.yaml")
        try yaml.write(to: path, atomically: true, encoding: .utf8)
        return try ConfigLoader.load(ConfigLoadRequest(path: path.path, base: .default(), language: "swift"))
    }

    private func run(_ arguments: [String]) -> (code: Int, stdout: String, stderr: String) {
        var stdout = ""
        var stderr = ""
        let code = CLI.run(CLIInvocation(arguments: arguments, stdout: { stdout += $0 }, stderr: { stderr += $0 }))
        return (code, stdout, stderr)
    }

    private func renderedDiagnostics(_ json: String) throws -> [String] {
        let payload = try JSONSerialization.jsonObject(with: Data(json.utf8)) as! [String: Any]
        return (payload["diagnostics"] as! [[String: Any]]).map { item in
            XCTAssertEqual(item["severity"] as? String, "error")
            return "\(item["file"]!):\(item["line"]!):\(item["column"]!): \(item["rule_id"]!): \(item["message"]!)"
        }
    }

    private func buildLayout(_ root: URL) throws {
        let layout = try String(contentsOf: fixture.appendingPathComponent("layout.txt"), encoding: .utf8)
        for line in layout.components(separatedBy: "\n") {
            let fields = line.split(separator: " ").map(String.init)
            guard let kind = fields.first, !kind.hasPrefix("#") else {
                continue
            }

            let path = root.appendingPathComponent(fields[1])
            try FileManager.default.createDirectory(
                at: path.deletingLastPathComponent(),
                withIntermediateDirectories: true
            )
            switch (kind, fields.count) {
            case ("file", 2):
                try writeFile(path)
            case ("symlink", 3):
                try FileManager.default.createSymbolicLink(atPath: path.path, withDestinationPath: fields[2])
            default:
                XCTFail("invalid layout line \(line)")
            }
        }
    }

    private func writeFile(_ path: URL) throws {
        try FileManager.default.createDirectory(at: path.deletingLastPathComponent(), withIntermediateDirectories: true)
        try "".write(to: path, atomically: true, encoding: .utf8)
    }

    private func writeConfig(_ directory: URL, _ name: String, _ rule: String) throws -> String {
        let path = directory.appendingPathComponent(name).path
        try "version: 1\nrules:\n  forbidden-files:\n    \(rule)\n".write(toFile: path, atomically: true, encoding: .utf8)
        return path
    }

    private func inDirectory(_ directory: URL, _ body: () throws -> Void) throws {
        let originalDirectory = FileManager.default.currentDirectoryPath
        defer {
            _ = FileManager.default.changeCurrentDirectoryPath(originalDirectory)
        }
        XCTAssertTrue(FileManager.default.changeCurrentDirectoryPath(directory.path))
        try body()
    }

    private func temporaryDirectory() -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }
}
