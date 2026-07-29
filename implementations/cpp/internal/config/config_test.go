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
  indent:
    type: spaces
    width: 4
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
	if cfg.Indent.Type != IndentSpaces {
		t.Fatalf("expected spaces indent type, got %q", cfg.Indent.Type)
	}
	if cfg.Indent.Width != 4 {
		t.Fatalf("expected indent width 4, got %d", cfg.Indent.Width)
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
  indent:
    type: tabs
    width: 0
languages:
  cpp:
    files:
      - src/...
    exclude:
      - "**/generated/**"
    rules:
      indent:
        type: spaces
        width: 2
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
	if cfg.Indent.Type != IndentSpaces {
		t.Fatalf("expected language override spaces, got %q", cfg.Indent.Type)
	}
	if cfg.Indent.Width != 2 {
		t.Fatalf("expected language override width 2, got %d", cfg.Indent.Width)
	}
}

func TestValidateRejectsInvalidIndentType(t *testing.T) {
	cfg := Default()
	cfg.Indent.Type = "mixed"
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected validation error")
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
