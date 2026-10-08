package goanalysis

import (
	"testing"

	"github.com/gdevillele/vet/implementations/go/internal/config"
	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
)

func TestAnalyzeForbiddenFile(t *testing.T) {
	enabled := config.Default()
	enabled.ForbiddenFiles = config.ForbiddenFilesRule{
		Enabled:  true,
		Patterns: []string{"**/*.py", "**/pyproject.toml", "**/requirements*.txt"},
		Exclude:  []string{"design/archive/**"},
		Reason:   " Use Go for tooling and checks\n",
	}
	disabled := enabled
	disabled.ForbiddenFiles.Enabled = false
	noPatterns := enabled
	noPatterns.ForbiddenFiles.Patterns = nil

	for _, test := range []struct {
		name    string
		cfg     config.Config
		path    string
		message string
	}{
		{"nested", enabled, "foo/bar.py", `file type is forbidden (matches "**/*.py") (reason: Use Go for tooling and checks)`},
		{"root", enabled, "pyproject.toml", `file type is forbidden (matches "**/pyproject.toml") (reason: Use Go for tooling and checks)`},
		{"wildcard", enabled, "tools/requirements-dev.txt", `file type is forbidden (matches "**/requirements*.txt") (reason: Use Go for tooling and checks)`},
		{"excluded", enabled, "design/archive/old/tool.py", ""},
		{"near miss", enabled, "foo/bar.py.txt", ""},
		{"disabled", disabled, "foo/bar.py", ""},
		{"no patterns", noPatterns, "foo/bar.py", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := New(test.cfg).AnalyzeForbiddenFile(AnalyzeForbiddenFileRequest{
				Path:         "./" + test.path,
				RelativePath: test.path,
			})
			if test.message == "" {
				if len(diagnostics) != 0 {
					t.Fatalf("expected no diagnostics, got %#v", diagnostics)
				}
				return
			}

			want := diagnostic.Diagnostic{
				RuleID:   RuleForbiddenFiles,
				Severity: diagnostic.SeverityError,
				Message:  test.message,
				File:     "./" + test.path,
				Line:     1,
				Column:   1,
			}
			if len(diagnostics) != 1 || diagnostics[0] != want {
				t.Fatalf("expected %#v, got %#v", want, diagnostics)
			}
		})
	}
}
