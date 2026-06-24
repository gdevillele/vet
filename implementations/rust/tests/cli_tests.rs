mod common;

use std::fs;

use tempfile::TempDir;
use vet::analysis::{RULE_FUNCTION_BODY_LINES, RULE_FUNCTION_DOCSTRING, RULE_SOURCE_FILE_LINES};

use common::{path_string, run_cli, run_cli_in_dir};

#[test]
fn run_reports_diagnostics() {
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
fn run_allows_configured_parameter_limit() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        "fn accepted(left: i32, right: i32) {}\n",
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--max-function-parameters".to_string(),
        "2".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 0, "stdout={stdout:?} stderr={stderr:?}");
    assert_eq!(stdout, "");
    assert_eq!(stderr, "");
}

#[test]
fn run_accepts_recursive_rust_pattern() {
    let dir = TempDir::new().unwrap();
    let nested = dir.path().join("nested");
    fs::create_dir(&nested).unwrap();
    fs::write(
        nested.join("sample.rs"),
        "fn rejected(left: i32, right: i32) {}\n",
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([path_string(dir.path().join("..."))]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_reports_missing_required_header() {
    let dir = TempDir::new().unwrap();
    fs::write(dir.path().join("sample.rs"), "fn accepted(value: i32) {}\n").unwrap();

    let (code, stdout, stderr) =
        run_cli(["--require-file-header".to_string(), path_string(dir.path())]);

    assert_eq!(code, 1, "stderr={stderr:?}");
    assert!(stdout.contains("VET002"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_rejects_invalid_header_length_bounds() {
    let (code, stdout, stderr) = run_cli([
        "--min-file-header-length".to_string(),
        "10".to_string(),
        "--max-file-header-length".to_string(),
        "5".to_string(),
    ]);

    assert_eq!(code, 2, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_reads_config_file() {
    let dir = TempDir::new().unwrap();
    fs::write(dir.path().join("sample.rs"), "fn accepted(value: i32) {}\n").unwrap();
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        r#"version: 1
rules:
  source-file-header:
    required: true
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--config".to_string(),
        path_string(&config),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET002"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_reads_default_config_file() {
    let dir = TempDir::new().unwrap();
    fs::write(dir.path().join("sample.rs"), "fn accepted(value: i32) {}\n").unwrap();
    fs::write(
        dir.path().join("vet.yaml"),
        r#"version: 1
rules:
  source-file-header:
    required: true
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli_in_dir(dir.path(), [".".to_string()]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET002"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_applies_rust_language_config_override() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        "fn accepted(left: i32, right: i32) {}\n",
    )
    .unwrap();
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        r#"version: 1
rules:
  max-function-parameters:
    max: 1
languages:
  go:
    rules:
      max-function-parameters:
        max: 1
  rust:
    rules:
      max-function-parameters:
        max: 2
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--config".to_string(),
        path_string(&config),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 0, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_uses_rust_language_file_selection_from_config() {
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
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
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

    let (code, stdout, stderr) = run_cli(["--config".to_string(), path_string(&config)]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("included.rs"), "{stdout:?}");
    assert!(!stdout.contains("ignored_test.rs"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_explicit_paths_override_config_file_selection() {
    let dir = TempDir::new().unwrap();
    let configured = dir.path().join("configured.rs");
    fs::write(&configured, "fn rejected(left: i32, right: i32) {}\n").unwrap();

    let explicit = dir.path().join("explicit.rs");
    fs::write(&explicit, "fn accepted(value: i32) {}\n").unwrap();

    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        format!(
            r#"version: 1
languages:
  rust:
    files:
      - {}
"#,
            path_string(&configured)
        ),
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--config".to_string(),
        path_string(&config),
        path_string(&explicit),
    ]);

    assert_eq!(code, 0, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_flags_override_config_file() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        "// Tiny\nfn accepted(value: i32) {}\n",
    )
    .unwrap();
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        r#"version: 1
rules:
  source-file-header:
    min-length: 10
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--config".to_string(),
        path_string(&config),
        "--min-file-header-length".to_string(),
        "4".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 0, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_accepts_short_config_flag() {
    let dir = TempDir::new().unwrap();
    fs::write(dir.path().join("sample.rs"), "fn accepted(value: i32) {}\n").unwrap();
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        r#"version: 1
rules:
  source-file-header:
    required: true
"#,
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "-c".to_string(),
        path_string(&config),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET002"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_rejects_single_dash_long_config_flag() {
    let (code, _stdout, stderr) = run_cli(["-config".to_string(), "vet.yaml".to_string()]);

    assert_eq!(code, 2);
    assert!(stderr.contains("use -c or --config"), "{stderr:?}");
}

#[test]
fn run_rejects_conflicting_config_aliases() {
    let (code, stdout, stderr) = run_cli([
        "-c".to_string(),
        "one.yaml".to_string(),
        "--config".to_string(),
        "two.yaml".to_string(),
    ]);

    assert_eq!(code, 2, "stdout={stdout:?} stderr={stderr:?}");
}

#[test]
fn run_reports_new_rule_diagnostics() {
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
        "--max-source-file-lines".to_string(),
        "2".to_string(),
        "--max-function-body-lines".to_string(),
        "1".to_string(),
        "--function-docstring-policy".to_string(),
        "mandatory".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    let lines = stdout.trim().split('\n').collect::<Vec<_>>();
    assert_eq!(lines.len(), 1, "{stdout:?}");
    assert!(lines[0].contains("VET005"), "{stdout:?}");
    assert!(!stdout.contains("VET006"), "{stdout:?}");
    assert!(!stdout.contains("VET007"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_reports_all_diagnostics_as_json() {
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
fn run_reports_indent_diagnostics() {
    let dir = TempDir::new().unwrap();
    fs::write(
        dir.path().join("sample.rs"),
        "fn rejected() {\n  println!(\"one\");\n}\n",
    )
    .unwrap();

    let (code, stdout, stderr) = run_cli([
        "--indent-type".to_string(),
        "spaces".to_string(),
        "--indent-width".to_string(),
        "4".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET009"), "{stdout:?}");
    assert_eq!(stderr, "");
}

#[test]
fn run_reports_casing_diagnostics() {
    let dir = TempDir::new().unwrap();
    fs::write(dir.path().join("sample.rs"), "fn Rejected() {}\n").unwrap();

    let (code, stdout, stderr) = run_cli([
        "--function-casing".to_string(),
        "snake_case".to_string(),
        path_string(dir.path()),
    ]);

    assert_eq!(code, 1, "stdout={stdout:?} stderr={stderr:?}");
    assert!(stdout.contains("VET010"), "{stdout:?}");
    assert_eq!(stderr, "");
}
