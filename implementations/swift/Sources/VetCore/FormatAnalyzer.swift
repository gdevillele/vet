import Foundation

struct FormatAnalyzeRequest {
    let path: String
    let source: String
    let rule: FormatRule
}

public enum FormatError: Error, CustomStringConvertible, Equatable {
    case missingTool
    case executionFailed(String)

    public var description: String {
        switch self {
        case .missingTool:
            "swift-format not found in PATH; install swift-format to enforce VET008"
        case let .executionFailed(message):
            message
        }
    }
}

enum FormatAnalyzer {
    static func analyze(_ request: FormatAnalyzeRequest) throws -> [Diagnostic] {
        guard request.rule.enabled else {
            return []
        }

        guard let executable = resolveExecutable("swift-format") else {
            throw FormatError.missingTool
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = [
            "lint",
            "--strict",
            "--no-color-diagnostics",
            "--assume-filename", request.path,
            "-",
        ]

        let stdinPipe = Pipe()
        process.standardInput = stdinPipe
        // Discard formatter output so a large diagnostic stream cannot fill the
        // OS pipe buffer and deadlock waitUntilExit (~64KB). VET008 only needs
        // the exit status: status 0 = formatted, status 1 = style issues.
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice

        do {
            try process.run()
        } catch {
            throw FormatError.executionFailed("failed to run swift-format: \(error)")
        }

        // Write stdin off the waiting path so a full pipe buffer cannot stall us
        // against a child still consuming input.
        let sourceData = request.source.data(using: .utf8) ?? Data()
        let writeGroup = DispatchGroup()
        writeGroup.enter()
        DispatchQueue.global(qos: .userInitiated).async {
            defer {
                try? stdinPipe.fileHandleForWriting.close()
                writeGroup.leave()
            }
            stdinPipe.fileHandleForWriting.write(sourceData)
        }

        writeGroup.wait()
        process.waitUntilExit()

        if process.terminationStatus == 0 {
            return []
        }

        // swift-format lint --strict exits non-zero when style issues are found.
        // Surface a single VET008 diagnostic (Go/Rust style) rather than re-encoding
        // every formatter finding.
        if process.terminationStatus == 1 {
            return [Diagnostic(DiagnosticRequest(DiagnosticSource(DiagnosticSourceRequest(
                ruleID: RuleID.sourceFormat,
                severity: .error,
                message: "file is not swift-format-formatted",
                file: request.path,
                location: SourceLocation(line: 1, column: 1)
            ))))]
        }

        throw FormatError.executionFailed(
            "swift-format exited with status \(process.terminationStatus)"
        )
    }

    static func resolveExecutable(_ name: String) -> String? {
        let pathEnv = ProcessInfo.processInfo.environment["PATH"] ?? ""
        for directory in pathEnv.split(separator: ":") {
            let candidate = (String(directory) as NSString).appendingPathComponent(name)
            if FileManager.default.isExecutableFile(atPath: candidate) {
                return candidate
            }
        }
        return nil
    }
}
