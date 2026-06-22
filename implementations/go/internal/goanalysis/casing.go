package goanalysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"

	"github.com/gdevillele/vet/implementations/go/internal/config"
	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
)

var (
	camelCasePattern      = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	upperCamelCasePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	snakeCasePattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	snakeUpperCasePattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$`)
)

type casingKind struct {
	RuleID string
	Name   string
	Style  config.CasingStyle
}

func (a Analyzer) checkCasing(request casingCheck) []diagnostic.Diagnostic {
	if !a.config.Casing.Enabled {
		return nil
	}

	var diagnostics []diagnostic.Diagnostic
	ast.Inspect(request.File, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.FuncDecl:
			diagnostics = append(diagnostics, a.checkFunctionNameCasing(request, item.Name)...)
			diagnostics = append(diagnostics, a.checkFieldListVariableCasing(request, item.Recv)...)
			diagnostics = append(diagnostics, a.checkFieldListVariableCasing(request, item.Type.Params)...)
			diagnostics = append(diagnostics, a.checkFieldListVariableCasing(request, item.Type.Results)...)
		case *ast.FuncLit:
			diagnostics = append(diagnostics, a.checkFieldListVariableCasing(request, item.Type.Params)...)
			diagnostics = append(diagnostics, a.checkFieldListVariableCasing(request, item.Type.Results)...)
		case *ast.GenDecl:
			diagnostics = append(diagnostics, a.checkGenDeclCasing(request, item)...)
			return false
		case *ast.AssignStmt:
			if item.Tok == token.DEFINE {
				diagnostics = append(diagnostics, a.checkAssignedVariableCasing(request, item.Lhs)...)
			}
		case *ast.RangeStmt:
			if item.Tok == token.DEFINE {
				diagnostics = append(diagnostics, a.checkRangeVariableCasing(request, item.Key)...)
				diagnostics = append(diagnostics, a.checkRangeVariableCasing(request, item.Value)...)
			}
		}

		return true
	})

	return diagnostics
}

func (a Analyzer) checkGenDeclCasing(request casingCheck, declaration *ast.GenDecl) []diagnostic.Diagnostic {
	var diagnostics []diagnostic.Diagnostic
	for _, spec := range declaration.Specs {
		switch item := spec.(type) {
		case *ast.ValueSpec:
			kind := a.variableCasingKind()
			if declaration.Tok == token.CONST {
				kind = a.constantCasingKind()
			}
			for _, name := range item.Names {
				diagnostics = append(diagnostics, a.checkIdentifierCasing(request, name, kind)...)
			}
		case *ast.TypeSpec:
			diagnostics = append(diagnostics, a.checkIdentifierCasing(request, item.Name, a.typeCasingKind())...)
			diagnostics = append(diagnostics, a.checkInterfaceMethodCasing(request, item.Type)...)
		}
	}
	return diagnostics
}

func (a Analyzer) checkFunctionNameCasing(request casingCheck, name *ast.Ident) []diagnostic.Diagnostic {
	return a.checkIdentifierCasing(request, name, a.functionCasingKind())
}

func (a Analyzer) checkFieldListVariableCasing(request casingCheck, fields *ast.FieldList) []diagnostic.Diagnostic {
	if fields == nil {
		return nil
	}

	var diagnostics []diagnostic.Diagnostic
	for _, field := range fields.List {
		for _, name := range field.Names {
			diagnostics = append(diagnostics, a.checkIdentifierCasing(request, name, a.variableCasingKind())...)
		}
	}
	return diagnostics
}

func (a Analyzer) checkAssignedVariableCasing(request casingCheck, expressions []ast.Expr) []diagnostic.Diagnostic {
	var diagnostics []diagnostic.Diagnostic
	for _, expression := range expressions {
		if name, ok := expression.(*ast.Ident); ok {
			diagnostics = append(diagnostics, a.checkIdentifierCasing(request, name, a.variableCasingKind())...)
		}
	}
	return diagnostics
}

func (a Analyzer) checkRangeVariableCasing(request casingCheck, expression ast.Expr) []diagnostic.Diagnostic {
	if name, ok := expression.(*ast.Ident); ok {
		return a.checkIdentifierCasing(request, name, a.variableCasingKind())
	}
	return nil
}

func (a Analyzer) checkInterfaceMethodCasing(request casingCheck, expression ast.Expr) []diagnostic.Diagnostic {
	interfaceType, ok := expression.(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil {
		return nil
	}

	var diagnostics []diagnostic.Diagnostic
	for _, method := range interfaceType.Methods.List {
		if _, ok := method.Type.(*ast.FuncType); !ok {
			continue
		}
		for _, name := range method.Names {
			diagnostics = append(diagnostics, a.checkIdentifierCasing(request, name, a.functionCasingKind())...)
		}
	}
	return diagnostics
}

func (a Analyzer) checkIdentifierCasing(request casingCheck, identifier *ast.Ident, kind casingKind) []diagnostic.Diagnostic {
	if identifier == nil || shouldIgnoreIdentifierCasing(a.config.Casing, identifier.Name) {
		return nil
	}

	style := effectiveGoCasingStyle(kind.Style, identifier.Name)
	if style == config.CasingOff || casingStyleMatches(style, identifier.Name) {
		return nil
	}

	position := request.FileSet.Position(identifier.Pos())
	return []diagnostic.Diagnostic{{
		RuleID:   kind.RuleID,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("%s %q must use %s", kind.Name, identifier.Name, style),
		File:     request.Path,
		Line:     position.Line,
		Column:   position.Column,
	}}
}

func (a Analyzer) functionCasingKind() casingKind {
	return casingKind{
		RuleID: RuleFunctionCasing,
		Name:   "function",
		Style:  a.config.Casing.Functions,
	}
}

func (a Analyzer) variableCasingKind() casingKind {
	return casingKind{
		RuleID: RuleVariableCasing,
		Name:   "variable",
		Style:  a.config.Casing.Variables,
	}
}

func (a Analyzer) typeCasingKind() casingKind {
	return casingKind{
		RuleID: RuleTypeCasing,
		Name:   "type",
		Style:  a.config.Casing.Types,
	}
}

func (a Analyzer) constantCasingKind() casingKind {
	return casingKind{
		RuleID: RuleConstantCasing,
		Name:   "constant",
		Style:  a.config.Casing.Constants,
	}
}

func effectiveGoCasingStyle(style config.CasingStyle, name string) config.CasingStyle {
	if style != config.CasingLanguageDefault {
		return style
	}

	if ast.IsExported(name) {
		return config.CasingUpperCamelCase
	}
	return config.CasingCamelCase
}

func casingStyleMatches(style config.CasingStyle, name string) bool {
	switch style {
	case config.CasingCamelCase:
		return camelCasePattern.MatchString(name)
	case config.CasingUpperCamelCase:
		return upperCamelCasePattern.MatchString(name)
	case config.CasingSnakeCase:
		return snakeCasePattern.MatchString(name)
	case config.CasingSnakeUpperCase:
		return snakeUpperCasePattern.MatchString(name)
	case config.CasingOff:
		return true
	default:
		return false
	}
}

func shouldIgnoreIdentifierCasing(rule config.CasingRule, name string) bool {
	if name == "_" {
		return true
	}
	for _, ignored := range rule.IgnoreNames {
		if name == ignored {
			return true
		}
	}
	for _, pattern := range rule.IgnorePatterns {
		if matched, err := regexp.MatchString(pattern, name); err == nil && matched {
			return true
		}
	}
	return false
}
