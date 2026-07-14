import Foundation
#if os(Linux)
import Glibc
#else
import Darwin
#endif

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
    var configLongPath: String?
    var configShortPath: String?
    var format = "text"
    var maxFunctionParameters: Int?
    var requireFileHeader: Bool?
    var minFileHeaderLength: Int?
    var maxFileHeaderLength: Int?
    var maxSourceFileLines: Int?
    var maxFunctionBodyLines: Int?
    var functionDocstringPolicy: FunctionDocstringPolicy?
    var indentType: IndentType?
    var indentWidth: Int?
    var casingEnabled: Bool?
    var functionCasing: CasingStyle?
    var variableCasing: CasingStyle?
    var typeCasing: CasingStyle?
    var constantCasing: CasingStyle?
    var githubActionsPinned: Bool?
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

struct FileCollectionRequest {
    let paths: [String]
    let exclude: [String]
}

struct WorkflowCollectionRequest {
    let paths: [String]
    let explicit: Bool
}

struct DiagnosticSortRequest {
    let left: Diagnostic
    let right: Diagnostic
}

public enum CLI {
    private static let defaultConfigPath = "vet.yaml"

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
        let selectedConfigPath: String?
        do {
            selectedConfigPath = try selectConfigPath(options)
        } catch {
            invocation.stderr("vet: \(error)\n")
            return 2
        }

