import Foundation
import XCTest

@testable import VetCore

final class CommentTests: XCTestCase {
    func testConfigAndCLI() throws {
        let dir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: dir) }
        let file = dir.appendingPathComponent("sample.swift")
        let cfg = dir.appendingPathComponent("vet.yaml")
        try "// Header\nlet value = 1\n".write(to: file, atomically: true, encoding: .utf8)
        try """
        version: 1
        rules:
          format: { enabled: false }
          source-file-header: { required: false }
          no-comments: { enabled: false }
        languages:
          swift:
            rules:
              no-comments: { enabled: true }
              source-file-header: { required: true }
        """.write(to: cfg, atomically: true, encoding: .utf8)
        let cases: [([String], Int)] = [
            ([], 0),
            (["--require-file-header=false"], 1),
            (["--require-file-header=false", "--no-comments=false"], 0),
            (["--require-file-header=false", "--no-comments"], 1),
            (["--no-comments=invalid"], 2),
            (["--require-file-header=invalid"], 2),
        ]
        for (flags, expected) in cases {
            var stdout = ""
            var stderr = ""
            let code = CLI.run(
                CLIInvocation(
                    arguments: ["--config", cfg.path, "--format", "json"] + flags + [file.path],
                    stdout: { stdout += $0 }, stderr: { stderr += $0 }
                ))
            XCTAssertEqual(code, expected, stdout + stderr)
            if code == 1 {
                let payload = try JSONSerialization.jsonObject(with: Data(stdout.utf8)) as! [String: Any]
                let diagnostics = payload["diagnostics"] as! [[String: Any]]
                XCTAssertEqual(diagnostics.count, 1)
                XCTAssertEqual(diagnostics[0]["rule_id"] as? String, "VET015")
                XCTAssertEqual(diagnostics[0]["line"] as? Int, 1)
                XCTAssertEqual(diagnostics[0]["column"] as? Int, 1)
            }
        }
    }

    func testConformance() throws {
        let fixture = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
            .appendingPathComponent("../../spec/conformance/no-comments/swift")
        let source = try String(contentsOf: fixture.appendingPathComponent("comments.swift"), encoding: .utf8)
        let expected = try JSONDecoder().decode(
            [[String: Int]].self, from: Data(contentsOf: fixture.appendingPathComponent("expected.json")))
        for enabled in [false, true] {
            for headerRequired in [false, true] {
                var config = VetConfig.default()
                config.format.enabled = false
                config.sourceFileHeader.required = headerRequired
                config.noComments = NoCommentsRule(enabled: enabled)
                let diagnostics = try SwiftAnalyzer(config: config).analyzeFile(
                    AnalyzeFileRequest(path: "comments.swift", source: source))
                let want = headerRequired ? expected : [["line": 1, "column": 1]] + expected
                XCTAssertEqual(diagnostics.map { ["line": $0.line, "column": $0.column] }, enabled ? want : [])
                for diagnostic in diagnostics {
                    XCTAssertEqual(diagnostic.ruleID, RuleID.noComments)
                    XCTAssertEqual(diagnostic.message, "comment is not allowed")
                }
            }
        }
    }
}
