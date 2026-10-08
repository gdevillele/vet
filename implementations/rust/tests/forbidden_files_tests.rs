mod common;

use std::{fs, path::Path};

use tempfile::TempDir;
use vet::{
    analysis::{AnalyzeForbiddenFileRequest, Analyzer, RULE_FORBIDDEN_FILES},
    config::{self, Config, ForbiddenFilesRule, LoadFileRequest},
};

use common::{path_string, run_cli_in_dir};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../spec/conformance/forbidden-files"
);

#[cfg(unix)]
fn symlink(original: impl AsRef<Path>, link: impl AsRef<Path>) -> std::io::Result<()> {
    std::os::unix::fs::symlink(original, link)
}

#[cfg(windows)]
fn symlink(original: impl AsRef<Path>, link: impl AsRef<Path>) -> std::io::Result<()> {
    std::os::windows::fs::symlink_dir(original, link)
}

fn write_file(root: &Path, path: &str) {
    let path = root.join(path);
    fs::create_dir_all(path.parent().unwrap()).unwrap();
    fs::write(path, "").unwrap();
}

fn write_config(dir: &TempDir, name: &str, rule: &str) -> String {
    let path = dir.path().join(name);
    fs::write(
        &path,
        format!("version: 1\nrules:\n  forbidden-files:\n    {rule}\n"),
    )
    .unwrap();
    path_string(path)
}

fn load(data: &str) -> Result<Config, config::ConfigError> {
    let dir = TempDir::new().unwrap();
    let path = dir.path().join("vet.yaml");
    fs::write(&path, format!("version: 1\n{data}")).unwrap();
    config::load_file(LoadFileRequest {
        path: path_string(path),
        base: Config::default(),
        language: Some("rust".to_string()),
    })
}

/// Builds layout.txt and returns false when symlinks are unavailable.
fn build_layout(root: &Path) -> bool {
    let layout = fs::read_to_string(Path::new(FIXTURE).join("layout.txt")).unwrap();
    for line in layout.lines() {
        let fields = line.split_whitespace().collect::<Vec<_>>();
        if fields.is_empty() || fields[0].starts_with('#') {
            continue;
        }
        let path = root.join(fields[1]);
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        match fields.as_slice() {
            ["file", _] => fs::write(&path, "").unwrap(),
            ["symlink", _, target] => {
                if symlink(target, &path).is_err() {
                    return false;
                }
            }
            _ => panic!("invalid layout line {line:?}"),
        }
    }
    true
}

fn rendered_diagnostics(stdout: &str) -> Vec<String> {
    let payload: serde_json::Value = serde_json::from_str(stdout).unwrap();
    payload["diagnostics"]
        .as_array()
        .unwrap()
        .iter()
        .map(|item| {
            assert_eq!(item["severity"], "error");
            format!(
                "{}:{}:{}: {}: {}",
                item["file"].as_str().unwrap(),
                item["line"],
                item["column"],
                item["rule_id"].as_str().unwrap(),
                item["message"].as_str().unwrap()
            )
        })
        .collect()
}

fn diagnostic_files(stdout: &str) -> Vec<String> {
    let payload: serde_json::Value = serde_json::from_str(stdout).unwrap();
    payload["diagnostics"]
        .as_array()
        .unwrap()
        .iter()
        .map(|item| {
            assert_eq!(item["rule_id"], "VET016");
            assert_eq!(item["line"], 1);
            assert_eq!(item["column"], 1);
            item["file"].as_str().unwrap().to_string()
        })
        .collect()
}

#[test]
fn forbidden_files_conformance() {
    let expected = fs::read_to_string(Path::new(FIXTURE).join("expected.txt")).unwrap();
    let expected = expected.trim().lines().collect::<Vec<_>>();
    let root = TempDir::new().unwrap();
    if !build_layout(root.path()) {
        return;
    }
    let config = path_string(Path::new(FIXTURE).join("vet.yaml"));

    let (code, stdout, stderr) =
        run_cli_in_dir(root.path(), ["--config", &config, "--format", "json"]);
    assert_eq!(code, 1, "stdout={stdout} stderr={stderr}");
    assert_eq!(stderr, "");
    assert_eq!(rendered_diagnostics(&stdout), expected);

    let (code, stdout, stderr) = run_cli_in_dir(root.path(), ["--config", &config]);
    assert_eq!(code, 1, "stderr={stderr}");
    assert_eq!(stdout, format!("{}\n", expected[0]));
}

