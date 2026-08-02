package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReportsMissingRequiredHeader(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.c")
	if err := os.WriteFile(file, []byte("int main(void) { return 0; }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-require-file-header", dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET002") {
		t.Fatalf("expected VET002 diagnostic, got %q", stdout.String())
	}
}

func TestRunReportsSourceFileLineLimit(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.cpp")
	source := []byte("int a = 1;\nint b = 2;\nint c = 3;\nint d = 4;\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-max-source-file-lines", "3", dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET005") {
		t.Fatalf("expected VET005 diagnostic, got %q", stdout.String())
	}
}

func TestRunReportsIndentType(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.c")
	source := []byte("int main(void) {\n\treturn 0;\n}\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-indent-type", "spaces", dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET008") {
		t.Fatalf("expected VET008 diagnostic, got %q", stdout.String())
	}
}

func TestRunAcceptsRecursiveCppPattern(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	file := filepath.Join(nested, "sample.hpp")
	if err := os.WriteFile(file, []byte("int main() {\n\treturn 0;\n}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-indent-type", "spaces", filepath.Join(dir, "...")},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET008") {
		t.Fatalf("expected VET008 diagnostic, got %q", stdout.String())
	}
}

func TestCollectCppFilesRecognizesExtensions(t *testing.T) {
	dir := t.TempDir()
	names := []string{"a.c", "b.h", "c.cc", "d.cpp", "e.cxx", "f.hpp", "g.hh", "skip.txt"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("int x;\n"), 0o600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	files, err := collectCppFiles(fileCollectionRequest{Paths: []string{dir}})
	if err != nil {
		t.Fatalf("collectCppFiles returned error: %v", err)
	}
	if len(files) != 7 {
		t.Fatalf("expected 7 C/C++ files, got %d: %#v", len(files), files)
	}
}

func TestCollectCppFilesHandlesDirectorySymlinkCycle(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	sourcePath := filepath.Join(target, "sample.c")
	if err := os.WriteFile(sourcePath, []byte("int x;\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	linkedDirectory := filepath.Join(root, "source")
	if err := os.Symlink(target, linkedDirectory); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	if err := os.Symlink(root, filepath.Join(target, "loop")); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}

	files, err := collectCppFiles(fileCollectionRequest{Paths: []string{root}})
	if err != nil {
		t.Fatalf("collectCppFiles returned error: %v", err)
	}

	expected := filepath.Join(linkedDirectory, "sample.c")
	if len(files) != 1 || files[0] != expected {
		t.Fatalf("expected only %q, got %#v", expected, files)
	}
}

func TestRunRejectsInvalidHeaderLengthBounds(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-min-file-header-length", "20", "-max-file-header-length", "10", "."},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stderr=%q", code, stderr.String())
	}
}

func TestRunPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-version"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != Version {
		t.Fatalf("expected version %q, got %q", Version, stdout.String())
	}
}

func TestRunRejectsUnsupportedRuleFlags(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.c")
	if err := os.WriteFile(file, []byte("int main(void) { return 0; }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cases := [][]string{
		{"-max-function-parameters", "2", dir},
		{"-max-function-body-lines", "5", dir},
		{"-function-docstring-policy", "mandatory", dir},
		{"-casing", dir},
		{"-function-casing", "camelCase", dir},
	}
	for _, args := range cases {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run(Invocation{
			Args:   args,
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if code != 2 {
			t.Fatalf("args %v: expected exit code 2, got %d; stderr=%q", args, code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "not supported for C/C++") {
			t.Fatalf("args %v: expected not-supported message, got %q", args, stderr.String())
		}
	}
}

func TestRunWarnsOnUnsupportedRulesFromConfig(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.c")
	if err := os.WriteFile(file, []byte("int main(void) { return 0; }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	configPath := filepath.Join(dir, "vet.yaml")
	configData := []byte(`version: 1
rules:
  max-function-parameters:
    enabled: true
    max: 3
`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-c", configPath, file},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "ignoring unsupported C/C++ rule settings") {
		t.Fatalf("expected unsupported-rule warning, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "max-function-parameters") {
		t.Fatalf("expected max-function-parameters in warning, got %q", stderr.String())
	}
}
