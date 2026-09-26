package cppanalysis

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdevillele/vet/implementations/cpp/internal/diagnostic"
)

const RuleNoComments = "VET015"

func (a Analyzer) checkComments(request AnalyzeFileRequest) ([]diagnostic.Diagnostic, error) {
	rule := a.config.NoComments
	if !rule.Enabled {
		return nil, nil
	}
	binary, err := a.runner.LookPath("clang")
	if err != nil {
		return nil, fmt.Errorf("clang not found in PATH (required for no-comments / VET015); install clang or disable with -no-comments=false: %w", err)
	}
	file, err := os.CreateTemp("", "vet-comments-*.cc")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(request.Source)
	closeErr := file.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	language := "c++"
	if strings.HasSuffix(request.Path, ".c") {
		language = "c"
	}
	output, code, err := a.runner.Run(binary, []string{"-x", language, "-fsyntax-only", "-Xclang", "-dump-raw-tokens", file.Name()})
	if err != nil {
		return nil, fmt.Errorf("run clang on %s: %w", request.Path, err)
	}
	if code != 0 {
		return nil, fmt.Errorf("clang comment scan failed for %s (exit %d): %s", request.Path, code, output)
	}
	return commentDiagnostics(request, rule.AllowHeader, string(output), file.Name())
}

func commentDiagnostics(request AnalyzeFileRequest, allowHeader bool, output string, tokenPath string) ([]diagnostic.Diagnostic, error) {
	source := string(request.Source)
	header := findSourceFileHeader(source)
	startLine, startColumn := offsetToLineColumn(source, header.Offset)
	endLine, endColumn := offsetToLineColumn(source, header.FirstCodeOffset)
	// Token spellings can contain newlines; only the temporary file location ends a record.
	location := regexp.MustCompile(`\tLoc=<` + regexp.QuoteMeta(tokenPath) + `:(\d+):(\d+)>\r?\n`)
	var diagnostics []diagnostic.Diagnostic
	cursor := 0
	for _, match := range location.FindAllStringSubmatchIndex(output, -1) {
		token := output[cursor:match[0]]
		cursor = match[1]
		if !strings.HasPrefix(token, "comment '") {
			continue
		}
		line, _ := strconv.Atoi(output[match[2]:match[3]])
		column, _ := strconv.Atoi(output[match[4]:match[5]])
		afterStart := line > startLine || (line == startLine && column >= startColumn)
		beforeEnd := line < endLine || (line == endLine && column < endColumn)
		if allowHeader && header.Present && afterStart && beforeEnd {
			continue
		}
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			RuleID: RuleNoComments, Severity: diagnostic.SeverityError,
			Message: "comment is not allowed", File: request.Path, Line: line, Column: column,
		})
	}
	if strings.TrimSpace(output[cursor:]) != "" || (cursor == 0 && strings.Trim(source, " \t\r\n\ufeff") != "") {
		return nil, fmt.Errorf("unrecognized clang raw token output for %s", request.Path)
	}
	return diagnostics, nil
}
