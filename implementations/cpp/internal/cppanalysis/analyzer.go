package cppanalysis

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gdevillele/vet/implementations/cpp/internal/config"
	"github.com/gdevillele/vet/implementations/cpp/internal/diagnostic"
)

const (
	RuleSourceFileHeaderRequired = "VET002"
	RuleSourceFileHeaderMin      = "VET003"
	RuleSourceFileHeaderMax      = "VET004"
	RuleSourceFileLines          = "VET005"
	RuleSourceFormat             = "VET008"

	clangFormatBinary = "clang-format"
)

type Analyzer struct {
	config config.Config
	runner CommandRunner
}

type AnalyzeFileRequest struct {
	Path   string
	Source []byte
}

type sourceFileHeader struct {
	Present         bool
	Text            string
	Offset          int
	FirstCodeOffset int
}

func New(cfg config.Config) Analyzer {
	return NewWithRunner(cfg, ExecRunner{})
}

func NewWithRunner(cfg config.Config, runner CommandRunner) Analyzer {
	if runner == nil {
		runner = ExecRunner{}
	}
	return Analyzer{config: cfg, runner: runner}
}

func (a Analyzer) AnalyzeFile(request AnalyzeFileRequest) ([]diagnostic.Diagnostic, error) {
	source := string(request.Source)
	var diagnostics []diagnostic.Diagnostic
	diagnostics = append(diagnostics, a.checkSourceFileLines(request.Path, source)...)

	formatDiagnostics, err := a.checkFormat(request.Path)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, formatDiagnostics...)
	diagnostics = append(diagnostics, a.checkFileHeader(request.Path, source)...)
	return diagnostics, nil
}

func (a Analyzer) checkSourceFileLines(path string, source string) []diagnostic.Diagnostic {
	rule := a.config.SourceFileLines
	if rule.Max <= 0 {
		return nil
	}

	count := sourceLineCount(source)
	if count <= rule.Max {
		return nil
	}

	return []diagnostic.Diagnostic{{
		RuleID:   RuleSourceFileLines,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("source file has %d lines; maximum allowed is %d", count, rule.Max),
		File:     path,
		Line:     1,
		Column:   1,
	}}
}

// checkFormat runs clang-format --dry-run --Werror when format checking is
// enabled. A missing clang-format binary is a hard error (never a silent skip).
func (a Analyzer) checkFormat(path string) ([]diagnostic.Diagnostic, error) {
	if !a.config.Format.Enabled {
		return nil, nil
	}

	binary, err := a.runner.LookPath(clangFormatBinary)
	if err != nil {
		return nil, fmt.Errorf(
			"clang-format not found in PATH (required for source-format / VET008); install clang-format or disable with -check-format=false: %w",
			err,
		)
	}

	_, exitCode, err := a.runner.Run(binary, []string{"--dry-run", "--Werror", path})
	if err != nil {
		return nil, fmt.Errorf("run clang-format on %s: %w", path, err)
	}
	if exitCode == 0 {
		return nil, nil
	}

	return []diagnostic.Diagnostic{{
		RuleID:   RuleSourceFormat,
		Severity: diagnostic.SeverityError,
		Message:  "file is not clang-format-formatted",
		File:     path,
		Line:     1,
		Column:   1,
	}}, nil
}

func (a Analyzer) checkFileHeader(path string, source string) []diagnostic.Diagnostic {
	rule := a.config.SourceFileHeader
	header := findSourceFileHeader(source)

	if !header.Present {
		if !rule.Required {
			return nil
		}
		line, column := offsetToLineColumn(source, header.FirstCodeOffset)
		return []diagnostic.Diagnostic{{
			RuleID:   RuleSourceFileHeaderRequired,
			Severity: diagnostic.SeverityError,
			Message:  "source file has no header",
			File:     path,
			Line:     line,
			Column:   column,
		}}
	}

	length := utf8.RuneCountInString(header.Text)
	line, column := offsetToLineColumn(source, header.Offset)
	diagnostics := make([]diagnostic.Diagnostic, 0, 2)

	if rule.MinLength > 0 && length < rule.MinLength {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			RuleID:   RuleSourceFileHeaderMin,
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("file header has %d characters; minimum allowed is %d", length, rule.MinLength),
			File:     path,
			Line:     line,
			Column:   column,
		})
	}

	if rule.MaxLength > 0 && length > rule.MaxLength {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			RuleID:   RuleSourceFileHeaderMax,
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("file header has %d characters; maximum allowed is %d", length, rule.MaxLength),
			File:     path,
			Line:     line,
			Column:   column,
		})
	}

	return diagnostics
}

