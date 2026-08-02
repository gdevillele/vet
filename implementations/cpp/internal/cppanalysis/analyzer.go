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
	RuleIndentType               = "VET008"
	RuleIndentWidth              = "VET009"
)

type Analyzer struct {
	config config.Config
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
	return Analyzer{config: cfg}
}

func (a Analyzer) AnalyzeFile(request AnalyzeFileRequest) []diagnostic.Diagnostic {
	source := string(request.Source)
	var diagnostics []diagnostic.Diagnostic
	diagnostics = append(diagnostics, a.checkSourceFileLines(request.Path, source)...)
	diagnostics = append(diagnostics, a.checkIndentation(request.Path, source)...)
	diagnostics = append(diagnostics, a.checkFileHeader(request.Path, source)...)
	return diagnostics
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

func (a Analyzer) checkIndentation(path string, source string) []diagnostic.Diagnostic {
	rule := a.config.Indent
	effectiveType := rule.Type
	if effectiveType == config.IndentLanguageDefault {
		effectiveType = config.IndentSpaces
	}

	ignoredLines := nonCodeLeadingLines(source)
	diagnostics := make([]diagnostic.Diagnostic, 0)
	lines := strings.Split(source, "\n")

	for index, line := range lines {
		lineNumber := index + 1
		if ignoredLines[lineNumber] || strings.TrimSpace(line) == "" {
			continue
		}

		leading := leadingIndent(line)
		if leading == "" {
			continue
		}

		switch effectiveType {
		case config.IndentSpaces:
			if column := strings.IndexRune(leading, '\t'); column >= 0 {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{
					RuleID:   RuleIndentType,
					Severity: diagnostic.SeverityError,
					Message:  "line indentation uses tabs; expected spaces",
					File:     path,
					Line:     lineNumber,
					Column:   column + 1,
				})
				continue
			}
			if rule.Width > 0 && len(leading)%rule.Width != 0 {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{
					RuleID:   RuleIndentWidth,
					Severity: diagnostic.SeverityError,
					Message:  fmt.Sprintf("line indentation has %d spaces; expected a multiple of %d", len(leading), rule.Width),
					File:     path,
					Line:     lineNumber,
					Column:   1,
				})
			}
		case config.IndentTabs:
			if column := strings.IndexRune(leading, ' '); column >= 0 {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{
					RuleID:   RuleIndentType,
					Severity: diagnostic.SeverityError,
					Message:  "line indentation uses spaces; expected tabs",
					File:     path,
					Line:     lineNumber,
					Column:   column + 1,
				})
			}
		}
	}

	return diagnostics
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

