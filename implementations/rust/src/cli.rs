use crate::{
    analysis::{AnalyzeFileRequest, Analyzer},
    config::{self, CasingStyle, Config, FunctionDocstringPolicy, IndentType, LoadFileRequest},
    diagnostic::Diagnostic,
};
use glob::glob;
use serde::Serialize;
use std::{
    collections::{BTreeSet, HashMap},
    fmt, fs,
    io::Write,
    path::Path,
};

pub const VERSION: &str = "0.1.0-dev";
const DEFAULT_CONFIG_FILENAME: &str = "vet.yaml";

#[derive(Default)]
struct CliOptions {
    config_long: Option<String>,
    config_short: Option<String>,
    format: String,
    max_function_parameters: Option<i32>,
    require_file_header: Option<bool>,
    min_file_header_length: Option<i32>,
    max_file_header_length: Option<i32>,
    max_source_file_lines: Option<i32>,
    max_function_body_lines: Option<i32>,
    function_docstring_policy: Option<FunctionDocstringPolicy>,
    indent_type: Option<IndentType>,
    indent_width: Option<i32>,
    casing_enabled: Option<bool>,
    function_casing: Option<CasingStyle>,
    variable_casing: Option<CasingStyle>,
    type_casing: Option<CasingStyle>,
    constant_casing: Option<CasingStyle>,
    version: bool,
    paths: Vec<String>,
}

struct FileCollectionRequest {
    paths: Vec<String>,
    exclude: Vec<String>,
}

struct FileCollector {
    files: Vec<String>,
    seen: BTreeSet<String>,
    exclude: Vec<String>,
}

#[derive(Debug)]
enum CliError {
    Message(String),
    Io(std::io::Error),
    Pattern(glob::PatternError),
}

impl fmt::Display for CliError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CliError::Message(message) => f.write_str(message),
            CliError::Io(err) => err.fmt(f),
            CliError::Pattern(err) => err.fmt(f),
        }
    }
}

impl From<std::io::Error> for CliError {
    fn from(value: std::io::Error) -> Self {
        CliError::Io(value)
    }
}

impl From<glob::PatternError> for CliError {
    fn from(value: glob::PatternError) -> Self {
        CliError::Pattern(value)
    }
}

pub fn run<I, S, O, E>(args: I, stdout: &mut O, stderr: &mut E) -> i32
where
    I: IntoIterator<Item = S>,
    S: Into<String>,
    O: Write,
    E: Write,
{
    let args = args.into_iter().map(Into::into).collect::<Vec<_>>();
    match run_inner(args, stdout, stderr) {
        Ok(code) => code,
        Err(message) => {
            let _ = writeln!(stderr, "vet: {message}");
            2
        }
    }
}

fn run_inner<O: Write, E: Write>(
    args: Vec<String>,
    stdout: &mut O,
    stderr: &mut E,
) -> Result<i32, String> {
    if uses_single_dash_config(&args) {
        return Err("use -c or --config, not -config".to_string());
    }

    let options = parse_options(args)?;
    if options.version {
        writeln!(stdout, "{VERSION}").map_err(|err| err.to_string())?;
        return Ok(0);
    }

    let mut config = Config::default();
    let selected_config = select_config_path(&options)?;
    let selected_config = if selected_config.is_none() {
        default_config_path().map_err(|err| err.to_string())?
    } else {
        selected_config
    };

    if let Some(path) = selected_config {
        config = config::load_file(LoadFileRequest {
            path,
            base: config,
            language: Some("rust".to_string()),
        })
        .map_err(|err| err.to_string())?;
    }

    apply_options(&mut config, &options);
    config::validate(&config).map_err(|err| err.to_string())?;

    let selection = if options.paths.is_empty() {
        let paths = if config.file_selection.files.is_empty() {
            vec![".".to_string()]
        } else {
            config.file_selection.files.clone()
        };
        FileCollectionRequest {
            paths,
            exclude: config.file_selection.exclude.clone(),
        }
    } else {
        FileCollectionRequest {
            paths: options.paths,
            exclude: Vec::new(),
        }
    };

    let files = collect_rust_files(selection).map_err(|err| err.to_string())?;
    let analyzer = Analyzer::new(config);
    let mut diagnostics = Vec::new();
    for file in files {
        let source = fs::read_to_string(&file).map_err(|err| format!("{file}: {err}"))?;
        let file_diagnostics = analyzer
            .analyze_file(AnalyzeFileRequest {
                path: file.clone(),
                source,
            })
            .map_err(|err| format!("{file}: {err}"))?;
        diagnostics.extend(file_diagnostics);
    }

    sort_diagnostics(&mut diagnostics);
    match options.format.as_str() {
        "text" => render_text(stdout, &diagnostics).map_err(|err| err.to_string())?,
        "json" => render_json(stdout, &diagnostics)
            .map_err(|err| format!("failed to write json: {err}"))?,
        other => {
            let _ = writeln!(stderr, "vet: unsupported format {other:?}");
            return Ok(2);
        }
    }

    Ok(if diagnostics.is_empty() { 0 } else { 1 })
}

