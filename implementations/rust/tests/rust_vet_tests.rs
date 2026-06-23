use std::{fs, path::Path};

use tempfile::TempDir;
use vet::{
    analysis::{
        AnalyzeFileRequest, Analyzer, RULE_CONSTANT_CASING, RULE_FUNCTION_BODY_LINES,
        RULE_FUNCTION_CASING, RULE_FUNCTION_DOCSTRING, RULE_INDENT_TYPE, RULE_INDENT_WIDTH,
        RULE_MAX_FUNCTION_PARAMETERS, RULE_SOURCE_FILE_HEADER_MAX, RULE_SOURCE_FILE_HEADER_MIN,
        RULE_SOURCE_FILE_HEADER_REQUIRED, RULE_SOURCE_FILE_LINES, RULE_TYPE_CASING,
        RULE_VARIABLE_CASING,
    },
    config::{self, CasingStyle, Config, FunctionDocstringPolicy, IndentType, LoadFileRequest},
    run,
};

#[test]
fn analyzer_reports_functions_with_too_many_parameters() {
    let source = r#"
fn accepted(value: i32) {}

fn rejected(left: i32, right: i32) {}
"#;

    let diagnostics = analyze(Config::default(), source);

    assert_eq!(diagnostics.len(), 1, "{diagnostics:#?}");
    assert_eq!(diagnostics[0].rule_id, RULE_MAX_FUNCTION_PARAMETERS);
    assert_eq!(diagnostics[0].line, 4);
    assert_eq!(diagnostics[0].column, 4);
}

#[test]
fn analyzer_does_not_count_method_receiver_as_parameter() {
    let source = r#"
struct Sample;

impl Sample {
    fn accepted(&self, value: i32) {}
}
"#;

    let diagnostics = analyze(Config::default(), source);

    assert!(diagnostics.is_empty(), "{diagnostics:#?}");
}

#[test]
fn analyzer_reports_source_file_header_rules() {
    let mut config = Config::default();
    config.source_file_header.required = true;
    let diagnostics = analyze(config.clone(), "fn missing() {}\n");
    assert_eq!(diagnostics[0].rule_id, RULE_SOURCE_FILE_HEADER_REQUIRED);

    config.source_file_header.required = false;
    config.source_file_header.min_length = 5;
    let diagnostics = analyze(config.clone(), "// Tiny\nfn accepted() {}\n");
    assert_eq!(diagnostics[0].rule_id, RULE_SOURCE_FILE_HEADER_MIN);

    config.source_file_header.min_length = 0;
    config.source_file_header.max_length = 5;
    let diagnostics = analyze(config, "// Too long\nfn accepted() {}\n");
    assert_eq!(diagnostics[0].rule_id, RULE_SOURCE_FILE_HEADER_MAX);
}

#[test]
fn analyzer_reports_line_body_and_docstring_rules() {
    let mut config = Config::default();
    config.source_file_lines.max = 2;
    config.function_body_lines.max = 1;
    config.function_docstring.policy = FunctionDocstringPolicy::Mandatory;

    let diagnostics = analyze(
        config,
        r#"fn missing() {
    println!("one");
    println!("two");
}
"#,
    );

    let rule_ids = diagnostics
        .iter()
        .map(|diagnostic| diagnostic.rule_id.as_str())
        .collect::<Vec<_>>();
    assert!(rule_ids.contains(&RULE_SOURCE_FILE_LINES));
    assert!(rule_ids.contains(&RULE_FUNCTION_BODY_LINES));
    assert!(rule_ids.contains(&RULE_FUNCTION_DOCSTRING));
}

#[test]
fn analyzer_reports_forbidden_docstring() {
    let mut config = Config::default();
    config.function_docstring.policy = FunctionDocstringPolicy::Forbidden;

    let diagnostics = analyze(
        config,
        r#"/// documented has a docstring.
fn documented() {}
"#,
    );

    assert_eq!(diagnostics.len(), 1, "{diagnostics:#?}");
    assert_eq!(diagnostics[0].rule_id, RULE_FUNCTION_DOCSTRING);
}

#[test]
fn analyzer_reports_indent_diagnostics() {
    let mut config = Config::default();
    config.indent.r#type = IndentType::Spaces;
    let diagnostics = analyze(config.clone(), "fn rejected() {\n\tprintln!(\"one\");\n}\n");
    assert_eq!(diagnostics[0].rule_id, RULE_INDENT_TYPE);

    config.indent.width = 4;
    let diagnostics = analyze(config, "fn rejected() {\n  println!(\"one\");\n}\n");
    assert_eq!(diagnostics[0].rule_id, RULE_INDENT_WIDTH);
}

#[test]
fn analyzer_uses_rust_language_default_indentation() {
    let diagnostics = analyze(
        Config::default(),
        "fn rejected() {\n\tprintln!(\"one\");\n}\n",
    );

    assert_eq!(diagnostics.len(), 1, "{diagnostics:#?}");
    assert_eq!(diagnostics[0].rule_id, RULE_INDENT_TYPE);
}

