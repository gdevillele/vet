package goanalysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/gdevillele/vet/implementations/go/internal/config"
	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
)

const (
	RuleMaxFunctionParameters    = "VET001"
	RuleSourceFileHeaderRequired = "VET002"
	RuleSourceFileHeaderMin      = "VET003"
	RuleSourceFileHeaderMax      = "VET004"
)

type Analyzer struct {
	config config.Config
}

type AnalyzeFileRequest struct {
	Path   string
	Source []byte
}

type functionCheck struct {
	FileSet *token.FileSet
	Path    string
	Name    string
	Pos     token.Pos
	Params  *ast.FieldList
}

type fileHeaderCheck struct {
	FileSet *token.FileSet
	File    *ast.File
	Path    string
}

type sourceFileHeader struct {
	Present bool
	Text    string
	Pos     token.Pos
}

func New(cfg config.Config) Analyzer {
	return Analyzer{config: cfg}
}

func (a Analyzer) AnalyzeFile(request AnalyzeFileRequest) ([]diagnostic.Diagnostic, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, request.Path, request.Source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var diagnostics []diagnostic.Diagnostic
	diagnostics = append(diagnostics, a.checkFileHeader(fileHeaderCheck{
		FileSet: fileSet,
		File:    file,
		Path:    request.Path,
	})...)

	ast.Inspect(file, func(node ast.Node) bool {
		switch function := node.(type) {
		case *ast.FuncDecl:
			diagnostics = append(diagnostics, a.checkFunction(functionCheck{
				FileSet: fileSet,
				Path:    request.Path,
				Name:    function.Name.Name,
				Pos:     function.Name.Pos(),
				Params:  function.Type.Params,
			})...)
		case *ast.FuncLit:
			diagnostics = append(diagnostics, a.checkFunction(functionCheck{
				FileSet: fileSet,
				Path:    request.Path,
				Name:    "function literal",
				Pos:     function.Type.Func,
				Params:  function.Type.Params,
			})...)
		}

		return true
	})

	return diagnostics, nil
}

func (a Analyzer) checkFileHeader(request fileHeaderCheck) []diagnostic.Diagnostic {
	rule := a.config.SourceFileHeader
	header := findSourceFileHeader(request.File)

	if !header.Present {
		if !rule.Required {
			return nil
		}

		position := request.FileSet.Position(request.File.Package)
		return []diagnostic.Diagnostic{{
			RuleID:   RuleSourceFileHeaderRequired,
			Severity: diagnostic.SeverityError,
			Message:  "source file has no header",
			File:     request.Path,
			Line:     position.Line,
			Column:   position.Column,
		}}
	}

	length := len([]rune(header.Text))
	position := request.FileSet.Position(header.Pos)
	diagnostics := make([]diagnostic.Diagnostic, 0, 2)

	if rule.MinLength > 0 && length < rule.MinLength {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			RuleID:   RuleSourceFileHeaderMin,
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("file header has %d characters; minimum allowed is %d", length, rule.MinLength),
			File:     request.Path,
			Line:     position.Line,
			Column:   position.Column,
		})
	}

	if rule.MaxLength > 0 && length > rule.MaxLength {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			RuleID:   RuleSourceFileHeaderMax,
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("file header has %d characters; maximum allowed is %d", length, rule.MaxLength),
			File:     request.Path,
			Line:     position.Line,
			Column:   position.Column,
		})
	}

	return diagnostics
}

func (a Analyzer) checkFunction(request functionCheck) []diagnostic.Diagnostic {
	rule := a.config.MaxFunctionParameters
	if !rule.Enabled {
		return nil
	}

	count := parameterCount(request.Params)
	if count <= rule.Max {
		return nil
	}

	position := request.FileSet.Position(request.Pos)
	return []diagnostic.Diagnostic{{
		RuleID:   RuleMaxFunctionParameters,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("%s has %d parameters; maximum allowed is %d", request.Name, count, rule.Max),
		File:     request.Path,
		Line:     position.Line,
		Column:   position.Column,
	}}
}

func parameterCount(params *ast.FieldList) int {
	if params == nil {
		return 0
	}

	count := 0
	for _, field := range params.List {
		if len(field.Names) == 0 {
			count++
			continue
		}

		count += len(field.Names)
	}

	return count
}

func findSourceFileHeader(file *ast.File) sourceFileHeader {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}

		text := extractHeaderText(group)
		if text == "" {
			continue
		}

		return sourceFileHeader{
			Present: true,
			Text:    text,
			Pos:     group.Pos(),
		}
	}

	return sourceFileHeader{}
}

func extractHeaderText(group *ast.CommentGroup) string {
	lines := make([]string, 0, len(group.List))

	for _, comment := range group.List {
		for _, line := range commentTextLines(comment.Text) {
			line = normalizeHeaderLine(line)
			if shouldIgnoreHeaderLine(line) {
				continue
			}

			lines = append(lines, line)
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func commentTextLines(text string) []string {
	if strings.HasPrefix(text, "//") {
		return []string{strings.TrimPrefix(text, "//")}
	}

	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	return strings.Split(text, "\n")
}

func normalizeHeaderLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "*")
	return strings.TrimSpace(line)
}

func shouldIgnoreHeaderLine(line string) bool {
	if line == "" {
		return true
	}

	if strings.HasPrefix(line, "go:build ") || strings.HasPrefix(line, "+build ") {
		return true
	}

	return strings.HasPrefix(line, "Code generated ") && strings.Contains(line, "DO NOT EDIT.")
}
