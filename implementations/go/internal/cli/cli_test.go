package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReportsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func rejected(left int, right int) {}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "VET001") {
		t.Fatalf("expected VET001 diagnostic, got %q", stdout.String())
	}
}

func TestRunAllowsConfiguredParameterLimit(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func accepted(left int, right int) {}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-max-function-parameters", "2", dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunAcceptsRecursiveGoPattern(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")

	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	file := filepath.Join(nested, "sample.go")
	source := []byte(`package sample

func rejected(left int, right int) {}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{filepath.Join(dir, "...")},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
