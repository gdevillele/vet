package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNoCommentsConfigAndCLI(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	cfg := filepath.Join(dir, "vet.yaml")
	if err := os.WriteFile(file, []byte("// Header\npackage sample\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`version: 1
rules:
  format: { enabled: false }
  no-comments: { enabled: false }
  source-file-header: { required: false }
languages:
  go:
    rules:
      no-comments: { enabled: true }
      source-file-header: { required: true }
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		flags []string
		code  int
	}{
		{nil, 0},
		{[]string{"--require-file-header=false"}, 1},
		{[]string{"--require-file-header=false", "--no-comments=false"}, 0},
		{[]string{"--no-comments", "--require-file-header=false"}, 1},
		{[]string{"--no-comments=invalid"}, 2},
		{[]string{"--require-file-header=invalid"}, 2},
	} {
		var stdout, stderr bytes.Buffer
		args := append([]string{"--config", cfg, "--format", "json"}, test.flags...)
		args = append(args, file)
		code := Run(Invocation{Args: args, Stdout: &stdout, Stderr: &stderr})
		if code != test.code {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", test.flags, code, &stdout, &stderr)
		}
		if code == 1 {
			var payload struct {
				Diagnostics []struct {
					RuleID       string `json:"rule_id"`
					Line, Column int
				} `json:"diagnostics"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Diagnostics) != 1 || payload.Diagnostics[0].RuleID != "VET015" || payload.Diagnostics[0].Line != 1 || payload.Diagnostics[0].Column != 1 {
				t.Fatalf("unexpected payload: %s", &stdout)
			}
		}
	}
}