func sourceLineCount(source string) int {
	if source == "" {
		return 0
	}

	count := 1
	for _, char := range source {
		if char == '\n' {
			count++
		}
	}
	if strings.HasSuffix(source, "\n") {
		count--
	}
	return count
}

func findSourceFileHeader(source string) sourceFileHeader {
	cursor := skipUTF8BOM(source, 0)

	for {
		cursor = skipWhitespace(source, cursor)
		if cursor >= len(source) {
			return sourceFileHeader{
				FirstCodeOffset: cursor,
			}
		}

		if strings.HasPrefix(source[cursor:], "//") {
			lines, endOffset := readLineCommentGroup(source, cursor)
			text := normalizedHeaderText(lines)
			if text != "" {
				return sourceFileHeader{
					Present:         true,
					Text:            text,
					Offset:          cursor,
					FirstCodeOffset: endOffset,
				}
			}
			cursor = endOffset
			continue
		}

		if strings.HasPrefix(source[cursor:], "/*") {
			lines, endOffset := readBlockComment(source, cursor)
			text := normalizedHeaderText(lines)
			if text != "" {
				return sourceFileHeader{
					Present:         true,
					Text:            text,
					Offset:          cursor,
					FirstCodeOffset: endOffset,
				}
			}
			cursor = endOffset
			continue
		}

		return sourceFileHeader{
			FirstCodeOffset: cursor,
		}
	}
}

// skipUTF8BOM advances past a leading UTF-8 byte-order mark (EF BB BF / U+FEFF).
func skipUTF8BOM(source string, cursor int) int {
	if cursor == 0 && strings.HasPrefix(source, "\ufeff") {
		return len("\ufeff")
	}
	return cursor
}

func skipWhitespace(source string, cursor int) int {
	for cursor < len(source) {
		switch source[cursor] {
		case ' ', '\t', '\r', '\n':
			cursor++
		default:
			return cursor
		}
	}
	return cursor
}

func skipLine(source string, cursor int) int {
	for cursor < len(source) && source[cursor] != '\n' {
		cursor++
	}
	if cursor < len(source) {
		cursor++
	}
	return cursor
}

func readLineCommentGroup(source string, offset int) ([]string, int) {
	cursor := offset
	lines := make([]string, 0)

	for cursor < len(source) && strings.HasPrefix(source[cursor:], "//") {
		cursor += 2
		lineStart := cursor
		cursor = skipLine(source, cursor)
		lineEnd := cursor
		for lineEnd > lineStart && (source[lineEnd-1] == '\n' || source[lineEnd-1] == '\r') {
			lineEnd--
		}
		lines = append(lines, source[lineStart:lineEnd])
		cursor = skipWhitespace(source, cursor)
	}

	return lines, cursor
}

func readBlockComment(source string, offset int) ([]string, int) {
	bodyStart := offset + 2
	cursor := bodyStart
	for cursor+1 < len(source) {
		if source[cursor] == '*' && source[cursor+1] == '/' {
			body := source[bodyStart:cursor]
			return strings.Split(body, "\n"), cursor + 2
		}
		cursor++
	}

	body := source[bodyStart:]
	return strings.Split(body, "\n"), len(source)
}

func normalizedHeaderText(lines []string) string {
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		text := normalizeHeaderLine(line)
		if shouldIgnoreHeaderLine(text) {
			continue
		}
		normalized = append(normalized, text)
	}
	return strings.TrimSpace(strings.Join(normalized, "\n"))
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
	return strings.HasPrefix(line, "Code generated ") && strings.Contains(line, "DO NOT EDIT.")
}

func offsetToLineColumn(source string, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}

	line := 1
	column := 1
	for index := 0; index < offset; index++ {
		if source[index] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}
