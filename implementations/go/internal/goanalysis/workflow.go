package goanalysis

import (
	"fmt"
	"strings"

	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
	"gopkg.in/yaml.v3"
)

const RuleGithubActionsPinned = "VET014"

type AnalyzeWorkflowFileRequest struct {
	Path   string
	Source []byte
}

func (a Analyzer) AnalyzeWorkflowFile(request AnalyzeWorkflowFileRequest) ([]diagnostic.Diagnostic, error) {
	if !a.config.GithubActionsPinned.Enabled {
		return nil, nil
	}

	var document yaml.Node
	if err := yaml.Unmarshal(request.Source, &document); err != nil {
		return nil, err
	}

	root := yamlDocumentRoot(&document)
	jobs := yamlMappingValue(root, "jobs")
	if jobs == nil || jobs.Kind != yaml.MappingNode {
		return nil, nil
	}

	var diagnostics []diagnostic.Diagnostic
	for index := 1; index < len(jobs.Content); index += 2 {
		job := jobs.Content[index]
		steps := yamlMappingValue(job, "steps")
		if steps == nil || steps.Kind != yaml.SequenceNode {
			continue
		}

		for _, step := range steps.Content {
			uses := yamlMappingValue(step, "uses")
			if uses == nil || uses.Kind != yaml.ScalarNode {
				continue
			}
			action := uses.Value
			if githubActionPinned(action) {
				continue
			}

			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				RuleID:   RuleGithubActionsPinned,
				Severity: diagnostic.SeverityError,
				Message:  fmt.Sprintf("GitHub action %q must be pinned to a full-length commit SHA", action),
				File:     request.Path,
				Line:     uses.Line,
				Column:   uses.Column,
			})
		}
	}

	return diagnostics, nil
}

func yamlDocumentRoot(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	node = yamlDocumentRoot(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	for index := 0; index+1 < len(node.Content); index += 2 {
		keyNode := node.Content[index]
		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == key {
			return yamlDocumentRoot(node.Content[index+1])
		}
	}
	return nil
}

func githubActionPinned(action string) bool {
	if strings.HasPrefix(action, "./") || strings.HasPrefix(action, "docker://") {
		return true
	}

	index := strings.LastIndex(action, "@")
	if index < 0 || index == len(action)-1 {
		return false
	}

	ref := action[index+1:]
	if len(ref) != 40 {
		return false
	}
	for _, char := range ref {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
