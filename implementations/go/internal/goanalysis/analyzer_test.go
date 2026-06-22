package goanalysis

import (
	"testing"

	"github.com/gdevillele/vet/implementations/go/internal/config"
)

func TestAnalyzeFileReportsFunctionsWithTooManyParameters(t *testing.T) {
	source := []byte(`package sample

func accepted(value int) {}

func rejected(left int, right int) {}

var _ = func(first string, second string) {}
`)

	diagnostics, err := New(config.Default()).AnalyzeFile(AnalyzeFileRequest{
		Path:   "sample.go",
		Source: source,
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %#v", len(diagnostics), diagnostics)
	}

	if diagnostics[0].RuleID != RuleMaxFunctionParameters {
		t.Fatalf("expected first diagnostic rule %q, got %q", RuleMaxFunctionParameters, diagnostics[0].RuleID)
	}

	if diagnostics[0].Line != 5 {
		t.Fatalf("expected rejected function on line 5, got line %d", diagnostics[0].Line)
	}

	if diagnostics[1].Line != 7 {
		t.Fatalf("expected function literal on line 7, got line %d", diagnostics[1].Line)
	}
}

func TestAnalyzeFileCountsGroupedParameterNames(t *testing.T) {
	source := []byte(`package sample

func rejected(left, right int) {}
`)

	diagnostics, err := New(config.Default()).AnalyzeFile(AnalyzeFileRequest{
		Path:   "sample.go",
		Source: source,
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %#v", len(diagnostics), diagnostics)
	}
}
