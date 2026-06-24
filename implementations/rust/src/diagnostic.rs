use proc_macro2::Span;
use serde::Serialize;

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Severity {
    Error,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct Diagnostic {
    #[serde(rename = "rule_id")]
    pub rule_id: String,
    pub severity: Severity,
    pub message: String,
    pub file: String,
    pub line: usize,
    pub column: usize,
}

impl Diagnostic {
    pub fn new(
        rule_id: impl Into<String>,
        message: impl Into<String>,
        file: impl Into<String>,
        line: usize,
        column: usize,
    ) -> Self {
        Self {
            rule_id: rule_id.into(),
            severity: Severity::Error,
            message: message.into(),
            file: file.into(),
            line,
            column,
        }
    }

    pub fn from_span(
        rule_id: impl Into<String>,
        message: impl Into<String>,
        file: impl Into<String>,
        span: Span,
    ) -> Self {
        let start = span.start();
        Self::new(rule_id, message, file, start.line, start.column + 1)
    }
}

pub fn diagnostic_at_offset(
    rule_id: impl Into<String>,
    message: impl Into<String>,
    file: impl Into<String>,
    source: &str,
    offset: usize,
) -> Diagnostic {
    let (line, column) = location(source, offset);
    Diagnostic::new(rule_id, message, file, line, column)
}

fn location(source: &str, offset: usize) -> (usize, usize) {
    let bounded = offset.min(source.len());
    let mut line = 1;
    let mut column = 1;

    for (index, ch) in source.char_indices() {
        if index >= bounded {
            break;
        }
        if ch == '\n' {
            line += 1;
            column = 1;
        } else {
            column += 1;
        }
    }

    (line, column)
}