#[test]
fn analyzer_reports_casing_diagnostics() {
    let mut config = Config::default();
    config.casing.enabled = true;
    config.casing.functions = CasingStyle::SnakeCase;
    config.casing.variables = CasingStyle::SnakeCase;
    config.casing.types = CasingStyle::UpperCamelCase;
    config.casing.constants = CasingStyle::SnakeUpperCase;

    let diagnostics = analyze(
        config,
        r#"const max_connections: usize = 1;

struct user_record;

fn Rejected(BadParam: i32) {
    let BadLocal = BadParam;
}
"#,
    );

    let rule_ids = diagnostics
        .iter()
        .map(|diagnostic| diagnostic.rule_id.as_str())
        .collect::<Vec<_>>();
    for expected in [
        RULE_CONSTANT_CASING,
        RULE_TYPE_CASING,
        RULE_FUNCTION_CASING,
        RULE_VARIABLE_CASING,
    ] {
        assert!(rule_ids.contains(&expected), "{diagnostics:#?}");
    }
}

#[test]
fn analyzer_accepts_rust_language_default_casing() {
    let mut config = Config::default();
    config.casing.enabled = true;

    let diagnostics = analyze(
        config,
        r#"const MAX_CONNECTIONS: usize = 1;
static REQUEST_LIMIT: usize = 2;

struct UserRecord;
enum UserState {}

fn serve_http() {
    let request_id = 1;
}
"#,
    );

    assert!(diagnostics.is_empty(), "{diagnostics:#?}");
}

#[test]
fn config_loads_rules_and_rust_language_overrides() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  max-function-parameters:
    max: 3
  indent:
    type: tabs
    width: 0
languages:
  rust:
    files:
      - src/*.rs
    exclude:
      - "**/*_test.rs"
    rules:
      max-function-parameters:
        max: 2
      indent:
        type: spaces
        width: 4
"#,
    )
    .unwrap();

    let config = config::load_file(LoadFileRequest {
        path: path_string(&path),
        base: Config::default(),
        language: Some("rust".to_string()),
    })
    .unwrap();

    assert_eq!(config.max_function_parameters.max, 2);
    assert_eq!(config.indent.r#type, IndentType::Spaces);
    assert_eq!(config.indent.width, 4);
    assert_eq!(config.file_selection.files, vec!["src/*.rs"]);
    assert_eq!(config.file_selection.exclude, vec!["**/*_test.rs"]);
}

#[test]
fn config_rejects_invalid_values() {
    let mut config = Config::default();
    config.source_file_header.min_length = 20;
    config.source_file_header.max_length = 10;
    assert!(config::validate(&config).is_err());

    let mut config = Config::default();
    config.casing.ignore_patterns = vec!["[".to_string()];
    assert!(config::validate(&config).is_err());
}

#[test]
fn cli_reports_diagnostics() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        "fn rejected(left: i32, right: i32) {}\n",
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([path_string(dir.path())]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET001"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn cli_applies_flags_and_outputs_json() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        r#"fn missing() {
    println!("one");
    println!("two");
}
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--format".to_string(),
        "json".to_string(),
        "--max-source-file-lines".to_string(),
        "2".to_string(),
        "--max-function-body-lines".to_string(),
        "1".to_string(),
        "--function-docstring-policy".to_string(),
        "mandatory".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert_eq!(stderr, "");

    let payload: serde_json::Value = serde_json::from_str(&stdout).unwrap();
    let rule_ids = payload["diagnostics"]
        .as_array()
        .unwrap()
        .iter()
        .map(|item| item["rule_id"].as_str().unwrap().to_string())
        .collect::<Vec<_>>();
    assert_eq!(
        rule_ids,
        vec![
            RULE_SOURCE_FILE_LINES,
            RULE_FUNCTION_BODY_LINES,
            RULE_FUNCTION_DOCSTRING,
        ]
    );
}

#[test]
fn cli_reads_default_config_and_rust_file_selection() {
    let dir = TempDir::new().unwrap();
    let source = dir.path().join("source");
    fs::create_dir(&source).unwrap();
    fs::write(
        source.join("included.rs"),
        "fn rejected(left: i32, right: i32) {}\n",
    )
    .unwrap();
    fs::write(
        source.join("ignored_test.rs"),
        "fn ignored(left: i32, right: i32) {}\n",
    )
    .unwrap();
    fs::write(
        dir.path().join("vet.yaml"),
        format!(
            r#"version: 1
languages:
  rust:
    files:
      - {}/*.rs
    exclude:
      - "**/*_test.rs"
"#,
            path_string(&source)
        ),
    )
    .unwrap();

    let original = std::env::current_dir().unwrap();
    std::env::set_current_dir(dir.path()).unwrap();
    let result = run_cli(Vec::<String>::new());
    std::env::set_current_dir(original).unwrap();

    let (code, stdout, stderr) = result;
    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("included.rs"), "{stdout:?}");
    assert!(!stdout.contains("ignored_test.rs"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn cli_rejects_single_dash_long_config_flag() {
    let (code, _stdout, stderr) = run_cli(["-config".to_string(), "vet.yaml".to_string()]);

    assert_eq!(code, 2);
    assert!(stderr.contains("use -c or --config"), "{stderr:?}");
}

fn analyze(config: Config, source: &str) -> Vec<vet::diagnostic::Diagnostic> {
    Analyzer::new(config)
        .analyze_file(AnalyzeFileRequest {
            path: "sample.rs".to_string(),
            source: source.to_string(),
        })
        .unwrap()
}

fn run_cli<I, S>(args: I) -> (i32, String, String)
where
    I: IntoIterator<Item = S>,
    S: Into<String>,
{
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    let code = run(args, &mut stdout, &mut stderr);
    (
        code,
        String::from_utf8(stdout).unwrap(),
        String::from_utf8(stderr).unwrap(),
    )
}

fn path_string(path: impl AsRef<Path>) -> String {
    path.as_ref().to_string_lossy().into_owned()
}
