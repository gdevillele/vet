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

func TestRunReportsMissingRequiredHeader(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte("package sample\n")

	if err := os.WriteFile(file, source, 0o600); err != nil {
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

func TestRunRejectsInvalidHeaderLengthBounds(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-min-file-header-length", "10", "-max-file-header-length", "5"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "sample.go")
	source := []byte("package sample\n")

	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
rules:
  source-file-header:
    required: true
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--config", configPath, dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), "VET002") {
		t.Fatalf("expected VET002 diagnostic, got %q", stdout.String())
	}
}

func TestRunFlagsOverrideConfigFile(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "sample.go")
	source := []byte(`// Tiny
package sample
`)

	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
rules:
  source-file-header:
    min-length: 10
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--config", configPath, "-min-file-header-length", "4", dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunAcceptsShortConfigFlag(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "sample.go")
	source := []byte("package sample\n")

	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
rules:
  source-file-header:
    required: true
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-c", configPath, dir},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), "VET002") {
		t.Fatalf("expected VET002 diagnostic, got %q", stdout.String())
	}
}

func TestRunRejectsSingleDashLongConfigFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-config", "vet.yaml"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stderr.String(), "use -c or --config") {
		t.Fatalf("expected config flag guidance, got %q", stderr.String())
	}
}

func TestRunRejectsConflictingConfigAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"-c", "one.yaml", "--config", "two.yaml"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