#[test]
fn forbidden_files_config_and_cli() {
    let root = TempDir::new().unwrap();
    for file in [
        "foo/bar.py",
        "other/tool.py",
        "docs/notes.md",
        "design/archive/old.py",
        ".git/hooks/hook.py",
        ".tools/README.md",
    ] {
        write_file(root.path(), file);
    }
    if symlink("missing", root.path().join(".tools").join("broken.py")).is_err() {
        return;
    }

    let configs = TempDir::new().unwrap();
    let enabled = write_config(
        &configs,
        "enabled.yaml",
        "enabled: true\n    patterns: [\"**/*.py\"]\n    exclude: [\"design/archive/**\"]",
    );
    let disabled = write_config(
        &configs,
        "disabled.yaml",
        "enabled: false\n    patterns: [\"**/*.py\"]",
    );
    let empty = write_config(&configs, "empty.yaml", "enabled: true");

    let all = vec![".tools/broken.py", "foo/bar.py", "other/tool.py"];
    for (args, code, files) in [
        (vec!["--config", &enabled], 1, all.clone()),
        (vec!["--config", &enabled, "foo"], 1, vec!["foo/bar.py"]),
        (
            vec!["--config", &enabled, "other/tool.py", "foo/..."],
            1,
            vec!["foo/bar.py", "other/tool.py"],
        ),
        (
            vec!["--config", &enabled, "design", "docs", ".git"],
            0,
            vec![],
        ),
        (
            vec!["--config", &enabled, "--forbidden-files=false"],
            0,
            vec![],
        ),
        (vec!["--config", &disabled], 0, vec![]),
        (
            vec!["--config", &disabled, "--forbidden-files"],
            1,
            vec![
                ".tools/broken.py",
                "design/archive/old.py",
                "foo/bar.py",
                "other/tool.py",
            ],
        ),
        (vec!["--config", &empty], 0, vec![]),
        (vec!["--forbidden-files"], 0, vec![]),
        (
            vec!["--config", &enabled, "--forbidden-files=invalid"],
            2,
            vec![],
        ),
        (vec!["--config", &enabled, "missing"], 2, vec![]),
    ] {
        let mut cli_args = vec!["--format", "json"];
        cli_args.extend(args.iter().copied());
        let (actual, stdout, stderr) = run_cli_in_dir(root.path(), cli_args);
        assert_eq!(actual, code, "{args:?}: stdout={stdout} stderr={stderr}");
        if code != 2 {
            assert_eq!(diagnostic_files(&stdout), files, "{args:?}");
        }
    }
}

#[test]
fn forbidden_files_rejects_language_override() {
    let dir = TempDir::new().unwrap();
    let config = dir.path().join("vet.yaml");
    fs::write(
        &config,
        "version: 1\nlanguages:\n  rust:\n    rules:\n      forbidden-files:\n        enabled: true\n",
    )
    .unwrap();

    let (code, _, stderr) = run_cli_in_dir(dir.path(), ["--config", &path_string(&config)]);
    assert_eq!(code, 2);
    assert!(
        stderr.contains("languages.rust.rules.forbidden-files is not supported"),
        "{stderr}"
    );
}

#[test]
fn load_file_applies_forbidden_files_config() {
    assert_eq!(
        Config::default().forbidden_files,
        ForbiddenFilesRule::default()
    );

    let config = load(
        "rules:\n  forbidden-files:\n    enabled: true\n    patterns: [\"**/*.py\", \"**/pyproject.toml\"]\n    exclude: [\"design/archive/**\"]\n    reason: Use Go for tooling and checks\n",
    )
    .unwrap();
    assert_eq!(
        config.forbidden_files,
        ForbiddenFilesRule {
            enabled: true,
            patterns: vec!["**/*.py".to_string(), "**/pyproject.toml".to_string()],
            exclude: vec!["design/archive/**".to_string()],
            reason: "Use Go for tooling and checks".to_string(),
        }
    );
}

#[test]
fn load_file_rejects_invalid_forbidden_files_config() {
    for (data, want) in [
        (
            "languages:\n  rust:\n    rules: {}\n  go:\n    rules:\n      forbidden-files:\n        enabled: true\n",
            "languages.go.rules.forbidden-files is not supported; forbidden-files is repo-wide and must be set under top-level rules",
        ),
        (
            "rules:\n  forbidden-files:\n    paths: [\"**/*.py\"]\n",
            "unknown field `paths`",
        ),
        (
            "rules:\n  forbidden-files:\n    patterns: [\"\"]\n",
            "forbidden-files.patterns must not contain empty patterns",
        ),
        (
            "rules:\n  forbidden-files:\n    exclude: [\" \"]\n",
            "forbidden-files.exclude must not contain empty patterns",
        ),
    ] {
        let err = load(data).unwrap_err().to_string();
        assert!(err.contains(want), "expected {want:?} in {err:?}");
    }
}

#[test]
fn analyze_forbidden_file() {
    let mut enabled = Config::default();
    enabled.forbidden_files = ForbiddenFilesRule {
        enabled: true,
        patterns: vec![
            "**/*.py".to_string(),
            "**/pyproject.toml".to_string(),
            "**/requirements*.txt".to_string(),
        ],
        exclude: vec!["design/archive/**".to_string()],
        reason: " Use Go for tooling and checks\n".to_string(),
    };
    let mut disabled = enabled.clone();
    disabled.forbidden_files.enabled = false;
    let mut no_patterns = enabled.clone();
    no_patterns.forbidden_files.patterns.clear();

    for (config, path, message) in [
        (&enabled, "foo/bar.py", Some("**/*.py")),
        (&enabled, "pyproject.toml", Some("**/pyproject.toml")),
        (
            &enabled,
            "tools/requirements-dev.txt",
            Some("**/requirements*.txt"),
        ),
        (&enabled, "design/archive/old/tool.py", None),
        (&enabled, "foo/bar.py.txt", None),
        (&disabled, "foo/bar.py", None),
        (&no_patterns, "foo/bar.py", None),
    ] {
        let diagnostics =
            Analyzer::new(config.clone()).analyze_forbidden_file(AnalyzeForbiddenFileRequest {
                path: format!("./{path}"),
                relative_path: path.to_string(),
            });
        match message {
            None => assert!(diagnostics.is_empty(), "{path}: {diagnostics:?}"),
            Some(pattern) => {
                assert_eq!(diagnostics.len(), 1, "{path}: {diagnostics:?}");
                let diagnostic = &diagnostics[0];
                assert_eq!(diagnostic.rule_id, RULE_FORBIDDEN_FILES);
                assert_eq!(
                    diagnostic.message,
                    format!(
                        "file type is forbidden (matches \"{pattern}\") (reason: Use Go for tooling and checks)"
                    )
                );
                assert_eq!(diagnostic.file, format!("./{path}"));
                assert_eq!((diagnostic.line, diagnostic.column), (1, 1));
            }
        }
    }
}
