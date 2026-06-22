import Foundation

public typealias OutputWriter = (String) -> Void

public struct CLIInvocation {
    let arguments: [String]
    let stdout: OutputWriter
    let stderr: OutputWriter

    public init(arguments: [String], stdout: @escaping OutputWriter, stderr: @escaping OutputWriter) {
        self.arguments = arguments
        self.stdout = stdout
        self.stderr = stderr
    }
}

struct CLIOptions {
    var configPath: String?
    var format = "text"
    var maxFunctionParameters: Int?
    var requireFileHeader: Bool?
    var minFileHeaderLength: Int?
    var maxFileHeaderLength: Int?
    var version = false
    var paths: [String] = []
}

struct CLIParseRequest {
    let arguments: [String]
}

struct RenderRequest {
    let diagnostics: [Diagnostic]
    let writer: OutputWriter
}

struct DiagnosticSortRequest {
    let left: Diagnostic
    let right: Diagnostic
}

public enum CLI {
    public static func run(_ invocation: CLIInvocation) -> Int {
        let options: CLIOptions
        do {
            options = try parseOptions(CLIParseRequest(arguments: invocation.arguments))
        } catch {
            invocation.stderr("vet: \(error)\n")
            return 2
        }

        if options.version {
            invocation.stdout("0.1.0-dev\n")
            return 0
        }

        var config = VetConfig.default()
        if let path = options.configPath {
            do {
                config = try ConfigLoader.load(ConfigLoadRequest(path: path, base: config))
            } catch {
                invocation.stderr("vet: \(error)\n")
                return 2
            }
        }

        config = applyOptions(OptionsApplyRequest(options: options, config: config))

        do {
            try ConfigLoader.validate(config)
        } catch {
            invocation.stderr("vet: \(error)\n")
            return 2
        }

        let paths = options.paths.isEmpty ? ["."] : options.paths
        let files: [String]
        do {
            files = try collectSwiftFiles(paths)
        } catch {
            invocation.stderr("vet: \(error)\n")
            return 2
        }

        let analyzer = SwiftAnalyzer(config: config)
        var diagnostics: [Diagnostic] = []
        for file in files {
            do {
                let source = try String(contentsOfFile: file, encoding: .utf8)
                diagnostics.append(contentsOf: analyzer.analyzeFile(AnalyzeFileRequest(path: file, source: source)))
            } catch {
                invocation.stderr("vet: \(file): \(error)\n")
                return 2
            }
        }

        diagnostics.sort { left, right in
            diagnosticComesBefore(DiagnosticSortRequest(left: left, right: right))
        }
        switch options.format {
        case "text":
            renderText(RenderRequest(diagnostics: diagnostics, writer: invocation.stdout))
        case "json":
            do {
                try renderJSON(RenderRequest(diagnostics: diagnostics, writer: invocation.stdout))
            } catch {
                invocation.stderr("vet: failed to write json: \(error)\n")
                return 2
            }
        default:
            invocation.stderr("vet: unsupported format \(options.format)\n")
            return 2
        }

        return diagnostics.isEmpty ? 0 : 1
    }

    private static func parseOptions(_ request: CLIParseRequest) throws -> CLIOptions {
        var options = CLIOptions()
        var cursor = 0
        let arguments = request.arguments

        while cursor < arguments.count {
            let argument = arguments[cursor]
            switch argument {
            case "--":
                options.paths.append(contentsOf: arguments.dropFirst(cursor + 1))
                return options
            case "-config":
                throw CLIError.message("use -c or --config, not -config")
            case "-c", "--config":
                cursor += 1
                options.configPath = try value(ArgumentValueRequest(arguments: arguments, offset: cursor, flag: argument))
            case "--format", "-format":
                cursor += 1
                options.format = try value(ArgumentValueRequest(arguments: arguments, offset: cursor, flag: argument))
            case "--max-function-parameters", "-max-function-parameters":
                cursor += 1
                options.maxFunctionParameters = try intValue(ArgumentValueRequest(arguments: arguments, offset: cursor, flag: argument))
            case "--require-file-header", "-require-file-header":
                options.requireFileHeader = true
            case "--min-file-header-length", "-min-file-header-length":
                cursor += 1
                options.minFileHeaderLength = try intValue(ArgumentValueRequest(arguments: arguments, offset: cursor, flag: argument))
            case "--max-file-header-length", "-max-file-header-length":
                cursor += 1
                options.maxFileHeaderLength = try intValue(ArgumentValueRequest(arguments: arguments, offset: cursor, flag: argument))
            case "--version", "-version":
                options.version = true
            default:
                if argument.hasPrefix("-") {
                    throw CLIError.message("unknown flag \(argument)")
                }
                options.paths.append(argument)
            }
            cursor += 1
        }

        return options
    }

