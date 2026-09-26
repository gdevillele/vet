package goanalysis

import "github.com/gdevillele/vet/implementations/go/internal/diagnostic"

func (a Analyzer) checkComments(request fileHeaderCheck) []diagnostic.Diagnostic {
	rule := a.config.NoComments
	if !rule.Enabled {
		return nil
	}
	header := findSourceFileHeader(request.File)
	var diagnostics []diagnostic.Diagnostic
	for _, group := range request.File.Comments {
		for _, comment := range group.List {
			if a.config.SourceFileHeader.Required && header.Present && comment.Pos() >= header.Pos && comment.End() <= header.End {
				continue
			}
			position := request.FileSet.PositionFor(comment.Pos(), false)
			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				RuleID: RuleNoComments, Severity: diagnostic.SeverityError,
				Message: "comment is not allowed", File: request.Path,
				Line: position.Line, Column: position.Column,
			})
		}
	}
	return diagnostics
}