fn parse_options(args: Vec<String>) -> Result<CliOptions, String> {
    let mut options = CliOptions {
        format: "text".to_string(),
        ..CliOptions::default()
    };
    let mut cursor = 0;
    let value_flags = value_flag_parsers();

    while cursor < args.len() {
        let argument = &args[cursor];
        if argument == "--" {
            options.paths.extend(args.iter().skip(cursor + 1).cloned());
            return Ok(options);
        }

        let (flag, inline_value) = split_inline_value(argument);
        match flag {
            "-c" => {
                let (value, consumed) = flag_value(&args, cursor, inline_value, flag)?;
                options.config_short = Some(value);
                cursor += consumed;
            }
            "--config" => {
                let (value, consumed) = flag_value(&args, cursor, inline_value, flag)?;
                options.config_long = Some(value);
                cursor += consumed;
            }
            "--format" | "-format" => {
                let (value, consumed) = flag_value(&args, cursor, inline_value, flag)?;
                options.format = value;
                cursor += consumed;
            }
            "--require-file-header" | "-require-file-header" => {
                options.require_file_header = Some(optional_bool(inline_value, flag)?);
            }
            "--casing" | "-casing" => {
                options.casing_enabled = Some(optional_bool(inline_value, flag)?);
            }
            "--version" | "-version" => {
                options.version = optional_bool(inline_value, flag)?;
            }
            _ => {
                if let Some(parser) = value_flags.get(flag) {
                    let (value, consumed) = flag_value(&args, cursor, inline_value, flag)?;
                    parser(&mut options, &value, flag)?;
                    cursor += consumed;
                } else if argument.starts_with('-') {
                    return Err(format!("unknown flag {argument}"));
                } else {
                    options.paths.push(argument.clone());
                }
            }
        }

        cursor += 1;
    }

    Ok(options)
}

type ValueParser = fn(&mut CliOptions, &str, &str) -> Result<(), String>;

fn value_flag_parsers() -> HashMap<&'static str, ValueParser> {
    HashMap::from([
        (
            "--max-function-parameters",
            parse_max_function_parameters as ValueParser,
        ),
        ("-max-function-parameters", parse_max_function_parameters),
        ("--min-file-header-length", parse_min_file_header_length),
        ("-min-file-header-length", parse_min_file_header_length),
        ("--max-file-header-length", parse_max_file_header_length),
        ("-max-file-header-length", parse_max_file_header_length),
        ("--max-source-file-lines", parse_max_source_file_lines),
        ("-max-source-file-lines", parse_max_source_file_lines),
        ("--max-function-body-lines", parse_max_function_body_lines),
        ("-max-function-body-lines", parse_max_function_body_lines),
        (
            "--function-docstring-policy",
            parse_function_docstring_policy,
        ),
        (
            "-function-docstring-policy",
            parse_function_docstring_policy,
        ),
        ("--indent-type", parse_indent_type),
        ("-indent-type", parse_indent_type),
        ("--indent-width", parse_indent_width),
        ("-indent-width", parse_indent_width),
        ("--function-casing", parse_function_casing),
        ("-function-casing", parse_function_casing),
        ("--variable-casing", parse_variable_casing),
        ("-variable-casing", parse_variable_casing),
        ("--type-casing", parse_type_casing),
        ("-type-casing", parse_type_casing),
        ("--constant-casing", parse_constant_casing),
        ("-constant-casing", parse_constant_casing),
    ])
}

