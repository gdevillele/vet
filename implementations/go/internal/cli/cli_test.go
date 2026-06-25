package cli

import (
	"bytes"
	"encoding/json"
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

func TestRunReadsDefaultConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

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
		Args:   []string{"."},
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

func TestRunAppliesGoLanguageConfigOverride(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func accepted(left int, right int) {}
`)

	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
rules:
  max-function-parameters:
    max: 1
languages:
  go:
    rules:
      max-function-parameters:
        max: 2
  swift:
    rules:
      max-function-parameters:
        max: 1
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

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunUsesGoLanguageFileSelectionFromConfig(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	includedPath := filepath.Join(sourceDir, "included.go")
	includedSource := []byte(`package sample

func rejected(left int, right int) {}
`)
	if err := os.WriteFile(includedPath, includedSource, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	excludedPath := filepath.Join(sourceDir, "ignored_test.go")
	excludedSource := []byte(`package sample

func ignored(left int, right int) {}
`)
	if err := os.WriteFile(excludedPath, excludedSource, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
languages:
  go:
    files:
      - ` + filepath.ToSlash(filepath.Join(sourceDir, "*.go")) + `
    exclude:
      - "**/*_test.go"
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--config", configPath},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "included.go") {
		t.Fatalf("expected included file diagnostic, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "ignored_test.go") {
		t.Fatalf("expected excluded file to be omitted, got %q", stdout.String())
	}
}

func TestRunExplicitPathsOverrideConfigFileSelection(t *testing.T) {
	dir := t.TempDir()
	configuredPath := filepath.Join(dir, "configured.go")
	configuredSource := []byte(`package sample

func rejected(left int, right int) {}
`)
	if err := os.WriteFile(configuredPath, configuredSource, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	explicitPath := filepath.Join(dir, "explicit.go")
	explicitSource := []byte(`package sample

func accepted(value int) {}
`)
	if err := os.WriteFile(explicitPath, explicitSource, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	configPath := filepath.Join(dir, "vet.yaml")
	config := []byte(`version: 1
languages:
  go:
    files:
      - ` + filepath.ToSlash(configuredPath) + `
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--config", configPath, explicitPath},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

func TestRunReportsNewRuleDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func missing() {
	println("one")
	println("two")
}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args: []string{
			"--max-source-file-lines", "2",
			"--max-function-body-lines", "1",
			"--function-docstring-policy", "mandatory",
			dir,
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one text diagnostic, got %d: %q", len(lines), stdout.String())
	}
	if !strings.Contains(lines[0], "VET005") {
		t.Fatalf("expected first sorted diagnostic VET005, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "VET006") || strings.Contains(stdout.String(), "VET007") {
		t.Fatalf("expected default text output to omit later diagnostics, got %q", stdout.String())
	}
}

func TestRunReportsAllDiagnosticsAsJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func missing() {
	println("one")
	println("two")
}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args: []string{
			"--format", "json",
			"--max-source-file-lines", "2",
			"--max-function-body-lines", "1",
			"--function-docstring-policy", "mandatory",
			dir,
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var payload struct {
		Diagnostics []struct {
			RuleID string `json:"rule_id"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v; stdout=%q", err, stdout.String())
	}

	got := make([]string, 0, len(payload.Diagnostics))
	for _, item := range payload.Diagnostics {
		got = append(got, item.RuleID)
	}
	want := []string{"VET005", "VET006", "VET007"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected diagnostics %v, got %v; stdout=%q", want, got, stdout.String())
	}
}

func TestRunReportsIndentDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte("package sample\n\nfunc rejected() {\n  println(\"one\")\n}\n")

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args: []string{
			"--indent-type", "spaces",
			"--indent-width", "4",
			dir,
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), "VET009") {
		t.Fatalf("expected VET009 diagnostic, got %q", stdout.String())
	}
}

func TestRunReportsCasingDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	source := []byte(`package sample

func Rejected() {}
`)

	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args: []string{
			"--function-casing", "camelCase",
			dir,
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), "VET010") {
		t.Fatalf("expected VET010 diagnostic, got %q", stdout.String())
	}
}

func TestRunGithubActionsPinnedScansDefaultWorkflows(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	workflowDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	workflow := []byte(`name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
`)
	if err := os.WriteFile(filepath.Join(workflowDir, "build.yml"), workflow, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--github-actions-pinned"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET014") {
		t.Fatalf("expected VET014 diagnostic, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `.github/workflows/build.yml`) {
		t.Fatalf("expected default workflow path in diagnostic, got %q", stdout.String())
	}
}

func TestRunGithubActionsPinnedHandlesMissingDefaultWorkflowDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--github-actions-pinned"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("expected no output, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunGithubActionsPinnedAcceptsExplicitWorkflowPath(t *testing.T) {
	dir := t.TempDir()
	workflowPath := filepath.Join(dir, "build.yaml")
	workflow := []byte(`name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout
`)
	if err := os.WriteFile(workflowPath, workflow, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--github-actions-pinned", workflowPath},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "VET014") {
		t.Fatalf("expected VET014 diagnostic, got %q", stdout.String())
	}
}

func TestRunGithubActionsPinnedReportsSortedJSONDiagnostics(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.yml")
	second := filepath.Join(dir, "z.yml")
	workflow := []byte(`name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@main
`)
	if err := os.WriteFile(first, workflow, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(second, workflow, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(Invocation{
		Args:   []string{"--format", "json", "--github-actions-pinned", second, first},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var payload struct {
		Diagnostics []struct {
			RuleID string `json:"rule_id"`
			File   string `json:"file"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v; stdout=%q", err, stdout.String())
	}
	if len(payload.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %q", len(payload.Diagnostics), stdout.String())
	}
	if payload.Diagnostics[0].RuleID != "VET014" || payload.Diagnostics[1].RuleID != "VET014" {
		t.Fatalf("expected VET014 diagnostics, got %#v", payload.Diagnostics)
	}
	if payload.Diagnostics[0].File != first || payload.Diagnostics[1].File != second {
		t.Fatalf("expected diagnostics sorted by file, got %#v", payload.Diagnostics)
	}
}
