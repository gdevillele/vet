package goanalysis

import (
	"testing"

	"github.com/gdevillele/vet/implementations/go/internal/config"
)

func TestAnalyzeFileAppendsRuleReason(t *testing.T) {
	cfg := config.Default()
	cfg.MaxFunctionParameters.Reason = " wrap related values in a struct \n"
	cfg.SourceFileHeader.Required = true

	diagnostics, err := New(cfg).AnalyzeFile(AnalyzeFileRequest{
		Path:   "sample.go",
		Source: []byte("package sample\n\nfunc rejected(left int, right int) {}\n"),
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	messages := map[string]string{}
	for _, item := range diagnostics {
		messages[item.RuleID] = item.Message
	}
	if got, want := messages[RuleMaxFunctionParameters], "rejected has 2 parameters; maximum allowed is 1 (reason: wrap related values in a struct)"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got, want := messages[RuleSourceFileHeaderRequired], "source file has no header"; got != want {
		t.Fatalf("expected rule without reason to keep %q, got %q", want, got)
	}
}

func TestAnalyzeWorkflowFileAppendsRuleReason(t *testing.T) {
	cfg := config.Default()
	cfg.GithubActionsPinned.Enabled = true
	cfg.GithubActionsPinned.Reason = "tags can be moved to malicious commits"

	diagnostics, err := New(cfg).AnalyzeWorkflowFile(AnalyzeWorkflowFileRequest{
		Path:   ".github/workflows/ci.yml",
		Source: []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"),
	})
	if err != nil {
		t.Fatalf("AnalyzeWorkflowFile returned error: %v", err)
	}

	want := `GitHub action "actions/checkout@v4" must be pinned to a full-length commit SHA (reason: tags can be moved to malicious commits)`
	if len(diagnostics) != 1 || diagnostics[0].Message != want {
		t.Fatalf("expected one diagnostic %q, got %#v", want, diagnostics)
	}
}
