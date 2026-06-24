mod common;

use std::fs;

use tempfile::TempDir;
use vet::config::{
    self, CasingStyle, Config, FunctionDocstringPolicy, IndentType, LoadFileRequest,
};

use common::path_string;

#[test]
fn load_file_applies_rule_config() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  max-function-parameters:
    enabled: false
    max: 3
  source-file-header:
    required: true
    min-length: 10
    max-length: 80
  max-source-file-lines:
    max: 100
  max-function-body-lines:
    max: 12
  function-docstring:
    policy: mandatory
  indent:
    type: spaces
    width: 4
  casing:
    enabled: true
    functions: camelCase
    variables: snake_case
    types: UpperCamelCase
    constants: SNAKE_CASE_FULL_CAPS
    ignore-names:
      - generated_name
    ignore-patterns:
      - "^Test[A-Z]"
"#,
    )
    .unwrap();

    let config = config::load_file(LoadFileRequest {
        path: path_string(&path),
        base: Config::default(),
        language: None,
    })
    .unwrap();

    assert!(!config.max_function_parameters.enabled);
    assert_eq!(config.max_function_parameters.max, 3);
    assert!(config.source_file_header.required);
    assert_eq!(config.source_file_header.min_length, 10);
    assert_eq!(config.source_file_header.max_length, 80);
    assert_eq!(config.source_file_lines.max, 100);
    assert_eq!(config.function_body_lines.max, 12);
    assert_eq!(
        config.function_docstring.policy,
        FunctionDocstringPolicy::Mandatory
    );
    assert_eq!(config.indent.r#type, IndentType::Spaces);
    assert_eq!(config.indent.width, 4);
    assert!(config.casing.enabled);
    assert_eq!(config.casing.functions, CasingStyle::CamelCase);
    assert_eq!(config.casing.variables, CasingStyle::SnakeCase);
    assert_eq!(config.casing.types, CasingStyle::UpperCamelCase);
    assert_eq!(config.casing.constants, CasingStyle::SnakeUpperCase);
    assert_eq!(config.casing.ignore_names, vec!["generated_name"]);
    assert_eq!(config.casing.ignore_patterns, vec!["^Test[A-Z]"]);
}

#[test]
fn load_file_applies_language_overrides() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  max-function-parameters:
    enabled: true
    max: 3
  indent:
    type: tabs
    width: 0
  casing:
    enabled: false
    functions: language-default
languages:
  go:
    rules:
      max-function-parameters:
        max: 2
      casing:
        enabled: true
        functions: camelCase
  rust:
    files:
      - src/*.rs
    exclude:
      - "**/*_test.rs"
    rules:
      max-function-parameters:
        max: 5
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

    assert_eq!(config.max_function_parameters.max, 5);
    assert_eq!(config.file_selection.files, vec!["src/*.rs"]);
    assert_eq!(config.file_selection.exclude, vec!["**/*_test.rs"]);
    assert_eq!(config.indent.r#type, IndentType::Spaces);
    assert_eq!(config.indent.width, 4);
    assert!(!config.casing.enabled);
    assert_eq!(config.casing.functions, CasingStyle::LanguageDefault);
}

#[test]
fn load_file_ignores_language_overrides_without_language() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  max-function-parameters:
    max: 3
languages:
  rust:
    rules:
      max-function-parameters:
        max: 5
"#,
    )
    .unwrap();

    let config = config::load_file(LoadFileRequest {
        path: path_string(&path),
        base: Config::default(),
        language: None,
    })
    .unwrap();

    assert_eq!(config.max_function_parameters.max, 3);
}

#[test]
fn load_file_rejects_unknown_fields() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  source-file-header:
    minimum: 10
"#,
    )
    .unwrap();

    assert!(load_config(&path).is_err());
}

#[test]
fn validate_rejects_invalid_header_bounds() {
    let mut config = Config::default();
    config.source_file_header.min_length = 20;
    config.source_file_header.max_length = 10;

    assert!(config::validate(&config).is_err());
}

#[test]
fn validate_rejects_invalid_line_bounds() {
    let mut config = Config::default();
    config.source_file_lines.max = -1;
    assert!(config::validate(&config).is_err());

    let mut config = Config::default();
    config.function_body_lines.max = -1;
    assert!(config::validate(&config).is_err());
}

#[test]
fn load_file_rejects_invalid_function_docstring_policy() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  function-docstring:
    policy: sometimes
"#,
    )
    .unwrap();

    assert!(load_config(&path).is_err());
}

#[test]
fn load_file_rejects_invalid_indent_type() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  indent:
    type: mixed
"#,
    )
    .unwrap();

    assert!(load_config(&path).is_err());
}

#[test]
fn validate_rejects_invalid_indent_width() {
    let mut config = Config::default();
    config.indent.width = -1;

    assert!(config::validate(&config).is_err());
}

#[test]
fn load_file_rejects_invalid_casing_style() {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(
        &path,
        r#"version: 1
rules:
  casing:
    functions: mixed
"#,
    )
    .unwrap();

    assert!(load_config(&path).is_err());
}

#[test]
fn validate_rejects_invalid_casing_ignore_pattern() {
    let mut config = Config::default();
    config.casing.ignore_patterns = vec!["[".to_string()];

    assert!(config::validate(&config).is_err());
}

fn load_config(path: impl AsRef<std::path::Path>) -> Result<Config, config::ConfigError> {
    config::load_file(LoadFileRequest {
        path: path_string(path),
        base: Config::default(),
        language: None,
    })
}
