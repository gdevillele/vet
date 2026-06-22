package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

const DefaultMaxFunctionParameters = 1

type Config struct {
	MaxFunctionParameters MaxFunctionParametersRule
	SourceFileHeader      SourceFileHeaderRule
	SourceFileLines       SourceFileLinesRule
	FunctionBodyLines     FunctionBodyLinesRule
	FunctionDocstring     FunctionDocstringRule
	Indent                IndentRule
	Casing                CasingRule
}

type MaxFunctionParametersRule struct {
	Enabled bool
	Max     int
}

type SourceFileHeaderRule struct {
	Required  bool
	MinLength int
	MaxLength int
}

type SourceFileLinesRule struct {
	Max int
}

type FunctionBodyLinesRule struct {
	Max int
}

type FunctionDocstringPolicy string

const (
	FunctionDocstringForbidden FunctionDocstringPolicy = "forbidden"
	FunctionDocstringOptional  FunctionDocstringPolicy = "optional"
	FunctionDocstringMandatory FunctionDocstringPolicy = "mandatory"
)

type FunctionDocstringRule struct {
	Policy FunctionDocstringPolicy
}

type IndentType string

const (
	IndentTabs            IndentType = "tabs"
	IndentSpaces          IndentType = "spaces"
	IndentLanguageDefault IndentType = "language-default"
)

type IndentRule struct {
	Type  IndentType
	Width int
}

type CasingStyle string

const (
	CasingOff             CasingStyle = "off"
	CasingLanguageDefault CasingStyle = "language-default"
	CasingCamelCase       CasingStyle = "camelCase"
	CasingUpperCamelCase  CasingStyle = "UpperCamelCase"
	CasingSnakeCase       CasingStyle = "snake_case"
	CasingSnakeUpperCase  CasingStyle = "SNAKE_CASE_FULL_CAPS"
)

type CasingRule struct {
	Enabled        bool
	Functions      CasingStyle
	Variables      CasingStyle
	Types          CasingStyle
	Constants      CasingStyle
	IgnoreNames    []string
	IgnorePatterns []string
}

type LoadFileRequest struct {
	Path string
	Base Config
}

type fileConfig struct {
	Version *int      `yaml:"version"`
	Rules   rulesFile `yaml:"rules"`
}

type rulesFile struct {
	MaxFunctionParameters *maxFunctionParametersFile `yaml:"max-function-parameters"`
	SourceFileHeader      *sourceFileHeaderFile      `yaml:"source-file-header"`
	SourceFileLines       *sourceFileLinesFile       `yaml:"max-source-file-lines"`
	FunctionBodyLines     *functionBodyLinesFile     `yaml:"max-function-body-lines"`
	FunctionDocstring     *functionDocstringFile     `yaml:"function-docstring"`
	Indent                *indentFile                `yaml:"indent"`
	Casing                *casingFile                `yaml:"casing"`
}

type maxFunctionParametersFile struct {
	Enabled *bool `yaml:"enabled"`
	Max     *int  `yaml:"max"`
}

type sourceFileHeaderFile struct {
	Required  *bool `yaml:"required"`
	MinLength *int  `yaml:"min-length"`
	MaxLength *int  `yaml:"max-length"`
}

type sourceFileLinesFile struct {
	Max *int `yaml:"max"`
}

type functionBodyLinesFile struct {
	Max *int `yaml:"max"`
}

type functionDocstringFile struct {
	Policy *FunctionDocstringPolicy `yaml:"policy"`
}

type indentFile struct {
	Type  *IndentType `yaml:"type"`
	Width *int        `yaml:"width"`
}

type casingFile struct {
	Enabled        *bool        `yaml:"enabled"`
	Functions      *CasingStyle `yaml:"functions"`
	Variables      *CasingStyle `yaml:"variables"`
	Types          *CasingStyle `yaml:"types"`
	Constants      *CasingStyle `yaml:"constants"`
	IgnoreNames    []string     `yaml:"ignore-names"`
	IgnorePatterns []string     `yaml:"ignore-patterns"`
}

func Default() Config {
	return Config{
		MaxFunctionParameters: MaxFunctionParametersRule{
			Enabled: true,
			Max:     DefaultMaxFunctionParameters,
		},
		SourceFileHeader: SourceFileHeaderRule{
			Required:  false,
			MinLength: 0,
			MaxLength: 0,
		},
		SourceFileLines: SourceFileLinesRule{
			Max: 0,
		},
		FunctionBodyLines: FunctionBodyLinesRule{
			Max: 0,
		},
		FunctionDocstring: FunctionDocstringRule{
			Policy: FunctionDocstringOptional,
		},
		Indent: IndentRule{
			Type:  IndentLanguageDefault,
			Width: 0,
		},
		Casing: CasingRule{
			Enabled:   false,
			Functions: CasingLanguageDefault,
			Variables: CasingLanguageDefault,
			Types:     CasingLanguageDefault,
			Constants: CasingLanguageDefault,
		},
	}
}

