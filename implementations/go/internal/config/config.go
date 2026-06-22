package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultMaxFunctionParameters = 1

type Config struct {
	MaxFunctionParameters MaxFunctionParametersRule
	SourceFileHeader      SourceFileHeaderRule
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

	return nil
}
