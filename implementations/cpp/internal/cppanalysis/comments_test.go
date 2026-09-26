package cppanalysis

import (
	"encoding/json"
	"fmt"
	"github.com/gdevillele/vet/implementations/cpp/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestNoCommentsConformance(t *testing.T) {
	fixture := "../../../../spec/conformance/no-comments/cpp"
	source, err := os.ReadFile(filepath.Join(fixture, "comments.cpp"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(fixture, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected []struct{ Line, Column int }
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		for _, headerRequired := range []bool{false, true} {
			cfg := config.Default()
			cfg.Format.Enabled = false
			cfg.SourceFileHeader.Required = headerRequired
			cfg.NoComments = config.NoCommentsRule{Enabled: enabled}
			diagnostics, err := New(cfg).AnalyzeFile(AnalyzeFileRequest{Path: "comments.cpp", Source: source})
			if err != nil {
				t.Fatal(err)
			}
			want := append(expected[:0:0], expected...)
			if !headerRequired {
				want = append([]struct{ Line, Column int }{{1, 1}}, want...)
			}
			if !enabled {
				want = nil
			}
			if len(diagnostics) != len(want) {
				t.Fatalf("enabled=%v headerRequired=%v: got %#v, want %#v", enabled, headerRequired, diagnostics, want)
			}
			for i, d := range diagnostics {
				if d.RuleID != RuleNoComments || d.Line != want[i].Line || d.Column != want[i].Column || d.Message != "comment is not allowed" {
					t.Fatalf("got %#v, want %#v", d, want[i])
				}
			}
		}
	}
}

func TestNoCommentsClangErrors(t *testing.T) {
	cfg := config.Default()
	cfg.Format.Enabled = false
	cfg.NoComments.Enabled = true
	for _, runner := range []*stubRunner{
		{lookPathErr: fmt.Errorf("not installed")},
		{runErr: fmt.Errorf("could not start")},
		{runExitCode: 1, runOutput: []byte("lexer failed")},
		{runOutput: []byte("unexpected output")},
		{},
	} {
		_, err := NewWithRunner(cfg, runner).AnalyzeFile(AnalyzeFileRequest{Path: "sample.c", Source: []byte("int value; // comment\n")})
		if err == nil {
			t.Fatal("expected a Clang scan error")
		}
		cfg.NoComments.Enabled = false
		_, err = NewWithRunner(cfg, runner).AnalyzeFile(AnalyzeFileRequest{Path: "sample.c", Source: []byte("int value; // comment\n")})
		if err != nil {
			t.Fatalf("disabled rule invoked Clang: %v", err)
		}
		cfg.NoComments.Enabled = true
	}
}

func TestNoCommentsCPreprocessor(t *testing.T) {
	cfg := config.Default()
	cfg.Format.Enabled = false
	cfg.NoComments.Enabled = true
	source := "#include <unavailable.h>\n#if 0\n// inactive\n#endif\n#define VALUE /* macro */ 1\nint value; /\\\n/ spliced comment\n"
	diagnostics, err := New(cfg).AnalyzeFile(AnalyzeFileRequest{Path: "sample.c", Source: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 3 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	for i, line := range []int{3, 5, 6} {
		if diagnostics[i].Line != line {
			t.Fatalf("unexpected location: %#v", diagnostics[i])
		}
	}
}
