package goanalysis

import (
	"encoding/json"
	"github.com/gdevillele/vet/implementations/go/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestNoCommentsConformance(t *testing.T) {
	fixture := "../../../../spec/conformance/no-comments/go"
	source, err := os.ReadFile(filepath.Join(fixture, "comments.go"))
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
		for _, allowHeader := range []bool{false, true} {
			cfg := config.Default()
			cfg.Format.Enabled = false
			cfg.SourceFileHeader.Required = true
			cfg.NoComments = config.NoCommentsRule{Enabled: enabled, AllowHeader: allowHeader}
			diagnostics, err := New(cfg).AnalyzeFile(AnalyzeFileRequest{Path: "comments.go", Source: source})
			if err != nil {
				t.Fatal(err)
			}
			want := append(expected[:0:0], expected...)
			if !allowHeader {
				want = append([]struct{ Line, Column int }{{1, 1}}, want...)
			}
			if !enabled {
				want = nil
			}
			if len(diagnostics) != len(want) {
				t.Fatalf("enabled=%v allowHeader=%v: got %#v, want %#v", enabled, allowHeader, diagnostics, want)
			}
			for i, d := range diagnostics {
				if d.RuleID != RuleNoComments || d.Line != want[i].Line || d.Column != want[i].Column || d.Message != "comment is not allowed" {
					t.Fatalf("got %#v, want %#v", d, want[i])
				}
			}
		}
	}
}
