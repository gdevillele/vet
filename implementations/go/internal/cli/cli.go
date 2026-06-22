package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdevillele/vet/implementations/go/internal/config"
	"github.com/gdevillele/vet/implementations/go/internal/diagnostic"
	"github.com/gdevillele/vet/implementations/go/internal/goanalysis"
)

const Version = "0.1.0-dev"

type Invocation struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

type fileCollection struct {
	files []string
	seen  map[string]bool
}

type diagnosticOrder struct {
	Left  diagnostic.Diagnostic
	Right diagnostic.Diagnostic
}

type renderRequest struct {
	Writer      io.Writer
	Diagnostics []diagnostic.Diagnostic
}

func Run(invocation Invocation) int {
	flags := flag.NewFlagSet("vet", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)

	format := flags.String("format", "text", "output format: text or json")
	maxFunctionParameters := flags.Int("max-function-parameters", config.DefaultMaxFunctionParameters, "maximum allowed function parameters")
	version := flags.Bool("version", false, "print version")

	if err := flags.Parse(invocation.Args); err != nil {
		return 2
	}

	if *version {
		fmt.Fprintln(invocation.Stdout, Version)
		return 0
	}

	if *maxFunctionParameters < 0 {
		fmt.Fprintln(invocation.Stderr, "vet: -max-function-parameters must be zero or greater")
		return 2
	}

	cfg := config.Default()
	cfg.MaxFunctionParameters.Max = *maxFunctionParameters

	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	files, err := collectGoFiles(paths)
	if err != nil {
		fmt.Fprintf(invocation.Stderr, "vet: %v\n", err)
		return 2
	}

	analyzer := goanalysis.New(cfg)
	var diagnostics []diagnostic.Diagnostic
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(invocation.Stderr, "vet: %s: %v\n", file, err)
			return 2
		}

		fileDiagnostics, err := analyzer.AnalyzeFile(goanalysis.AnalyzeFileRequest{
			Path:   file,
			Source: source,
		})
		if err != nil {
			fmt.Fprintf(invocation.Stderr, "vet: %s: %v\n", file, err)
			return 2
		}

		diagnostics = append(diagnostics, fileDiagnostics...)
	}

	sortDiagnostics(diagnostics)

	switch *format {
	case "text":
		renderText(renderRequest{
			Writer:      invocation.Stdout,
			Diagnostics: diagnostics,
		})
	case "json":
		if err := renderJSON(renderRequest{
			Writer:      invocation.Stdout,
			Diagnostics: diagnostics,
		}); err != nil {
			fmt.Fprintf(invocation.Stderr, "vet: failed to write json: %v\n", err)
			return 2
		}
	default:
		fmt.Fprintf(invocation.Stderr, "vet: unsupported format %q\n", *format)
		return 2
	}

	if len(diagnostics) > 0 {
		return 1
	}

	return 0
}

func collectGoFiles(paths []string) ([]string, error) {
	collection := fileCollection{
		seen: make(map[string]bool),
	}

	for _, path := range paths {
		if err := collection.addPath(normalizePath(path)); err != nil {
			return nil, err
		}
	}

	sort.Strings(collection.files)
	return collection.files, nil
}

func (c *fileCollection) addPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return c.addDir(path)
	}

	c.addFile(path)
	return nil
}

func (c *fileCollection) addDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && shouldSkipDir(entry.Name()) {
			continue
		}

		if err := c.addPath(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (c *fileCollection) addFile(path string) {
	if !strings.HasSuffix(path, ".go") || c.seen[path] {
		return
	}

	c.seen[path] = true
	c.files = append(c.files, path)
}

func normalizePath(path string) string {
	if path == "..." {
		return "."
	}

	if strings.HasSuffix(path, "/...") {
		base := strings.TrimSuffix(path, "/...")
		if base == "" {
			return "."
		}
		return base
	}

	return path
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "vendor", "node_modules":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func sortDiagnostics(diagnostics []diagnostic.Diagnostic) {
	for index := 1; index < len(diagnostics); index++ {
		current := diagnostics[index]
		cursor := index - 1

		for cursor >= 0 && diagnosticComesAfter(diagnosticOrder{Left: diagnostics[cursor], Right: current}) {
			diagnostics[cursor+1] = diagnostics[cursor]
			cursor--
		}

		diagnostics[cursor+1] = current
	}
}

func diagnosticComesAfter(order diagnosticOrder) bool {
	if order.Left.File != order.Right.File {
		return order.Left.File > order.Right.File
	}
	if order.Left.Line != order.Right.Line {
		return order.Left.Line > order.Right.Line
	}
	if order.Left.Column != order.Right.Column {
		return order.Left.Column > order.Right.Column
	}
	return order.Left.RuleID > order.Right.RuleID
}

func renderText(request renderRequest) {
	for _, item := range request.Diagnostics {
		fmt.Fprintf(request.Writer, "%s:%d:%d: %s: %s\n", item.File, item.Line, item.Column, item.RuleID, item.Message)
	}
}

func renderJSON(request renderRequest) error {
	payload := struct {
		Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
	}{
		Diagnostics: request.Diagnostics,
	}

	encoder := json.NewEncoder(request.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
