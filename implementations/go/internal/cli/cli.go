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

type configPathSelection struct {
	Visited map[string]bool
	Long    string
	Short   string
}

func Run(invocation Invocation) int {
	if usesSingleDashConfig(invocation.Args) {
		fmt.Fprintln(invocation.Stderr, "vet: use -c or --config, not -config")
		return 2
	}

	flags := flag.NewFlagSet("vet", flag.ContinueOnError)
	flags.SetOutput(invocation.Stderr)

	configPath := flags.String("config", "", "path to vet YAML config file")
	configShortPath := flags.String("c", "", "path to vet YAML config file")
	format := flags.String("format", "text", "output format: text or json")
	maxFunctionParameters := flags.Int("max-function-parameters", config.DefaultMaxFunctionParameters, "maximum allowed function parameters")
	requireFileHeader := flags.Bool("require-file-header", false, "require every source file to have a leading header comment")
	minFileHeaderLength := flags.Int("min-file-header-length", 0, "minimum header length in characters; 0 disables the bound")
	maxFileHeaderLength := flags.Int("max-file-header-length", 0, "maximum header length in characters; 0 disables the bound")
	maxSourceFileLines := flags.Int("max-source-file-lines", 0, "maximum source file lines; 0 disables the bound")
	maxFunctionBodyLines := flags.Int("max-function-body-lines", 0, "maximum function body lines; 0 disables the bound")
	functionDocstringPolicy := flags.String("function-docstring-policy", string(config.FunctionDocstringOptional), "function docstring policy: forbidden, optional, or mandatory")
	indentType := flags.String("indent-type", string(config.IndentLanguageDefault), "indent type: tabs, spaces, or language-default")
	indentWidth := flags.Int("indent-width", 0, "space indentation width; 0 disables the width check")
	casingEnabled := flags.Bool("casing", false, "enable identifier casing checks")
	functionCasing := flags.String("function-casing", string(config.CasingLanguageDefault), "function casing style")
	variableCasing := flags.String("variable-casing", string(config.CasingLanguageDefault), "variable casing style")
	typeCasing := flags.String("type-casing", string(config.CasingLanguageDefault), "type casing style")
	constantCasing := flags.String("constant-casing", string(config.CasingLanguageDefault), "constant casing style")
	version := flags.Bool("version", false, "print version")

	if err := flags.Parse(invocation.Args); err != nil {
		return 2
	}

	if *version {
		fmt.Fprintln(invocation.Stdout, Version)
		return 0
	}

	visited := visitedFlags(flags)
	cfg := config.Default()
	selectedConfigPath, err := selectConfigPath(configPathSelection{
		Visited: visited,
		Long:    *configPath,
		Short:   *configShortPath,
	})
	if err != nil {
		fmt.Fprintf(invocation.Stderr, "vet: %v\n", err)
		return 2
	}

	if selectedConfigPath != "" {
		loadedConfig, err := config.LoadFile(config.LoadFileRequest{
			Path:     selectedConfigPath,
			Base:     cfg,
			Language: "go",
		})
		if err != nil {
			fmt.Fprintf(invocation.Stderr, "vet: %v\n", err)
			return 2
		}

		cfg = loadedConfig
	}

	if visited["max-function-parameters"] {
		cfg.MaxFunctionParameters.Max = *maxFunctionParameters
	}
	if visited["require-file-header"] {
		cfg.SourceFileHeader.Required = *requireFileHeader
	}
	if visited["min-file-header-length"] {
		cfg.SourceFileHeader.MinLength = *minFileHeaderLength
	}
	if visited["max-file-header-length"] {
		cfg.SourceFileHeader.MaxLength = *maxFileHeaderLength
	}
	if visited["max-source-file-lines"] {
		cfg.SourceFileLines.Max = *maxSourceFileLines
	}
	if visited["max-function-body-lines"] {
		cfg.FunctionBodyLines.Max = *maxFunctionBodyLines
	}
	if visited["function-docstring-policy"] {
		cfg.FunctionDocstring.Policy = config.FunctionDocstringPolicy(*functionDocstringPolicy)
	}
	if visited["indent-type"] {
		cfg.Indent.Type = config.IndentType(*indentType)
	}
	if visited["indent-width"] {
		cfg.Indent.Width = *indentWidth
	}
	if visited["casing"] {
		cfg.Casing.Enabled = *casingEnabled
	}
	if visited["function-casing"] {
		cfg.Casing.Enabled = true
		cfg.Casing.Functions = config.CasingStyle(*functionCasing)
	}
	if visited["variable-casing"] {
		cfg.Casing.Enabled = true
		cfg.Casing.Variables = config.CasingStyle(*variableCasing)
	}
	if visited["type-casing"] {
		cfg.Casing.Enabled = true
		cfg.Casing.Types = config.CasingStyle(*typeCasing)
	}
	if visited["constant-casing"] {
		cfg.Casing.Enabled = true
		cfg.Casing.Constants = config.CasingStyle(*constantCasing)
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(invocation.Stderr, "vet: %v\n", err)
		return 2
	}

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

func usesSingleDashConfig(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}

		if arg == "-config" || strings.HasPrefix(arg, "-config=") {
			return true
		}
	}

	return false
}

func selectConfigPath(request configPathSelection) (string, error) {
	longVisited := request.Visited["config"]
	shortVisited := request.Visited["c"]
	if longVisited && shortVisited && request.Long != request.Short {
		return "", fmt.Errorf("-c and --config cannot point to different files")
	}

	if shortVisited {
		return request.Short, nil
	}

	if longVisited {
		return request.Long, nil
	}

	return "", nil
}

func visitedFlags(flags *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) {
		visited[item.Name] = true
	})

	return visited
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
