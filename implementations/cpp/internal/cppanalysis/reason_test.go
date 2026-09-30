package cppanalysis

import (
	"testing"

	"github.com/gdevillele/vet/implementations/cpp/internal/config"
)

func TestAnalyzeFileAppendsRuleReason(t *testing.T) {
	cfg := config.Default()
	cfg.Format.Enabled = false
	cfg.SourceFileHeader.Required = true
	cfg.SourceFileHeader.Reason = " headers carry the license notice \n"
	cfg.SourceFileLines.Max = 1

	diagnostics, err := New(cfg).AnalyzeFile(AnalyzeFileRequest{
		Path:   "sample.c",
		Source: []byte("int one;\nint two;\n"),
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	messages := map[string]string{}
	for _, item := range diagnostics {
		messages[item.RuleID] = item.Message
	}
	if got, want := messages[RuleSourceFileHeaderRequired], "source file has no header (reason: headers carry the license notice)"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got, want := messages[RuleSourceFileLines], "source file has 2 lines; maximum allowed is 1"; got != want {
		t.Fatalf("expected rule without reason to keep %q, got %q", want, got)
	}
}
