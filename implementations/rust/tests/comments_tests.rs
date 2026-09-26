mod common;
use vet::config::Config;

#[test]
fn no_comments_conformance() {
    let source = include_str!("../../../spec/conformance/no-comments/rust/comments.rs");
    let expected: serde_json::Value = serde_json::from_str(include_str!(
        "../../../spec/conformance/no-comments/rust/expected.json"
    ))
    .unwrap();
    for enabled in [false, true] {
        for allow_header in [false, true] {
            let mut config = Config::default();
            config.format.enabled = false;
            config.source_file_header.required = true;
            config.no_comments.enabled = enabled;
            config.no_comments.allow_header = allow_header;
            let diagnostics = common::analyze(config, source);
            let mut want = expected.as_array().unwrap().clone();
            if !allow_header {
                want.insert(0, serde_json::json!({"line": 1, "column": 1}));
            }
            if !enabled {
                want.clear();
            }
            assert_eq!(diagnostics.len(), want.len(), "{diagnostics:?}");
            for (d, position) in diagnostics.iter().zip(want) {
                assert_eq!(d.rule_id, "VET015");
                assert_eq!(d.message, "comment is not allowed");
                assert_eq!(d.line as u64, position["line"].as_u64().unwrap());
                assert_eq!(d.column as u64, position["column"].as_u64().unwrap());
            }
        }
    }
}

#[test]
fn no_comments_config_and_cli() {
    let dir = tempfile::tempdir().unwrap();
    let file = dir.path().join("sample.rs");
    let cfg = dir.path().join("vet.yaml");
    std::fs::write(&file, "// Header\nfn main() {}\n").unwrap();
    std::fs::write(&cfg, "version: 1\nrules:\n  format: { enabled: false }\n  source-file-header: { required: true }\n  no-comments: { enabled: true, allow-header: false }\nlanguages:\n  rust:\n    rules:\n      no-comments: { allow-header: true }\n").unwrap();
    for (flags, expected) in [
        (vec![], 0),
        (vec!["--allow-header-comments=false"], 1),
        (
            vec!["--allow-header-comments=false", "--no-comments=false"],
            0,
        ),
        (vec!["--allow-header-comments=false", "--no-comments"], 1),
        (vec!["--no-comments=invalid"], 2),
        (vec!["--allow-header-comments=invalid"], 2),
    ] {
        let mut args = vec!["--config", cfg.to_str().unwrap(), "--format", "json"];
        args.extend(flags);
        args.push(file.to_str().unwrap());
        let (code, stdout, stderr) = common::run_cli(args);
        assert_eq!(code, expected, "{stdout} {stderr}");
        if code == 1 {
            let payload: serde_json::Value = serde_json::from_str(&stdout).unwrap();
            assert_eq!(payload["diagnostics"].as_array().unwrap().len(), 1);
            assert_eq!(payload["diagnostics"][0]["rule_id"], "VET015");
            assert_eq!(payload["diagnostics"][0]["line"], 1);
            assert_eq!(payload["diagnostics"][0]["column"], 1);
        }
    }
}