func leadingIndent(line string) string {
	index := 0
	for index < len(line) {
		if line[index] != ' ' && line[index] != '\t' {
			break
		}
		index++
	}
	return line[:index]
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

// nonCodeLeadingLines returns line numbers whose leading whitespace should not
// be checked because they fall inside multi-line block comments or string /
// character literals. Continuation lines after the first line of such a span
// are ignored so that content indentation is not treated as source indentation.
func nonCodeLeadingLines(source string) map[int]bool {
	ignored := make(map[int]bool)
	bytes := []byte(source)
	cursor := 0
	line := 1

	// Skip a leading UTF-8 BOM so it is not misread as code.
	if len(bytes) >= 3 && bytes[0] == 0xEF && bytes[1] == 0xBB && bytes[2] == 0xBF {
		cursor = 3
	}

	for cursor < len(bytes) {
		if bytes[cursor] == '\n' {
			line++
			cursor++
			continue
		}

		if cursor+1 < len(bytes) && bytes[cursor] == '/' && bytes[cursor+1] == '/' {
			// Line comments continue across physical lines when the previous
			// physical line ends in an unescaped backslash (C/C++ phase 2).
			startLine := line
			cursor += 2
			for cursor < len(bytes) {
				if bytes[cursor] == '\\' {
					if cursor+1 < len(bytes) && bytes[cursor+1] == '\n' {
						line++
						if line > startLine {
							ignored[line] = true
						}
						cursor += 2
						continue
					}
					if cursor+2 < len(bytes) && bytes[cursor+1] == '\r' && bytes[cursor+2] == '\n' {
						line++
						if line > startLine {
							ignored[line] = true
						}
						cursor += 3
						continue
					}
				}
				if bytes[cursor] == '\n' {
					line++
					cursor++
					break
				}
				cursor++
			}
			continue
		}

		if cursor+1 < len(bytes) && bytes[cursor] == '/' && bytes[cursor+1] == '*' {
			startLine := line
			cursor += 2
			for cursor+1 < len(bytes) {
				if bytes[cursor] == '\n' {
					line++
					if line > startLine {
						ignored[line] = true
					}
					cursor++
					continue
				}
				if bytes[cursor] == '*' && bytes[cursor+1] == '/' {
					cursor += 2
					break
				}
				cursor++
			}
			continue
		}

		if bytes[cursor] == '"' || bytes[cursor] == '\'' {
			quote := bytes[cursor]
			startLine := line
			cursor++
			for cursor < len(bytes) {
				if bytes[cursor] == '\\' && cursor+1 < len(bytes) {
					if bytes[cursor+1] == '\n' {
						line++
						if line > startLine {
							ignored[line] = true
						}
					}
					cursor += 2
					continue
				}
				if bytes[cursor] == '\n' {
					line++
					if line > startLine {
						ignored[line] = true
					}
					cursor++
					continue
				}
				if bytes[cursor] == quote {
					cursor++
					break
				}
				cursor++
			}
			continue
		}

		// C++11 raw string literals: R"delim( ... )delim" (optional L/u/U/u8 prefix).
		if isRawStringPrefix(bytes, cursor) {
			startLine := line
			next, nextLine, ok := skipRawString(bytes, cursor, line)
			if !ok {
				// Fail closed: treat only 'R' as consumed so the following '"'
				// (if any) is scanned as an ordinary string literal.
				cursor++
				continue
			}
			cursor, line = next, nextLine
			for ignoredLine := startLine + 1; ignoredLine <= line; ignoredLine++ {
				ignored[ignoredLine] = true
			}
			continue
		}

		cursor++
	}

	return ignored
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// isRawStringPrefix reports whether bytes[cursor] starts a C++ raw string
// prefix token (optional encoding prefix L/u/U/u8, then R"). The R" sequence
// must begin at a token boundary so identifier tails like fooR" are rejected.
func isRawStringPrefix(bytes []byte, cursor int) bool {
	if cursor >= len(bytes) || bytes[cursor] != 'R' {
		return false
	}
	if cursor+1 >= len(bytes) || bytes[cursor+1] != '"' {
		return false
	}

	if cursor == 0 {
		return true
	}

	// u8R"
	if cursor >= 2 && bytes[cursor-2] == 'u' && bytes[cursor-1] == '8' {
		return cursor == 2 || !isIdentifierByte(bytes[cursor-3])
	}

	// L / u / U encoding prefixes
	prev := bytes[cursor-1]
	if prev == 'L' || prev == 'u' || prev == 'U' {
		return cursor == 1 || !isIdentifierByte(bytes[cursor-2])
	}

	return !isIdentifierByte(prev)
}

// isValidRawStringDChar reports whether b is allowed in a C++ raw-string
// d-char-sequence. The standard forbids space, ), \, control characters, and
// quotation marks among others; d-char-sequence length is at most 16.
func isValidRawStringDChar(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v', '"', '\\', ')', '(':
		return false
	}
	// Keep the scanner ASCII-oriented: reject DEL and non-printable controls.
	if b < 0x20 || b > 0x7E {
		return false
	}
	return true
}

// skipRawString advances past a well-formed raw string starting at R".
// ok is false when the delimiter is invalid; the caller should fail closed.
func skipRawString(bytes []byte, cursor int, line int) (int, int, bool) {
	// bytes[cursor] is 'R', bytes[cursor+1] is '"'.
	cursor += 2
	delimStart := cursor
	for cursor < len(bytes) {
		ch := bytes[cursor]
		if ch == '(' {
			break
		}
		if !isValidRawStringDChar(ch) || cursor-delimStart >= 16 {
			return 0, line, false
		}
		cursor++
	}
	if cursor >= len(bytes) || bytes[cursor] != '(' {
		// Never found a valid opening parenthesis after R".
		return 0, line, false
	}
	delimiter := string(bytes[delimStart:cursor])
	cursor++ // skip (

	closing := ")" + delimiter + `"`
	for cursor < len(bytes) {
		if bytes[cursor] == '\n' {
			line++
			cursor++
			continue
		}
		if cursor+len(closing) <= len(bytes) && string(bytes[cursor:cursor+len(closing)]) == closing {
			return cursor + len(closing), line, true
		}
		cursor++
	}
	// Unterminated raw string: still treat the span as non-code through EOF so
	// partial multi-line raw content is not indent-checked.
	return cursor, line, true
}
