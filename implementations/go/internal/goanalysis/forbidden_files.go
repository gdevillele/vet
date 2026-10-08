package goanalysis

import (
	"fmt"

	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
	"github.com/gdevillele/vet/implementations/go/internal/pathpattern"
)

const RuleForbiddenFiles = "VET016"

type AnalyzeForbiddenFileRequest struct {
	Path         string
	RelativePath string
}

func (a Analyzer) AnalyzeForbiddenFile(request AnalyzeForbiddenFileRequest) []diagnostic.Diagnostic {
	if !a.config.ForbiddenFiles.Enabled {
		return nil
	}

	for _, pattern := range a.config.ForbiddenFiles.Exclude {
		if pathpattern.Match(pattern, request.RelativePath) {
			return nil
		}
	}

	for _, pattern := range a.config.ForbiddenFiles.Patterns {
		if !pathpattern.Match(pattern, request.RelativePath) {
			continue
		}

		return a.withReasons([]diagnostic.Diagnostic{{
			RuleID:   RuleForbiddenFiles,
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf(`file type is forbidden (matches "%s")`, pattern),
			File:     request.Path,
			Line:     1,
			Column:   1,
		}})
	}

	return nil
}