fn parse_max_function_parameters(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.max_function_parameters = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_min_file_header_length(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.min_file_header_length = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_max_file_header_length(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.max_file_header_length = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_max_source_file_lines(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.max_source_file_lines = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_max_function_body_lines(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.max_function_body_lines = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_function_docstring_policy(
    options: &mut CliOptions,
    value: &str,
    flag: &str,
) -> Result<(), String> {
    options.function_docstring_policy = Some(match value {
        "forbidden" => FunctionDocstringPolicy::Forbidden,
        "optional" => FunctionDocstringPolicy::Optional,
        "mandatory" => FunctionDocstringPolicy::Mandatory,
        _ => return Err(format!("{flag} must be forbidden, optional, or mandatory")),
    });
    Ok(())
}

fn parse_indent_type(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.indent_type = Some(match value {
        "tabs" => IndentType::Tabs,
        "spaces" => IndentType::Spaces,
        "language-default" => IndentType::LanguageDefault,
        _ => return Err(format!("{flag} must be tabs, spaces, or language-default")),
    });
    Ok(())
}

fn parse_indent_width(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.indent_width = Some(int_value(value, flag)?);
    Ok(())
}

fn parse_function_casing(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.function_casing = Some(casing_style(value, flag)?);
    Ok(())
}

fn parse_variable_casing(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.variable_casing = Some(casing_style(value, flag)?);
    Ok(())
}

fn parse_type_casing(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.type_casing = Some(casing_style(value, flag)?);
    Ok(())
}

fn parse_constant_casing(options: &mut CliOptions, value: &str, flag: &str) -> Result<(), String> {
    options.constant_casing = Some(casing_style(value, flag)?);
    Ok(())
}

fn int_value(value: &str, flag: &str) -> Result<i32, String> {
    value
        .parse::<i32>()
        .map_err(|_| format!("{flag} must be an integer"))
}

fn casing_style(value: &str, flag: &str) -> Result<CasingStyle, String> {
    match value {
        "off" => Ok(CasingStyle::Off),
        "language-default" => Ok(CasingStyle::LanguageDefault),
        "camelCase" => Ok(CasingStyle::CamelCase),
        "UpperCamelCase" => Ok(CasingStyle::UpperCamelCase),
        "snake_case" => Ok(CasingStyle::SnakeCase),
        "SNAKE_CASE_FULL_CAPS" => Ok(CasingStyle::SnakeUpperCase),
        _ => Err(format!(
            "{flag} must be off, language-default, camelCase, UpperCamelCase, snake_case, or SNAKE_CASE_FULL_CAPS"
        )),
    }
}

fn optional_bool(value: Option<&str>, flag: &str) -> Result<bool, String> {
    match value {
        None => Ok(true),
        Some("true") => Ok(true),
        Some("false") => Ok(false),
        Some(_) => Err(format!("{flag} must be true or false")),
    }
}

fn split_inline_value(argument: &str) -> (&str, Option<&str>) {
    if let Some(index) = argument.find('=') {
        (&argument[..index], Some(&argument[index + 1..]))
    } else {
        (argument, None)
    }
}

fn flag_value(
    args: &[String],
    cursor: usize,
    inline_value: Option<&str>,
    flag: &str,
) -> Result<(String, usize), String> {
    if let Some(value) = inline_value {
        return Ok((value.to_string(), 0));
    }
    if cursor + 1 >= args.len() {
        return Err(format!("{flag} requires a value"));
    }
    Ok((args[cursor + 1].clone(), 1))
}

fn uses_single_dash_config(args: &[String]) -> bool {
    for arg in args {
        if arg == "--" {
            return false;
        }
        if arg == "-config" || arg.starts_with("-config=") {
            return true;
        }
    }
    false
}

fn select_config_path(options: &CliOptions) -> Result<Option<String>, String> {
    if let (Some(long), Some(short)) = (&options.config_long, &options.config_short) {
        if long != short {
            return Err("-c and --config cannot point to different files".to_string());
        }
    }

    if options.config_short.is_some() {
        return Ok(options.config_short.clone());
    }
    Ok(options.config_long.clone())
}

fn default_config_path() -> Result<Option<String>, std::io::Error> {
    match fs::metadata(DEFAULT_CONFIG_FILENAME) {
        Ok(metadata) => {
            if metadata.is_dir() {
                Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidInput,
                    format!("default config {DEFAULT_CONFIG_FILENAME:?} is a directory"),
                ))
            } else {
                Ok(Some(DEFAULT_CONFIG_FILENAME.to_string()))
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(None),
        Err(err) => Err(std::io::Error::new(
            err.kind(),
            format!("stat default config {DEFAULT_CONFIG_FILENAME:?}: {err}"),
        )),
    }
}

fn apply_options(config: &mut Config, options: &CliOptions) {
    if let Some(max) = options.max_function_parameters {
        config.max_function_parameters.max = max;
    }
    if let Some(required) = options.require_file_header {
        config.source_file_header.required = required;
    }
    if let Some(min_length) = options.min_file_header_length {
        config.source_file_header.min_length = min_length;
    }
    if let Some(max_length) = options.max_file_header_length {
        config.source_file_header.max_length = max_length;
    }
    if let Some(max) = options.max_source_file_lines {
        config.source_file_lines.max = max;
    }
    if let Some(max) = options.max_function_body_lines {
        config.function_body_lines.max = max;
    }
    if let Some(policy) = options.function_docstring_policy {
        config.function_docstring.policy = policy;
    }
    if let Some(indent_type) = options.indent_type {
        config.indent.r#type = indent_type;
    }
    if let Some(width) = options.indent_width {
        config.indent.width = width;
    }
    if let Some(enabled) = options.casing_enabled {
        config.casing.enabled = enabled;
    }
    if let Some(style) = options.function_casing {
        config.casing.enabled = true;
        config.casing.functions = style;
    }
    if let Some(style) = options.variable_casing {
        config.casing.enabled = true;
        config.casing.variables = style;
    }
    if let Some(style) = options.type_casing {
        config.casing.enabled = true;
        config.casing.types = style;
    }
    if let Some(style) = options.constant_casing {
        config.casing.enabled = true;
        config.casing.constants = style;
    }
}

fn collect_rust_files(request: FileCollectionRequest) -> Result<Vec<String>, CliError> {
    let mut collector = FileCollector {
        files: Vec::new(),
        seen: BTreeSet::new(),
        exclude: request.exclude,
    };

    for path in request.paths {
        collector.add_path(&normalize_path(&path))?;
    }

    collector.files.sort();
    Ok(collector.files)
}

impl FileCollector {
    fn add_path(&mut self, path: &str) -> Result<(), CliError> {
        if has_glob_syntax(path) {
            let mut matches = Vec::new();
            for entry in glob(path)? {
                match entry {
                    Ok(path) => matches.push(path),
                    Err(err) => {
                        return Err(CliError::Message(format!(
                            "invalid file pattern: {path}: {err}"
                        )))
                    }
                }
            }
            if matches.is_empty() {
                return Err(CliError::Message(format!(
                    "pattern matched no files: {path}"
                )));
            }
            for path in matches {
                self.add_path(&path_to_string(&path))?;
            }
            return Ok(());
        }

        let metadata = fs::metadata(path).map_err(|err| {
            if err.kind() == std::io::ErrorKind::NotFound {
                CliError::Message(format!("path does not exist: {path}"))
            } else {
                CliError::Io(err)
            }
        })?;
        if metadata.is_dir() {
            self.add_dir(path)
        } else {
            self.add_file(path);
            Ok(())
        }
    }

    fn add_dir(&mut self, path: &str) -> Result<(), CliError> {
        let mut entries = fs::read_dir(path)?.collect::<Result<Vec<_>, _>>()?;
        entries.sort_by_key(|entry| entry.file_name());
        for entry in entries {
            let file_name = entry.file_name();
            let file_name = file_name.to_string_lossy();
            if entry.file_type()?.is_dir() && should_skip_directory(&file_name) {
                continue;
            }
            let child = Path::new(path).join(file_name.as_ref());
            self.add_path(&path_to_string(&child))?;
        }
        Ok(())
    }

    fn add_file(&mut self, path: &str) {
        if !path.ends_with(".rs") || self.seen.contains(path) || self.is_excluded(path) {
            return;
        }
        self.seen.insert(path.to_string());
        self.files.push(path.to_string());
    }

    fn is_excluded(&self, path: &str) -> bool {
        self.exclude
            .iter()
            .any(|pattern| pattern_matches(pattern, path))
    }
}

fn normalize_path(path: &str) -> String {
    if path == "..." {
        return ".".to_string();
    }
    if let Some(base) = path.strip_suffix("/...") {
        if base.is_empty() {
            ".".to_string()
        } else {
            base.to_string()
        }
    } else {
        path.to_string()
    }
}

fn has_glob_syntax(path: &str) -> bool {
    path.contains('*') || path.contains('?') || path.contains('[')
}

fn path_to_string(path: &Path) -> String {
    path.to_string_lossy().into_owned()
}

fn should_skip_directory(name: &str) -> bool {
    matches!(name, ".git" | "target" | "node_modules") || name.starts_with('.')
}

fn pattern_matches(pattern: &str, file_path: &str) -> bool {
    let normalized_pattern = normalize_pattern(pattern);
    let normalized_path = normalize_pattern(file_path);

    if normalized_pattern.is_empty() {
        return false;
    }
    if normalized_pattern == "..." {
        return true;
    }
    if let Some(prefix) = normalized_pattern.strip_suffix("/...") {
        return normalized_path == prefix || normalized_path.starts_with(&format!("{prefix}/"));
    }
    if let Some(prefix) = normalized_pattern.strip_suffix("/**") {
        return normalized_path == prefix || normalized_path.starts_with(&format!("{prefix}/"));
    }
    if let Some(suffix_pattern) = normalized_pattern.strip_prefix("**/") {
        if pattern_matches(suffix_pattern, &normalized_path) {
            return true;
        }
        let parts = normalized_path.split('/').collect::<Vec<_>>();
        for index in 1..parts.len() {
            if pattern_matches(suffix_pattern, &parts[index..].join("/")) {
                return true;
            }
        }
        return false;
    }

    if glob::Pattern::new(&normalized_pattern)
        .map(|pattern| pattern.matches(&normalized_path))
        .unwrap_or(false)
    {
        return true;
    }
    if !normalized_pattern.contains('/') {
        if let Some(base) = Path::new(&normalized_path).file_name() {
            return glob::Pattern::new(&normalized_pattern)
                .map(|pattern| pattern.matches(&base.to_string_lossy()))
                .unwrap_or(false);
        }
    }

    false
}

fn normalize_pattern(value: &str) -> String {
    let mut result = value.replace('\\', "/");
    while result.starts_with("./") {
        result = result[2..].to_string();
    }
    result.trim_end_matches('/').to_string()
}

fn sort_diagnostics(diagnostics: &mut [Diagnostic]) {
    diagnostics.sort_by(|left, right| {
        left.file
            .cmp(&right.file)
            .then(left.line.cmp(&right.line))
            .then(left.column.cmp(&right.column))
            .then(left.rule_id.cmp(&right.rule_id))
    });
}

fn render_text<W: Write>(writer: &mut W, diagnostics: &[Diagnostic]) -> std::io::Result<()> {
    let Some(item) = diagnostics.first() else {
        return Ok(());
    };
    writeln!(
        writer,
        "{}:{}:{}: {}: {}",
        item.file, item.line, item.column, item.rule_id, item.message
    )
}

#[derive(Serialize)]
struct DiagnosticPayload<'a> {
    diagnostics: &'a [Diagnostic],
}

fn render_json<W: Write>(writer: &mut W, diagnostics: &[Diagnostic]) -> std::io::Result<()> {
    serde_json::to_writer_pretty(writer.by_ref(), &DiagnosticPayload { diagnostics })?;
    writeln!(writer)
}
