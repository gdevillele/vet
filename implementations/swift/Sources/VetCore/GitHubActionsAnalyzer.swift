import Foundation
import Yams

public struct GitHubActionsAnalyzer {
    private let config: VetConfig

    public init(config: VetConfig) {
        self.config = config
    }

    public func analyzeFile(_ request: AnalyzeWorkflowFileRequest) throws -> [Diagnostic] {
        guard config.githubActionsPinned.enabled else {
            return []
        }

        guard let document = try compose(yaml: request.source) else {
            return []
        }

        guard let jobs = mappingValue(document, "jobs"),
              let jobsMapping = jobs.mapping else {
            return []
        }

        var diagnostics: [Diagnostic] = []
        for (_, job) in jobsMapping {
            guard let steps = mappingValue(job, "steps"),
                  let sequence = steps.sequence else {
                continue
            }

            for step in sequence {
                guard let uses = mappingValue(step, "uses"),
                      let action = uses.string,
                      !githubActionPinned(action) else {
                    continue
                }

                let mark = uses.mark
                diagnostics.append(Diagnostic(DiagnosticRequest(DiagnosticSource(DiagnosticSourceRequest(
                    ruleID: RuleID.githubActionsPinned,
                    severity: .error,
                    message: "GitHub action \(String(reflecting: action)) must be pinned to a full-length commit SHA",
                    file: request.path,
                    location: SourceLocation(line: mark?.line ?? 1, column: mark?.column ?? 1)
                )))))
            }
        }

        return diagnostics
    }
}

private func mappingValue(_ node: Node, _ key: String) -> Node? {
    guard let mapping = node.mapping else {
        return nil
    }

    for pair in mapping where pair.key.string == key {
        return pair.value
    }
    return nil
}

private func githubActionPinned(_ action: String) -> Bool {
    if action.hasPrefix("./") || action.hasPrefix("docker://") {
        return true
    }

    guard let atIndex = action.lastIndex(of: "@") else {
        return false
    }
    let reference = action[action.index(after: atIndex)...]
    return reference.count == 40 && reference.allSatisfy(\.isHexDigit)
}
