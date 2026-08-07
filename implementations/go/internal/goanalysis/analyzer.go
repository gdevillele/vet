package goanalysis

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
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
	RuleSourceFileLines          = "VET005"
	RuleFunctionBodyLines        = "VET006"
	RuleFunctionDocstring        = "VET007"
	RuleSourceFormat             = "VET008"
	RuleFunctionCasing           = "VET010"
	RuleVariableCasing           = "VET011"
	RuleTypeCasing               = "VET012"
	RuleConstantCasing           = "VET013"
)

type Analyzer struct {
	config config.Config
}

type AnalyzeFileRequest struct {
	Path   string
	Source []byte
}

type functionCheck struct {
	FileSet   *token.FileSet
	Path      string
	Name      string
	Pos       token.Pos
	Params    *ast.FieldList
	Body      *ast.BlockStmt
	Docstring bool
}

type fileHeaderCheck struct {
	FileSet *token.FileSet
	File    *ast.File
	Path    string
}

type sourceLineCheck struct {
	Path   string
	Source []byte
}

type formatCheck struct {
	Path   string
	Source []byte
}

type casingCheck struct {
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
	diagnostics = append(diagnostics, a.checkSourceFileLines(sourceLineCheck{
		Path:   request.Path,
		Source: request.Source,
	})...)
	diagnostics = append(diagnostics, a.checkFormat(formatCheck{
		Path:   request.Path,
		Source: request.Source,
	})...)
	diagnostics = append(diagnostics, a.checkFileHeader(fileHeaderCheck{
		FileSet: fileSet,
		File:    file,
		Path:    request.Path,
	})...)
	diagnostics = append(diagnostics, a.checkCasing(casingCheck{
		FileSet: fileSet,
		File:    file,
		Path:    request.Path,
	})...)

	ast.Inspect(file, func(node ast.Node) bool {
		switch function := node.(type) {
		case *ast.FuncDecl:
			diagnostics = append(diagnostics, a.checkFunction(functionCheck{
				FileSet:   fileSet,
				Path:      request.Path,
				Name:      function.Name.Name,
				Pos:       function.Name.Pos(),
				Params:    function.Type.Params,
				Body:      function.Body,
				Docstring: hasDocstring(function.Doc),
			})...)
		case *ast.FuncLit:
			diagnostics = append(diagnostics, a.checkFunction(functionCheck{
				FileSet: fileSet,
				Path:    request.Path,
				Name:    "function literal",
				Pos:     function.Type.Func,
				Params:  function.Type.Params,
				Body:    function.Body,
			})...)
		}

		return true
	})

	return diagnostics, nil
}

func (a Analyzer) checkFormat(request formatCheck) []diagnostic.Diagnostic {
	if !a.config.Format.Enabled {
		return nil
	}

	formatted, err := format.Source(request.Source)
	if err != nil {
		// Parse errors are reported by AnalyzeFile's primary parse step.
		return nil
	}
	if bytes.Equal(formatted, request.Source) {
		return nil
	}

	return []diagnostic.Diagnostic{{
		RuleID:   RuleSourceFormat,
		Severity: diagnostic.SeverityError,
		Message:  "file is not gofmt-formatted",
		File:     request.Path,
		Line:     1,
		Column:   1,
	}}
}

func (a Analyzer) checkSourceFileLines(request sourceLineCheck) []diagnostic.Diagnostic {
	rule := a.config.SourceFileLines
	if rule.Max <= 0 {
		return nil
	}

	count := sourceLineCount(request.Source)
	if count <= rule.Max {
		return nil
	}

	return []diagnostic.Diagnostic{{
		RuleID:   RuleSourceFileLines,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("source file has %d lines; maximum allowed is %d", count, rule.Max),
		File:     request.Path,
		Line:     1,
		Column:   1,
	}}
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
	diagnostics := a.checkFunctionParameters(request)
	diagnostics = append(diagnostics, a.checkFunctionBodyLines(request)...)
	diagnostics = append(diagnostics, a.checkFunctionDocstring(request)...)
	return diagnostics
}

func (a Analyzer) checkFunctionParameters(request functionCheck) []diagnostic.Diagnostic {
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

func (a Analyzer) checkFunctionBodyLines(request functionCheck) []diagnostic.Diagnostic {
	rule := a.config.FunctionBodyLines
	if rule.Max <= 0 || request.Body == nil {
		return nil
	}

	count := functionBodyLineCount(functionBodyLineCountRequest{
		FileSet: request.FileSet,
		Body:    request.Body,
	})
	if count <= rule.Max {
		return nil
	}

	position := request.FileSet.Position(request.Pos)
	return []diagnostic.Diagnostic{{
		RuleID:   RuleFunctionBodyLines,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("%s body has %d lines; maximum allowed is %d", request.Name, count, rule.Max),
		File:     request.Path,
		Line:     position.Line,
		Column:   position.Column,
	}}
}

func (a Analyzer) checkFunctionDocstring(request functionCheck) []diagnostic.Diagnostic {
	rule := a.config.FunctionDocstring
	if request.Name == "function literal" || rule.Policy == config.FunctionDocstringOptional {
		return nil
	}

	if rule.Policy == config.FunctionDocstringMandatory && request.Docstring {
		return nil
	}
	if rule.Policy == config.FunctionDocstringForbidden && !request.Docstring {
		return nil
	}

	position := request.FileSet.Position(request.Pos)
	message := fmt.Sprintf("%s must have a docstring", request.Name)
	if rule.Policy == config.FunctionDocstringForbidden {
		message = fmt.Sprintf("%s must not have a docstring", request.Name)
	}

	return []diagnostic.Diagnostic{{
		RuleID:   RuleFunctionDocstring,
		Severity: diagnostic.SeverityError,
		Message:  message,
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

type functionBodyLineCountRequest struct {
	FileSet *token.FileSet
	Body    *ast.BlockStmt
}

func functionBodyLineCount(request functionBodyLineCountRequest) int {
	start := request.FileSet.Position(request.Body.Lbrace).Line
	end := request.FileSet.Position(request.Body.Rbrace).Line
	count := end - start - 1
	if count < 0 {
		return 0
	}
	return count
}

func sourceLineCount(source []byte) int {
	if len(source) == 0 {
		return 0
	}

	count := 1
	for _, char := range source {
		if char == '\n' {
			count++
		}
	}
	if source[len(source)-1] == '\n' {
		count--
	}
	return count
}

func hasDocstring(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}

	return strings.TrimSpace(group.Text()) != ""
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
