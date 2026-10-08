package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const forbiddenFilesFixture = "../../../../spec/conformance/forbidden-files"

type forbiddenFilesPayload struct {
	Diagnostics []struct {
		RuleID   string `json:"rule_id"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
	} `json:"diagnostics"`
}

func TestForbiddenFilesConformance(t *testing.T) {
	fixture, err := filepath.Abs(forbiddenFilesFixture)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(fixture, "expected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	expectedLines := strings.Split(strings.TrimSpace(string(expected)), "\n")

	root := t.TempDir()
	buildForbiddenFilesLayout(t, filepath.Join(fixture, "layout.txt"), root)
	t.Chdir(root)
	config := filepath.Join(fixture, "vet.yaml")

	var stdout, stderr bytes.Buffer
	code := Run(Invocation{Args: []string{"--config", config, "--format", "json"}, Stdout: &stdout, Stderr: &stderr})
	if code != 1 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	var payload forbiddenFilesPayload
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, item := range payload.Diagnostics {
		if item.Severity != "error" {
			t.Fatalf("expected error severity, got %#v", item)
		}
		got = append(got, fmt.Sprintf("%s:%d:%d: %s: %s", item.File, item.Line, item.Column, item.RuleID, item.Message))
	}
	if strings.Join(got, "\n") != strings.Join(expectedLines, "\n") {
		t.Fatalf("expected JSON diagnostics:\n%s\ngot:\n%s", strings.Join(expectedLines, "\n"), strings.Join(got, "\n"))
	}

	stdout.Reset()
	code = Run(Invocation{Args: []string{"--config", config}, Stdout: &stdout, Stderr: &stderr})
	if code != 1 || stdout.String() != expectedLines[0]+"\n" {
		t.Fatalf("expected text output %q, got code=%d stdout=%q stderr=%q", expectedLines[0], code, &stdout, &stderr)
	}
}

func TestForbiddenFilesConfigAndCLI(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for _, file := range []string{"foo/bar.py", "other/tool.py", "docs/notes.md", "design/archive/old.py", ".git/hooks/hook.py"} {
		writeForbiddenFilesFixtureFile(t, file)
	}
	writeForbiddenFilesFixtureFile(t, ".tools/README.md")
	if err := os.Symlink("missing", filepath.Join(root, ".tools", "broken.py")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	enabled := writeForbiddenFilesConfig(t, "enabled.yaml", "enabled: true\n    patterns: [\"**/*.py\"]\n    exclude: [\"design/archive/**\"]")
	disabled := writeForbiddenFilesConfig(t, "disabled.yaml", "enabled: false\n    patterns: [\"**/*.py\"]")
	empty := writeForbiddenFilesConfig(t, "empty.yaml", "enabled: true")

	for _, test := range []struct {
		args  []string
		code  int
		files []string
	}{
		{[]string{"--config", enabled}, 1, []string{".tools/broken.py", "foo/bar.py", "other/tool.py"}},
		{[]string{"--config", enabled, "foo"}, 1, []string{"foo/bar.py"}},
		{[]string{"--config", enabled, "other/tool.py", "foo/..."}, 1, []string{"foo/bar.py", "other/tool.py"}},
		{[]string{"--config", enabled, "design", "docs", ".git"}, 0, nil},
		{[]string{"--config", enabled, "--forbidden-files=false"}, 0, nil},
		{[]string{"--config", disabled}, 0, nil},
		{[]string{"--config", disabled, "--forbidden-files"}, 1, []string{".tools/broken.py", "design/archive/old.py", "foo/bar.py", "other/tool.py"}},
		{[]string{"--config", empty}, 0, nil},
		{[]string{"--forbidden-files"}, 0, nil},
		{[]string{"--config", enabled, "--forbidden-files=invalid"}, 2, nil},
		{[]string{"--config", enabled, "missing"}, 2, nil},
	} {
		var stdout, stderr bytes.Buffer
		args := append([]string{"--format", "json"}, test.args...)
		code := Run(Invocation{Args: args, Stdout: &stdout, Stderr: &stderr})
		if code != test.code {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", test.args, code, &stdout, &stderr)
		}
		if code == 2 {
			continue
		}

		var payload forbiddenFilesPayload
		if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
			t.Fatalf("%v: %v; stdout=%s", test.args, err, &stdout)
		}
		var files []string
		for _, item := range payload.Diagnostics {
			if item.RuleID != "VET016" || item.Line != 1 || item.Column != 1 {
				t.Fatalf("%v: unexpected diagnostic %#v", test.args, item)
			}
			files = append(files, item.File)
		}
		if strings.Join(files, ",") != strings.Join(test.files, ",") {
			t.Fatalf("%v: expected files %v, got %v", test.args, test.files, files)
		}
	}
}

func TestForbiddenFilesRejectsLanguageOverride(t *testing.T) {
	config := filepath.Join(t.TempDir(), "vet.yaml")
	data := "version: 1\nlanguages:\n  cpp:\n    rules:\n      forbidden-files:\n        enabled: true\n"
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(Invocation{Args: []string{"--config", config}, Stdout: &stdout, Stderr: &stderr})
	if code != 2 || !strings.Contains(stderr.String(), "languages.cpp.rules.forbidden-files is not supported") {
		t.Fatalf("expected config error, got code=%d stderr=%q", code, &stderr)
	}
}

func buildForbiddenFilesLayout(t *testing.T, layout string, root string) {
	t.Helper()

	file, err := os.Open(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}

		path := filepath.Join(root, filepath.FromSlash(fields[1]))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		switch {
		case fields[0] == "file" && len(fields) == 2:
			err = os.WriteFile(path, nil, 0o600)
		case fields[0] == "symlink" && len(fields) == 3:
			if err := os.Symlink(filepath.FromSlash(fields[2]), path); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}
		default:
			t.Fatalf("invalid layout line %q", scanner.Text())
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func writeForbiddenFilesFixtureFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeForbiddenFilesConfig(t *testing.T, name string, rule string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	data := "version: 1\nrules:\n  forbidden-files:\n    " + rule + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
