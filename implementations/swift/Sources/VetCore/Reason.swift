import Foundation

extension VetConfig {
    /// Appends the configured reason of the diagnostic's rule so developers know why the rule is enforced.
    func withReason(_ diagnostic: Diagnostic) -> Diagnostic {
        let reason = reason(for: diagnostic.ruleID).trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else {
            return diagnostic
        }
        var result = diagnostic
        result.message += " (reason: \(reason))"
        return result
    }

    private func reason(for ruleID: String) -> String {
        switch ruleID {
        case RuleID.sourceFileHeaderRequired, RuleID.sourceFileHeaderMin, RuleID.sourceFileHeaderMax:
            sourceFileHeader.reason
        case RuleID.sourceFileLines:
            sourceFileLines.reason
        case RuleID.sourceFormat:
            format.reason
        case RuleID.githubActionsPinned:
            githubActionsPinned.reason
        case RuleID.noComments:
            noComments.reason
        default:
            ""
        }
    }
}