        let configPath = selectedConfigPath ?? defaultConfigPathIfPresent()
        if let path = configPath {
            do {
                config = try ConfigLoader.load(ConfigLoadRequest(path: path, base: config, language: "swift"))
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

        let explicitPaths = options.paths
        let selection: FileCollectionRequest
        if explicitPaths.isEmpty {
            selection = FileCollectionRequest(
                paths: config.fileSelection.files.isEmpty ? ["."] : config.fileSelection.files,
                exclude: config.fileSelection.exclude
            )
        } else {
            selection = FileCollectionRequest(paths: explicitPaths, exclude: [])
        }

        let files: [String]
        do {
            files = try collectSwiftFiles(selection)
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
        if config.githubActionsPinned.enabled {
            let workflowFiles: [String]
            do {
                workflowFiles = try collectWorkflowFiles(WorkflowCollectionRequest(
                    paths: explicitPaths,
                    explicit: !explicitPaths.isEmpty
                ))
            } catch {
                invocation.stderr("vet: \(error)\n")
                return 2
            }

            let workflowAnalyzer = GitHubActionsAnalyzer(config: config)
            for file in workflowFiles {
                do {
                    let source = try String(contentsOfFile: file, encoding: .utf8)
                    diagnostics.append(contentsOf: try workflowAnalyzer.analyzeFile(AnalyzeWorkflowFileRequest(
                        path: file,
                        source: source
                    )))
                } catch {
                    invocation.stderr("vet: \(file): \(error)\n")
                    return 2
                }
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

    private static func defaultConfigPathIfPresent() -> String? {
        FileManager.default.fileExists(atPath: defaultConfigPath) ? defaultConfigPath : nil
    }

    private static func parseOptions(_ request: CLIParseRequest) throws -> CLIOptions {
        var options = CLIOptions()
        var cursor = 0
        let arguments = request.arguments

        while cursor < arguments.count {
            let argument = arguments[cursor]
            let (flag, inlineValue) = splitInlineValue(argument)
            switch argument {
            case "--":
                options.paths.append(contentsOf: arguments.dropFirst(cursor + 1))
                return options
            case "-config":
                throw CLIError.message("use -c or --config, not -config")
            default:
                switch flag {
                case "-c":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.configShortPath = parsed.value
                    cursor += parsed.consumed
                case "--config":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.configLongPath = parsed.value
                    cursor += parsed.consumed
                case "-config":
                    throw CLIError.message("use -c or --config, not -config")
                case "--format", "-format":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.format = parsed.value
                    cursor += parsed.consumed
                case "--max-function-parameters", "-max-function-parameters":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.maxFunctionParameters = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--require-file-header", "-require-file-header":
                    options.requireFileHeader = try optionalBool(inlineValue, flag: flag)
                case "--min-file-header-length", "-min-file-header-length":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.minFileHeaderLength = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--max-file-header-length", "-max-file-header-length":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.maxFileHeaderLength = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--max-source-file-lines", "-max-source-file-lines":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.maxSourceFileLines = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--max-function-body-lines", "-max-function-body-lines":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.maxFunctionBodyLines = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--function-docstring-policy", "-function-docstring-policy":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.functionDocstringPolicy = try docstringPolicy(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--indent-type", "-indent-type":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.indentType = try indentType(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--indent-width", "-indent-width":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.indentWidth = try intValue(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--casing", "-casing":
                    options.casingEnabled = try optionalBool(inlineValue, flag: flag)
                case "--function-casing", "-function-casing":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.functionCasing = try casingStyle(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--variable-casing", "-variable-casing":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.variableCasing = try casingStyle(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--type-casing", "-type-casing":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.typeCasing = try casingStyle(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--constant-casing", "-constant-casing":
                    let parsed = try flagValue(FlagValueRequest(arguments: arguments, offset: cursor, inlineValue: inlineValue, flag: flag))
                    options.constantCasing = try casingStyle(parsed.value, flag: flag)
                    cursor += parsed.consumed
                case "--github-actions-pinned", "-github-actions-pinned":
                    options.githubActionsPinned = try optionalBool(inlineValue, flag: flag)
                case "--version", "-version":
                    options.version = try optionalBool(inlineValue, flag: flag)
                default:
                    if argument.hasPrefix("-") {
                        throw CLIError.message("unknown flag \(argument)")
                    }
                    options.paths.append(argument)
                }
            }
            cursor += 1
        }

        return options
    }

    private static func flagValue(_ request: FlagValueRequest) throws -> FlagValueResult {
        if let inlineValue = request.inlineValue {
            return FlagValueResult(value: inlineValue, consumed: 0)
        }

        let valueOffset = request.offset + 1
        guard valueOffset < request.arguments.count else {
            throw CLIError.message("\(request.flag) requires a value")
        }

        return FlagValueResult(value: request.arguments[valueOffset], consumed: 1)
    }

    private static func splitInlineValue(_ argument: String) -> (flag: String, inlineValue: String?) {
        guard let index = argument.firstIndex(of: "=") else {
            return (argument, nil)
        }

        return (String(argument[..<index]), String(argument[argument.index(after: index)...]))
    }

    private static func optionalBool(_ raw: String?, flag: String) throws -> Bool {
        guard let raw else {
            return true
        }

        switch raw {
        case "true":
            return true
        case "false":
            return false
        default:
            throw CLIError.message("\(flag) must be true or false")
        }
    }

    private static func intValue(_ raw: String, flag: String) throws -> Int {
        guard let value = Int(raw) else {
            throw CLIError.message("\(flag) must be an integer")
        }

        return value
    }

    private static func docstringPolicy(_ raw: String, flag: String) throws -> FunctionDocstringPolicy {
        guard let policy = FunctionDocstringPolicy(rawValue: raw) else {
            throw CLIError.message("\(flag) must be forbidden, optional, or mandatory")
        }

        return policy
    }

    private static func indentType(_ raw: String, flag: String) throws -> IndentType {
        guard let type = IndentType(rawValue: raw) else {
            throw CLIError.message("\(flag) must be tabs, spaces, or language-default")
        }

        return type
    }

    private static func casingStyle(_ raw: String, flag: String) throws -> CasingStyle {
        guard let style = CasingStyle(rawValue: raw) else {
            throw CLIError.message("\(flag) must be off, language-default, camelCase, UpperCamelCase, snake_case, or SNAKE_CASE_FULL_CAPS")
        }

        return style
    }

    private static func selectConfigPath(_ options: CLIOptions) throws -> String? {
        if let long = options.configLongPath,
           let short = options.configShortPath,
           long != short {
            throw CLIError.message("-c and --config cannot point to different files")
        }

        return options.configShortPath ?? options.configLongPath
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
        if let max = request.options.maxSourceFileLines {
            config.sourceFileLines.max = max
        }
        if let max = request.options.maxFunctionBodyLines {
            config.functionBodyLines.max = max
        }
        if let policy = request.options.functionDocstringPolicy {
            config.functionDocstring.policy = policy
        }
        if let type = request.options.indentType {
            config.indent.type = type
        }
        if let width = request.options.indentWidth {
            config.indent.width = width
        }
        if let enabled = request.options.casingEnabled {
            config.casing.enabled = enabled
        }
        if let style = request.options.functionCasing {
            config.casing.enabled = true
            config.casing.functions = style
        }
        if let style = request.options.variableCasing {
            config.casing.enabled = true
            config.casing.variables = style
        }
        if let style = request.options.typeCasing {
            config.casing.enabled = true
            config.casing.types = style
        }
        if let style = request.options.constantCasing {
            config.casing.enabled = true
            config.casing.constants = style
        }
        if let enabled = request.options.githubActionsPinned {
            config.githubActionsPinned.enabled = enabled
        }
        return config
    }

    private static func collectSwiftFiles(_ request: FileCollectionRequest) throws -> [String] {
        var collector = FileCollector(exclude: request.exclude)
        for path in request.paths {
            try collector.addPath(normalizePath(path))
        }
        return collector.files.sorted()
    }

    private static func collectWorkflowFiles(_ request: WorkflowCollectionRequest) throws -> [String] {
        var collector = WorkflowFileCollector()
        if !request.explicit {
            try collector.addDefaultWorkflows()
            return collector.files.sorted()
        }

        for path in request.paths {
            try collector.addExplicitPath(normalizePath(path))
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
        guard let diagnostic = request.diagnostics.first else {
            return
        }

        request.writer("\(diagnostic.file):\(diagnostic.line):\(diagnostic.column): \(diagnostic.ruleID): \(diagnostic.message)\n")
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

struct FlagValueRequest {
    let arguments: [String]
    let offset: Int
    let inlineValue: String?
    let flag: String
}

struct FlagValueResult {
    let value: String
    let consumed: Int
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
    var seenFiles: Set<String> = []
    var seenDirectories: Set<String> = []
    var exclude: [String] = []

    mutating func addPath(_ path: String) throws {
        if hasGlobSyntax(path) {
            let matches = try expandGlob(path)
            if matches.isEmpty {
                throw CLIError.message("pattern matched no files: \(path)")
            }
            for match in matches {
                try addPath(match)
            }
            return
        }

        try addConcretePath(path)
    }

    private mutating func addConcretePath(_ path: String) throws {
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
        let resolvedPath = URL(fileURLWithPath: path).resolvingSymlinksInPath().standardizedFileURL.path
        guard seenDirectories.insert(resolvedPath).inserted else {
            return
        }

        let entries = try FileManager.default.contentsOfDirectory(atPath: path)
        for entry in entries {
            if CLI.shouldSkipDirectory(entry) {
                continue
            }
            try addPath((path as NSString).appendingPathComponent(entry))
        }
    }

    private mutating func addFile(_ path: String) {
        guard path.hasSuffix(".swift") && !seenFiles.contains(path) && !isExcluded(path) else {
            return
        }

        seenFiles.insert(path)
        files.append(path)
    }

    private func isExcluded(_ path: String) -> Bool {
        exclude.contains { patternMatches($0, path) }
    }

    private func hasGlobSyntax(_ path: String) -> Bool {
        path.contains("*") || path.contains("?") || path.contains("[")
    }

    private func expandGlob(_ pattern: String) throws -> [String] {
        var results = glob_t()
        let status = glob(pattern, 0, nil, &results)
        defer {
            globfree(&results)
        }

        if status == GLOB_NOMATCH {
            return []
        }
        if status != 0 {
            throw CLIError.message("invalid file pattern: \(pattern)")
        }

        var matches: [String] = []
        guard let paths = results.gl_pathv else {
            return matches
        }
        for index in 0..<Int(results.gl_pathc) {
            if let path = paths[index] {
                matches.append(String(cString: path))
            }
        }
        return matches
    }
}

struct WorkflowFileCollector {
    var files: [String] = []
    var seen: Set<String> = []

    mutating func addDefaultWorkflows() throws {
        let path = ".github/workflows"
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory),
              isDirectory.boolValue else {
            return
        }

        try addWorkflowDirectory(path)
    }

    mutating func addExplicitPath(_ path: String) throws {
        if hasGlobSyntax(path) {
            let matches = try expandGlob(path)
            if matches.isEmpty {
                throw CLIError.message("pattern matched no files: \(path)")
            }
            for match in matches {
                try addExplicitPath(match)
            }
            return
        }

        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) else {
            throw CLIError.message("path does not exist: \(path)")
        }

        if isDirectory.boolValue {
            let nested = (path as NSString).appendingPathComponent(".github/workflows")
            var nestedIsDirectory: ObjCBool = false
            if FileManager.default.fileExists(atPath: nested, isDirectory: &nestedIsDirectory),
               nestedIsDirectory.boolValue {
                try addWorkflowDirectory(nested)
            } else {
                try addWorkflowDirectory(path)
            }
        } else {
            addFile(path)
        }
    }

    private mutating func addWorkflowDirectory(_ path: String) throws {
        let entries = try FileManager.default.contentsOfDirectory(atPath: path)
        for entry in entries {
            var isDirectory: ObjCBool = false
            let child = (path as NSString).appendingPathComponent(entry)
            if FileManager.default.fileExists(atPath: child, isDirectory: &isDirectory),
               isDirectory.boolValue {
                continue
            }
            addFile(child)
        }
    }

    private mutating func addFile(_ path: String) {
        guard isWorkflowFile(path) && !seen.contains(path) else {
            return
        }

        seen.insert(path)
        files.append(path)
    }

    private func hasGlobSyntax(_ path: String) -> Bool {
        path.contains("*") || path.contains("?") || path.contains("[")
    }

    private func expandGlob(_ pattern: String) throws -> [String] {
        var results = glob_t()
        let status = glob(pattern, 0, nil, &results)
        defer {
            globfree(&results)
        }

        if status == GLOB_NOMATCH {
            return []
        }
        if status != 0 {
            throw CLIError.message("invalid file pattern: \(pattern)")
        }

        var matches: [String] = []
        guard let paths = results.gl_pathv else {
            return matches
        }
        for index in 0..<Int(results.gl_pathc) {
            if let path = paths[index] {
                matches.append(String(cString: path))
            }
        }
        return matches
    }
}

func isWorkflowFile(_ path: String) -> Bool {
    path.hasSuffix(".yml") || path.hasSuffix(".yaml")
}

func patternMatches(_ pattern: String, _ filePath: String) -> Bool {
    let normalizedPattern = normalizePattern(pattern)
    let normalizedPath = normalizePattern(filePath)

    if normalizedPattern.isEmpty {
        return false
    }
    if normalizedPattern == "..." {
        return true
    }
    if normalizedPattern.hasSuffix("/...") {
        let prefix = String(normalizedPattern.dropLast(4))
        return normalizedPath == prefix || normalizedPath.hasPrefix(prefix + "/")
    }
    if normalizedPattern.hasSuffix("/**") {
        let prefix = String(normalizedPattern.dropLast(3))
        return normalizedPath == prefix || normalizedPath.hasPrefix(prefix + "/")
    }
    if normalizedPattern.hasPrefix("**/") {
        let suffixPattern = String(normalizedPattern.dropFirst(3))
        if patternMatches(suffixPattern, normalizedPath) {
            return true
        }

        let parts = normalizedPath.split(separator: "/").map(String.init)
        guard parts.count > 1 else {
            return false
        }
        for index in 1..<parts.count {
            if patternMatches(suffixPattern, parts[index...].joined(separator: "/")) {
                return true
            }
        }
        return false
    }

    if fnmatch(normalizedPattern, normalizedPath, FNM_PATHNAME) == 0 {
        return true
    }
    if !normalizedPattern.contains("/") {
        let baseName = (normalizedPath as NSString).lastPathComponent
        if fnmatch(normalizedPattern, baseName, 0) == 0 {
            return true
        }
    }

    return false
}

func normalizePattern(_ value: String) -> String {
    var result = value.replacingOccurrences(of: "\\", with: "/")
    while result.hasPrefix("./") {
        result.removeFirst(2)
    }
    while result.hasSuffix("/") {
        result.removeLast()
    }
    return result
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