    private static func value(_ request: ArgumentValueRequest) throws -> String {
        guard request.offset < request.arguments.count else {
            throw CLIError.message("\(request.flag) requires a value")
        }

        return request.arguments[request.offset]
    }

    private static func intValue(_ request: ArgumentValueRequest) throws -> Int {
        let raw = try value(request)
        guard let value = Int(raw) else {
            throw CLIError.message("\(request.flag) must be an integer")
        }

        return value
    }

    private static func applyOptions(_ request: OptionsApplyRequest) -> VetConfig {
        var config = request.config
        if let max = request.options.maxFunctionParameters {
            config.maxFunctionParameters.max = max
        }
        if let required = request.options.requireFileHeader {
            config.sourceFileHeader.required = required
        }
        if let minLength = request.options.minFileHeaderLength {
            config.sourceFileHeader.minLength = minLength
        }
        if let maxLength = request.options.maxFileHeaderLength {
            config.sourceFileHeader.maxLength = maxLength
        }
        return config
    }

    private static func collectSwiftFiles(_ paths: [String]) throws -> [String] {
        var collector = FileCollector()
        for path in paths {
            try collector.addPath(normalizePath(path))
        }
        return collector.files.sorted()
    }

    private static func normalizePath(_ path: String) -> String {
        if path == "..." {
            return "."
        }
        if path.hasSuffix("/...") {
            return String(path.dropLast(4))
        }
        return path
    }

    static func shouldSkipDirectory(_ name: String) -> Bool {
        if name == ".git" || name == ".build" || name == "node_modules" {
            return true
        }
        return name.hasPrefix(".")
    }

    private static func diagnosticComesBefore(_ request: DiagnosticSortRequest) -> Bool {
        if request.left.file != request.right.file {
            return request.left.file < request.right.file
        }
        if request.left.line != request.right.line {
            return request.left.line < request.right.line
        }
        if request.left.column != request.right.column {
            return request.left.column < request.right.column
        }
        return request.left.ruleID < request.right.ruleID
    }

    private static func renderText(_ request: RenderRequest) {
        for diagnostic in request.diagnostics {
            request.writer("\(diagnostic.file):\(diagnostic.line):\(diagnostic.column): \(diagnostic.ruleID): \(diagnostic.message)\n")
        }
    }

    private static func renderJSON(_ request: RenderRequest) throws {
        let payload = DiagnosticPayload(diagnostics: request.diagnostics)
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        let data = try encoder.encode(payload)
        request.writer(String(decoding: data, as: UTF8.self))
        request.writer("\n")
    }
}

struct ArgumentValueRequest {
    let arguments: [String]
    let offset: Int
    let flag: String
}

struct OptionsApplyRequest {
    let options: CLIOptions
    let config: VetConfig
}

struct DiagnosticPayload: Encodable {
    let diagnostics: [Diagnostic]
}

struct FileCollector {
    var files: [String] = []
    var seen: Set<String> = []

    mutating func addPath(_ path: String) throws {
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) else {
            throw CLIError.message("path does not exist: \(path)")
        }

        if isDirectory.boolValue {
            try addDirectory(path)
        } else {
            addFile(path)
        }
    }

    private mutating func addDirectory(_ path: String) throws {
        let entries = try FileManager.default.contentsOfDirectory(atPath: path)
        for entry in entries {
            if CLI.shouldSkipDirectory(entry) {
                continue
            }
            try addPath((path as NSString).appendingPathComponent(entry))
        }
    }

    private mutating func addFile(_ path: String) {
        guard path.hasSuffix(".swift") && !seen.contains(path) else {
            return
        }

        seen.insert(path)
        files.append(path)
    }
}

enum CLIError: Error, CustomStringConvertible {
    case message(String)

    var description: String {
        switch self {
        case let .message(message):
            message
        }
    }
}
