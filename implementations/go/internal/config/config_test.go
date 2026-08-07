package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileAppliesRuleConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  max-function-parameters:
    enabled: false
    max: 3
  source-file-header:
    required: true
    min-length: 10
    max-length: 80
  max-source-file-lines:
    max: 100
  max-function-body-lines:
    max: 12
  function-docstring:
    policy: mandatory
  format:
    enabled: false
  casing:
    enabled: true
    functions: camelCase
    variables: snake_case
    types: UpperCamelCase
    constants: SNAKE_CASE_FULL_CAPS
    ignore-names:
      - generated_name
    ignore-patterns:
      - "^Test[A-Z]"
  github-actions-pinned:
    enabled: true
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := LoadFile(LoadFileRequest{
		Path: path,
		Base: Default(),
	})
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.MaxFunctionParameters.Enabled {
		t.Fatalf("expected max function parameter rule to be disabled")
	}
	if cfg.MaxFunctionParameters.Max != 3 {
		t.Fatalf("expected max function parameters to be 3, got %d", cfg.MaxFunctionParameters.Max)
	}
	if !cfg.SourceFileHeader.Required {
		t.Fatalf("expected source file header to be required")
	}
	if cfg.SourceFileHeader.MinLength != 10 {
		t.Fatalf("expected min header length 10, got %d", cfg.SourceFileHeader.MinLength)
	}
	if cfg.SourceFileHeader.MaxLength != 80 {
		t.Fatalf("expected max header length 80, got %d", cfg.SourceFileHeader.MaxLength)
	}
	if cfg.SourceFileLines.Max != 100 {
		t.Fatalf("expected max source file lines 100, got %d", cfg.SourceFileLines.Max)
	}
	if cfg.FunctionBodyLines.Max != 12 {
		t.Fatalf("expected max function body lines 12, got %d", cfg.FunctionBodyLines.Max)
	}
	if cfg.FunctionDocstring.Policy != FunctionDocstringMandatory {
		t.Fatalf("expected mandatory docstring policy, got %q", cfg.FunctionDocstring.Policy)
	}
	if cfg.Format.Enabled {
		t.Fatalf("expected format rule to be disabled")
	}
	if !cfg.Casing.Enabled {
		t.Fatalf("expected casing rule to be enabled")
	}
	if cfg.Casing.Functions != CasingCamelCase {
		t.Fatalf("expected function casing camelCase, got %q", cfg.Casing.Functions)
	}
	if cfg.Casing.Variables != CasingSnakeCase {
		t.Fatalf("expected variable casing snake_case, got %q", cfg.Casing.Variables)
	}
	if cfg.Casing.Types != CasingUpperCamelCase {
		t.Fatalf("expected type casing UpperCamelCase, got %q", cfg.Casing.Types)
	}
	if cfg.Casing.Constants != CasingSnakeUpperCase {
		t.Fatalf("expected constant casing SNAKE_CASE_FULL_CAPS, got %q", cfg.Casing.Constants)
	}
	if len(cfg.Casing.IgnoreNames) != 1 || cfg.Casing.IgnoreNames[0] != "generated_name" {
		t.Fatalf("expected casing ignore names to load, got %#v", cfg.Casing.IgnoreNames)
	}
	if len(cfg.Casing.IgnorePatterns) != 1 || cfg.Casing.IgnorePatterns[0] != "^Test[A-Z]" {
		t.Fatalf("expected casing ignore patterns to load, got %#v", cfg.Casing.IgnorePatterns)
	}
	if !cfg.GithubActionsPinned.Enabled {
		t.Fatalf("expected github-actions-pinned rule to be enabled")
	}
}

func TestLoadFileAppliesLanguageOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  max-function-parameters:
    enabled: true
    max: 3
  format:
    enabled: true
  casing:
    enabled: false
    functions: language-default
languages:
  go:
    files:
      - cmd/herm/*.go
    exclude:
      - "**/*_test.go"
    rules:
      max-function-parameters:
        max: 2
      format:
        enabled: false
  swift:
    rules:
      max-function-parameters:
        max: 5
      casing:
        enabled: true
        functions: camelCase
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := LoadFile(LoadFileRequest{
		Path:     path,
		Base:     Default(),
		Language: "go",
	})
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.MaxFunctionParameters.Max != 2 {
		t.Fatalf("expected go max function parameters override to be 2, got %d", cfg.MaxFunctionParameters.Max)
	}
	if len(cfg.FileSelection.Files) != 1 || cfg.FileSelection.Files[0] != "cmd/herm/*.go" {
		t.Fatalf("expected go file selection to load, got %#v", cfg.FileSelection.Files)
	}
	if len(cfg.FileSelection.Exclude) != 1 || cfg.FileSelection.Exclude[0] != "**/*_test.go" {
		t.Fatalf("expected go file excludes to load, got %#v", cfg.FileSelection.Exclude)
	}
	if cfg.Format.Enabled {
		t.Fatalf("expected go format override to be disabled")
	}
	if cfg.Casing.Enabled {
		t.Fatalf("expected swift casing override to be ignored for go")
	}
	if cfg.Casing.Functions != CasingLanguageDefault {
		t.Fatalf("expected global casing function style to remain language-default, got %q", cfg.Casing.Functions)
	}
}

func TestLoadFileIgnoresLanguageOverridesWithoutLanguage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  max-function-parameters:
    max: 3
languages:
  go:
    rules:
      max-function-parameters:
        max: 2
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := LoadFile(LoadFileRequest{
		Path: path,
		Base: Default(),
	})
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.MaxFunctionParameters.Max != 3 {
		t.Fatalf("expected language overrides to be ignored without a requested language, got %d", cfg.MaxFunctionParameters.Max)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  source-file-header:
    minimum: 10
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := LoadFile(LoadFileRequest{Path: path, Base: Default()}); err == nil {
		t.Fatalf("expected LoadFile to reject unknown field")
	}
}

func TestLoadFileRejectsUnknownGithubActionsPinnedFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  github-actions-pinned:
    pin: true
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := LoadFile(LoadFileRequest{Path: path, Base: Default()}); err == nil {
		t.Fatalf("expected LoadFile to reject unknown github-actions-pinned field")
	}
}

func TestValidateRejectsInvalidHeaderBounds(t *testing.T) {
	cfg := Default()
	cfg.SourceFileHeader.MinLength = 20
	cfg.SourceFileHeader.MaxLength = 10

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid header bounds")
	}
}

func TestValidateRejectsInvalidLineBounds(t *testing.T) {
	cfg := Default()
	cfg.SourceFileLines.Max = -1

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid source file line bound")
	}

	cfg = Default()
	cfg.FunctionBodyLines.Max = -1
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid function body line bound")
	}
}

func TestValidateRejectsInvalidFunctionDocstringPolicy(t *testing.T) {
	cfg := Default()
	cfg.FunctionDocstring.Policy = "sometimes"

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid docstring policy")
	}
}

func TestValidateAcceptsFormatConfig(t *testing.T) {
	cfg := Default()
	cfg.Format.Enabled = false
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected Validate to accept format config, got %v", err)
	}
}

func TestValidateRejectsInvalidCasingConfig(t *testing.T) {
	cfg := Default()
	cfg.Casing.Functions = "mixed"

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid casing style")
	}

	cfg = Default()
	cfg.Casing.IgnorePatterns = []string{"["}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected Validate to reject invalid casing ignore pattern")
	}
}