func LoadFile(request LoadFileRequest) (Config, error) {
	data, err := os.ReadFile(request.Path)
	if err != nil {
		return request.Base, fmt.Errorf("read config %q: %w", request.Path, err)
	}

	document := fileConfig{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		return request.Base, fmt.Errorf("parse config %q: %w", request.Path, err)
	}

	if document.Version != nil && *document.Version != 1 {
		return request.Base, fmt.Errorf("config %q uses unsupported version %d", request.Path, *document.Version)
	}

	result := request.Base
	if document.Rules.MaxFunctionParameters != nil {
		rule := document.Rules.MaxFunctionParameters
		if rule.Enabled != nil {
			result.MaxFunctionParameters.Enabled = *rule.Enabled
		}
		if rule.Max != nil {
			result.MaxFunctionParameters.Max = *rule.Max
		}
	}

	if document.Rules.SourceFileHeader != nil {
		rule := document.Rules.SourceFileHeader
		if rule.Required != nil {
			result.SourceFileHeader.Required = *rule.Required
		}
		if rule.MinLength != nil {
			result.SourceFileHeader.MinLength = *rule.MinLength
		}
		if rule.MaxLength != nil {
			result.SourceFileHeader.MaxLength = *rule.MaxLength
		}
	}

	if document.Rules.SourceFileLines != nil {
		rule := document.Rules.SourceFileLines
		if rule.Max != nil {
			result.SourceFileLines.Max = *rule.Max
		}
	}

	if document.Rules.FunctionBodyLines != nil {
		rule := document.Rules.FunctionBodyLines
		if rule.Max != nil {
			result.FunctionBodyLines.Max = *rule.Max
		}
	}

	if document.Rules.FunctionDocstring != nil {
		rule := document.Rules.FunctionDocstring
		if rule.Policy != nil {
			result.FunctionDocstring.Policy = *rule.Policy
		}
	}

	if document.Rules.Indent != nil {
		rule := document.Rules.Indent
		if rule.Type != nil {
			result.Indent.Type = *rule.Type
		}
		if rule.Width != nil {
			result.Indent.Width = *rule.Width
		}
	}

	if document.Rules.Casing != nil {
		rule := document.Rules.Casing
		if rule.Enabled != nil {
			result.Casing.Enabled = *rule.Enabled
		}
		if rule.Functions != nil {
			result.Casing.Functions = *rule.Functions
		}
		if rule.Variables != nil {
			result.Casing.Variables = *rule.Variables
		}
		if rule.Types != nil {
			result.Casing.Types = *rule.Types
		}
		if rule.Constants != nil {
			result.Casing.Constants = *rule.Constants
		}
		if rule.IgnoreNames != nil {
			result.Casing.IgnoreNames = rule.IgnoreNames
		}
		if rule.IgnorePatterns != nil {
			result.Casing.IgnorePatterns = rule.IgnorePatterns
		}
	}

	if err := Validate(result); err != nil {
		return result, err
	}

	return result, nil
}

func Validate(cfg Config) error {
	if cfg.MaxFunctionParameters.Max < 0 {
		return fmt.Errorf("max-function-parameters.max must be zero or greater")
	}
	if cfg.SourceFileHeader.MinLength < 0 {
		return fmt.Errorf("source-file-header.min-length must be zero or greater")
	}
	if cfg.SourceFileHeader.MaxLength < 0 {
		return fmt.Errorf("source-file-header.max-length must be zero or greater")
	}
	if cfg.SourceFileHeader.MinLength > 0 && cfg.SourceFileHeader.MaxLength > 0 && cfg.SourceFileHeader.MaxLength < cfg.SourceFileHeader.MinLength {
		return fmt.Errorf("source-file-header.max-length must be greater than or equal to source-file-header.min-length")
	}
	if cfg.SourceFileLines.Max < 0 {
		return fmt.Errorf("max-source-file-lines.max must be zero or greater")
	}
	if cfg.FunctionBodyLines.Max < 0 {
		return fmt.Errorf("max-function-body-lines.max must be zero or greater")
	}
	switch cfg.FunctionDocstring.Policy {
	case FunctionDocstringForbidden, FunctionDocstringOptional, FunctionDocstringMandatory:
	default:
		return fmt.Errorf("function-docstring.policy must be forbidden, optional, or mandatory")
	}
	switch cfg.Indent.Type {
	case IndentTabs, IndentSpaces, IndentLanguageDefault:
	default:
		return fmt.Errorf("indent.type must be tabs, spaces, or language-default")
	}
	if cfg.Indent.Width < 0 {
		return fmt.Errorf("indent.width must be zero or greater")
	}
	if err := validateCasingStyle("casing.functions", cfg.Casing.Functions); err != nil {
		return err
	}
	if err := validateCasingStyle("casing.variables", cfg.Casing.Variables); err != nil {
		return err
	}
	if err := validateCasingStyle("casing.types", cfg.Casing.Types); err != nil {
		return err
	}
	if err := validateCasingStyle("casing.constants", cfg.Casing.Constants); err != nil {
		return err
	}
	for _, pattern := range cfg.Casing.IgnorePatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("casing.ignore-patterns contains invalid regex %q: %w", pattern, err)
		}
	}

	return nil
}

func validateCasingStyle(field string, style CasingStyle) error {
	switch style {
	case CasingOff, CasingLanguageDefault, CasingCamelCase, CasingUpperCamelCase, CasingSnakeCase, CasingSnakeUpperCase:
		return nil
	default:
		return fmt.Errorf("%s must be off, language-default, camelCase, UpperCamelCase, snake_case, or SNAKE_CASE_FULL_CAPS", field)
	}
}
