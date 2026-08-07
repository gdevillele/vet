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
		t.Fatalf("expected format to be disabled")
	}
	if !cfg.Casing.Enabled {
		t.Fatalf("expected casing to be enabled")
	}
	if !cfg.GithubActionsPinned.Enabled {
		t.Fatalf("expected github-actions-pinned to be enabled")
	}
}

func TestLoadFileAppliesLanguageSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vet.yaml")
	data := []byte(`version: 1
rules:
  format:
    enabled: true
languages:
  cpp:
    files:
      - src/...
    exclude:
      - "**/generated/**"
    rules:
      format:
        enabled: false
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := LoadFile(LoadFileRequest{
		Path:     path,
		Base:     Default(),
		Language: "cpp",
	})
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if len(cfg.FileSelection.Files) != 1 || cfg.FileSelection.Files[0] != "src/..." {
		t.Fatalf("unexpected files: %#v", cfg.FileSelection.Files)
	}
	if len(cfg.FileSelection.Exclude) != 1 || cfg.FileSelection.Exclude[0] != "**/generated/**" {
		t.Fatalf("unexpected exclude: %#v", cfg.FileSelection.Exclude)
	}
	if cfg.Format.Enabled {
		t.Fatalf("expected language override to disable format")
	}
}

func TestDefaultEnablesFormat(t *testing.T) {
	cfg := Default()
	if !cfg.Format.Enabled {
		t.Fatalf("expected format enabled by default")
	}
}

func TestDefaultDisablesUnimplementedMaxFunctionParameters(t *testing.T) {
	cfg := Default()
	if cfg.MaxFunctionParameters.Enabled {
		t.Fatalf("expected max-function-parameters disabled by default for C/C++")
	}
	if active := ActiveUnsupportedRules(cfg); len(active) != 0 {
		t.Fatalf("expected no active unsupported rules in Default(), got %#v", active)
	}
}

func TestActiveUnsupportedRulesReportsNonDefaultSettings(t *testing.T) {
	cfg := Default()
	cfg.MaxFunctionParameters.Enabled = true
	cfg.FunctionBodyLines.Max = 10
	cfg.FunctionDocstring.Policy = FunctionDocstringMandatory
	cfg.Casing.Enabled = true

	active := ActiveUnsupportedRules(cfg)
	if len(active) != 4 {
		t.Fatalf("expected 4 active unsupported rules, got %#v", active)
	}
}

func TestValidateAcceptsFormatConfig(t *testing.T) {
	cfg := Default()
	cfg.Format.Enabled = false
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsInvertedHeaderBounds(t *testing.T) {
	cfg := Default()
	cfg.SourceFileHeader.MinLength = 20
	cfg.SourceFileHeader.MaxLength = 10
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected validation error")
	}
}
