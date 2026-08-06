package speccontract

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type rulesDocument struct {
	Version          int                 `yaml:"version"`
	Languages        []string            `yaml:"languages"`
	DiagnosticFields []string            `yaml:"diagnostic_fields"`
	Rules            map[string]ruleSpec `yaml:"rules"`
}

type ruleSpec struct {
	Name                  string                           `yaml:"name"`
	DefaultSeverity       string                           `yaml:"default_severity"`
	DefaultConfig         map[string]any                   `yaml:"default_config"`
	LanguageCompatibility map[string]languageCompatibility `yaml:"language_compatibility"`
	Summary               string                           `yaml:"summary"`
	Notes                 []string                         `yaml:"notes"`
}

type languageCompatibility struct {
	Status         string   `yaml:"status"`
	Implementation string   `yaml:"implementation"`
	Reason         string   `yaml:"reason"`
	Notes          []string `yaml:"notes"`
}

func TestRulesDeclareLanguageCompatibility(t *testing.T) {
	document := loadRulesDocument(t)
	if document.Version != 1 {
		t.Fatalf("expected rules spec version 1, got %d", document.Version)
	}
	if len(document.Languages) == 0 {
		t.Fatalf("rules spec must declare supported languages")
	}
	if len(document.Rules) == 0 {
		t.Fatalf("rules spec must declare at least one rule")
	}
	if !contains(document.Languages, "cpp") {
		t.Fatalf("rules spec must declare cpp language")
	}

	for ruleID, rule := range document.Rules {
		t.Run(ruleID, func(t *testing.T) {
			if strings.TrimSpace(rule.Name) == "" {
				t.Fatalf("rule name is required")
			}
			if len(rule.LanguageCompatibility) != len(document.Languages) {
				t.Fatalf("expected compatibility for %d languages, got %d", len(document.Languages), len(rule.LanguageCompatibility))
			}

			for _, language := range document.Languages {
				entry, ok := rule.LanguageCompatibility[language]
				if !ok {
					t.Fatalf("missing compatibility entry for %q", language)
				}

				switch entry.Status {
				case "compatible":
				case "incompatible":
					if strings.TrimSpace(entry.Reason) == "" {
						t.Fatalf("incompatible language %q must include a reason", language)
					}
				default:
					t.Fatalf("language %q has invalid compatibility status %q", language, entry.Status)
				}

				switch entry.Implementation {
				case "implemented", "planned", "unimplemented", "not-applicable":
				default:
					t.Fatalf("language %q has invalid implementation status %q", language, entry.Implementation)
				}

				if entry.Status == "compatible" && entry.Implementation == "not-applicable" {
					t.Fatalf("compatible language %q cannot have not-applicable implementation status", language)
				}
				if entry.Status == "incompatible" && entry.Implementation != "not-applicable" {
					t.Fatalf("incompatible language %q must have not-applicable implementation status", language)
				}
			}

			for language := range rule.LanguageCompatibility {
				if !contains(document.Languages, language) {
					t.Fatalf("compatibility entry references unknown language %q", language)
				}
			}
		})
	}
}

func TestCppImplementedRulesMatchRunner(t *testing.T) {
	document := loadRulesDocument(t)
	implemented := map[string]bool{
		"VET002": true,
		"VET003": true,
		"VET004": true,
		"VET005": true,
		"VET008": true,
		"VET014": true,
	}
	// Language-aware rules are intentionally not scheduled for the Go-hosted
	// line-based C/C++ runner (no roadmap "planned" promise).
	unimplemented := map[string]bool{
		"VET001": true,
		"VET006": true,
		"VET007": true,
		"VET010": true,
		"VET011": true,
		"VET012": true,
		"VET013": true,
	}

	for ruleID, rule := range document.Rules {
		entry, ok := rule.LanguageCompatibility["cpp"]
		if !ok {
			t.Fatalf("rule %s missing cpp compatibility", ruleID)
		}
		switch {
		case implemented[ruleID]:
			if entry.Implementation != "implemented" {
				t.Fatalf("rule %s expected implemented for cpp, got %q", ruleID, entry.Implementation)
			}
		case unimplemented[ruleID]:
			if entry.Implementation != "unimplemented" {
				t.Fatalf("rule %s expected unimplemented for cpp, got %q", ruleID, entry.Implementation)
			}
		default:
			t.Fatalf("rule %s has no expected cpp implementation status", ruleID)
		}
	}
}

func loadRulesDocument(t *testing.T) rulesDocument {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file")
	}

	path := filepath.Join(filepath.Dir(filename), "../../../../spec/rules/v1.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}

	document := rulesDocument{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("Decode(%q) returned error: %v", path, err)
	}

	return document
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
