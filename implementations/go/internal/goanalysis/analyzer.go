package goanalysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/gdevillele/vet/implementations/go/internal/config"
	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
)

const RuleMaxFunctionParameters = "VET001"

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
