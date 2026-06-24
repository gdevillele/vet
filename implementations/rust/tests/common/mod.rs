#![allow(dead_code)]

use std::{env, path::Path, sync::Mutex};

use vet::{
    analysis::{AnalyzeFileRequest, Analyzer},
    config::Config,
};

static CURRENT_DIR_LOCK: Mutex<()> = Mutex::new(());

pub fn analyze(config: Config, source: &str) -> Vec<vet::diagnostic::Diagnostic> {
    Analyzer::new(config)
        .analyze_file(AnalyzeFileRequest {
            path: "sample.rs".to_string(),
            source: source.to_string(),
        })
        .unwrap()
}

pub fn run_cli<I, S>(args: I) -> (i32, String, String)
where
    I: IntoIterator<Item = S>,
    S: Into<String>,
{
    let _lock = lock_current_dir();
    run_cli_unlocked(args)
}

pub fn run_cli_in_dir<I, S>(path: impl AsRef<Path>, args: I) -> (i32, String, String)
where
    I: IntoIterator<Item = S>,
    S: Into<String>,
{
    let _lock = lock_current_dir();
    let original = env::current_dir().unwrap();
    env::set_current_dir(path).unwrap();
    let result = run_cli_unlocked(args);
    env::set_current_dir(original).unwrap();
    result
}

fn run_cli_unlocked<I, S>(args: I) -> (i32, String, String)
where
    I: IntoIterator<Item = S>,
    S: Into<String>,
{
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    let code = vet::run(args, &mut stdout, &mut stderr);
    (
        code,
        String::from_utf8(stdout).unwrap(),
        String::from_utf8(stderr).unwrap(),
    )
}

pub fn path_string(path: impl AsRef<Path>) -> String {
    path.as_ref().to_string_lossy().into_owned()
}

fn lock_current_dir() -> std::sync::MutexGuard<'static, ()> {
    CURRENT_DIR_LOCK
        .lock()
        .unwrap_or_else(|err| err.into_inner())
}
