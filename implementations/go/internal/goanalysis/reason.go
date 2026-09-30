package goanalysis

import (
	"strings"

	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
)

// withReasons appends each rule's configured reason so developers know why the rule is enforced.
func (a Analyzer) withReasons(diagnostics []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	for index := range diagnostics {
		if reason := strings.TrimSpace(a.reason(diagnostics[index].RuleID)); reason != "" {
			diagnostics[index].Message += " (reason: " + reason + ")"
		}
	}
	return diagnostics
}

func (a Analyzer) reason(ruleID string) string {
	switch ruleID {
	case RuleMaxFunctionParameters:
		return a.config.MaxFunctionParameters.Reason
	case RuleSourceFileHeaderRequired, RuleSourceFileHeaderMin, RuleSourceFileHeaderMax:
		return a.config.SourceFileHeader.Reason
	case RuleSourceFileLines:
		return a.config.SourceFileLines.Reason
	case RuleFunctionBodyLines:
		return a.config.FunctionBodyLines.Reason
	case RuleFunctionDocstring:
		return a.config.FunctionDocstring.Reason
	case RuleSourceFormat:
		return a.config.Format.Reason
	case RuleFunctionCasing, RuleVariableCasing, RuleTypeCasing, RuleConstantCasing:
		return a.config.Casing.Reason
	case RuleGithubActionsPinned:
		return a.config.GithubActionsPinned.Reason
	case RuleNoComments:
		return a.config.NoComments.Reason
	default:
		return ""
	}
}
