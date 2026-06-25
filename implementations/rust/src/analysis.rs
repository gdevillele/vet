use crate::{
    config::{CasingRule, CasingStyle, Config, FunctionDocstringPolicy, IndentType},
    diagnostic::{diagnostic_at_offset, Diagnostic},
};
use regex::Regex;
use saphyr::{LoadableYamlNode, MarkedYaml, YamlData};
use std::collections::HashSet;
use syn::{
    visit::{self, Visit},
    Attribute, Block, ExprForLoop, FnArg, ForeignItemFn, Ident, ImplItemConst, ImplItemFn,
    ItemConst, ItemEnum, ItemFn, ItemStatic, ItemStruct, ItemTrait, ItemType, ItemUnion, Local,
    Pat, TraitItemConst, TraitItemFn,
};

pub const RULE_MAX_FUNCTION_PARAMETERS: &str = "VET001";
pub const RULE_SOURCE_FILE_HEADER_REQUIRED: &str = "VET002";
pub const RULE_SOURCE_FILE_HEADER_MIN: &str = "VET003";
pub const RULE_SOURCE_FILE_HEADER_MAX: &str = "VET004";
pub const RULE_SOURCE_FILE_LINES: &str = "VET005";
pub const RULE_FUNCTION_BODY_LINES: &str = "VET006";
pub const RULE_FUNCTION_DOCSTRING: &str = "VET007";
pub const RULE_INDENT_TYPE: &str = "VET008";
pub const RULE_INDENT_WIDTH: &str = "VET009";
pub const RULE_FUNCTION_CASING: &str = "VET010";
pub const RULE_VARIABLE_CASING: &str = "VET011";
pub const RULE_TYPE_CASING: &str = "VET012";
pub const RULE_CONSTANT_CASING: &str = "VET013";
pub const RULE_GITHUB_ACTIONS_PINNED: &str = "VET014";

pub struct Analyzer {
    config: Config,
}

pub struct AnalyzeFileRequest {
    pub path: String,
    pub source: String,
}

pub struct AnalyzeWorkflowFileRequest {
    pub path: String,
    pub source: String,
}

#[derive(Clone, Debug)]
struct SourceFileHeader {
    present: bool,
    text: String,
    offset: usize,
    first_code_offset: usize,
}

struct CasingKind {
    rule_id: &'static str,
    name: &'static str,
    style: CasingStyle,
    language_default: CasingStyle,
}

impl Analyzer {
    pub fn new(config: Config) -> Self {
        Self { config }
    }

    pub fn analyze_file(&self, request: AnalyzeFileRequest) -> Result<Vec<Diagnostic>, syn::Error> {
        let file = syn::parse_file(&request.source)?;
        let mut diagnostics = Vec::new();

        diagnostics.extend(self.check_source_file_lines(&request.path, &request.source));
        diagnostics.extend(self.check_indentation(&request.path, &request.source));
        diagnostics.extend(self.check_file_header(&request.path, &request.source));

        let mut visitor = RustVisitor {
            analyzer: self,
            path: &request.path,
            diagnostics: Vec::new(),
        };
        visitor.visit_file(&file);
        diagnostics.extend(visitor.diagnostics);

        Ok(diagnostics)
    }

    pub fn analyze_workflow_file(
        &self,
        request: AnalyzeWorkflowFileRequest,
    ) -> Result<Vec<Diagnostic>, saphyr::ScanError> {
        if !self.config.github_actions_pinned.enabled {
            return Ok(Vec::new());
        }

        let documents = MarkedYaml::load_from_str(&request.source)?;
        let mut diagnostics = Vec::new();
        for document in &documents {
            diagnostics.extend(check_workflow_jobs(&request.path, document));
        }

        Ok(diagnostics)
    }

    fn check_source_file_lines(&self, path: &str, source: &str) -> Vec<Diagnostic> {
        let max = self.config.source_file_lines.max;
        if max <= 0 {
            return Vec::new();
        }

        let count = source_line_count(source);
        if count <= max as usize {
            return Vec::new();
        }

        vec![Diagnostic::new(
            RULE_SOURCE_FILE_LINES,
            format!("source file has {count} lines; maximum allowed is {max}"),
            path,
            1,
            1,
        )]
    }

    fn check_indentation(&self, path: &str, source: &str) -> Vec<Diagnostic> {
        let rule = &self.config.indent;
        let effective_type = match rule.r#type {
            IndentType::LanguageDefault => IndentType::Spaces,
            other => other,
        };
        let ignored_lines = string_literal_lines(source);
        let mut diagnostics = Vec::new();
        let mut offset = 0;

        for (index, line) in source.split('\n').enumerate() {
            let line_number = index + 1;
            let line_offset = offset;
            offset += line.len() + 1;

            if ignored_lines.contains(&line_number) || line.trim().is_empty() {
                continue;
            }

            let leading = leading_indent(line);
            if leading.is_empty() {
                continue;
            }

            match effective_type {
                IndentType::Spaces => {
                    if let Some(column) = leading.find('\t') {
                        diagnostics.push(diagnostic_at_offset(
                            RULE_INDENT_TYPE,
                            "line indentation uses tabs; expected spaces",
                            path,
                            source,
                            line_offset + column,
                        ));
                        continue;
                    }
                    if rule.width > 0 && leading.len() % rule.width as usize != 0 {
                        diagnostics.push(diagnostic_at_offset(
                            RULE_INDENT_WIDTH,
                            format!(
                                "line indentation has {} spaces; expected a multiple of {}",
                                leading.len(),
                                rule.width
                            ),
                            path,
                            source,
                            line_offset,
                        ));
                    }
                }
                IndentType::Tabs => {
                    if let Some(column) = leading.find(' ') {
                        diagnostics.push(diagnostic_at_offset(
                            RULE_INDENT_TYPE,
                            "line indentation uses spaces; expected tabs",
                            path,
                            source,
                            line_offset + column,
                        ));
                    }
                }
                IndentType::LanguageDefault => {}
            }
        }

        diagnostics
    }

    fn check_file_header(&self, path: &str, source: &str) -> Vec<Diagnostic> {
        let rule = &self.config.source_file_header;
        let header = find_source_file_header(source);

        if !header.present {
            if !rule.required {
                return Vec::new();
            }
            return vec![diagnostic_at_offset(
                RULE_SOURCE_FILE_HEADER_REQUIRED,
                "source file has no header",
                path,
                source,
                header.first_code_offset,
            )];
        }

        let length = header.text.chars().count();
        let mut diagnostics = Vec::new();
        if rule.min_length > 0 && length < rule.min_length as usize {
            diagnostics.push(diagnostic_at_offset(
                RULE_SOURCE_FILE_HEADER_MIN,
                format!(
                    "file header has {length} characters; minimum allowed is {}",
                    rule.min_length
                ),
                path,
                source,
                header.offset,
            ));
        }
        if rule.max_length > 0 && length > rule.max_length as usize {
            diagnostics.push(diagnostic_at_offset(
                RULE_SOURCE_FILE_HEADER_MAX,
                format!(
                    "file header has {length} characters; maximum allowed is {}",
                    rule.max_length
                ),
                path,
                source,
                header.offset,
            ));
        }

        diagnostics
    }
}

fn check_workflow_jobs(path: &str, document: &MarkedYaml<'_>) -> Vec<Diagnostic> {
    let Some(jobs) = yaml_mapping_value(document, "jobs") else {
        return Vec::new();
    };
    let Some(jobs_mapping) = yaml_data(jobs).as_mapping() else {
        return Vec::new();
    };

    let mut diagnostics = Vec::new();
    for (_job_name, job) in jobs_mapping {
        let Some(steps) = yaml_mapping_value(job, "steps") else {
            continue;
        };
        let Some(step_items) = yaml_data(steps).as_vec() else {
            continue;
        };

        for step in step_items {
            let Some(uses) = yaml_mapping_value(step, "uses") else {
                continue;
            };
            let Some(action) = yaml_data(uses).as_str() else {
                continue;
            };
            if github_action_pinned(action) {
                continue;
            }

            diagnostics.push(Diagnostic::new(
                RULE_GITHUB_ACTIONS_PINNED,
                format!("GitHub action {action:?} must be pinned to a full-length commit SHA"),
                path,
                uses.span.start.line(),
                uses.span.start.col(),
            ));
        }
    }

    diagnostics
}

fn yaml_mapping_value<'a>(node: &'a MarkedYaml<'a>, key: &str) -> Option<&'a MarkedYaml<'a>> {
    yaml_data(node).as_mapping_get(key)
}

fn yaml_data<'a>(node: &'a MarkedYaml<'a>) -> &'a YamlData<'a, MarkedYaml<'a>> {
    match &node.data {
        YamlData::Tagged(_, inner) => yaml_data(inner),
        data => data,
    }
}

fn github_action_pinned(action: &str) -> bool {
    if action.starts_with("./") || action.starts_with("docker://") {
        return true;
    }

    let Some((_, reference)) = action.rsplit_once('@') else {
        return false;
    };
    reference.len() == 40 && reference.bytes().all(|byte| byte.is_ascii_hexdigit())
}

struct RustVisitor<'a> {
    analyzer: &'a Analyzer,
    path: &'a str,
    diagnostics: Vec<Diagnostic>,
}

impl RustVisitor<'_> {
    fn check_named_function(
        &mut self,
        name: &Ident,
        inputs: &syn::punctuated::Punctuated<FnArg, syn::token::Comma>,
        body: Option<&Block>,
        attrs: &[Attribute],
    ) {
        self.check_function_parameters(name, inputs);
        self.check_function_body_lines(name, body);
        self.check_function_docstring(name, attrs);
        self.check_identifier_casing(
            name,
            CasingKind {
                rule_id: RULE_FUNCTION_CASING,
                name: "function",
                style: self.analyzer.config.casing.functions,
                language_default: CasingStyle::SnakeCase,
            },
        );
        self.check_function_parameter_casing(inputs);
    }

    fn check_function_parameters(
        &mut self,
        name: &Ident,
        inputs: &syn::punctuated::Punctuated<FnArg, syn::token::Comma>,
    ) {
        let rule = &self.analyzer.config.max_function_parameters;
        if !rule.enabled {
            return;
        }

        let count = inputs
            .iter()
            .filter(|arg| matches!(arg, FnArg::Typed(_)))
            .count();
        if count <= rule.max as usize {
            return;
        }

        let name_text = name.to_string();
        self.diagnostics.push(Diagnostic::from_span(
            RULE_MAX_FUNCTION_PARAMETERS,
            format!(
                "{name_text} has {count} parameters; maximum allowed is {}",
                rule.max
            ),
            self.path,
            name.span(),
        ));
    }

    fn check_function_body_lines(&mut self, name: &Ident, body: Option<&Block>) {
        let max = self.analyzer.config.function_body_lines.max;
        if max <= 0 {
            return;
        }
        let Some(body) = body else {
            return;
        };

        let count = function_body_line_count(body);
        if count <= max as usize {
            return;
        }

        let name_text = name.to_string();
        self.diagnostics.push(Diagnostic::from_span(
            RULE_FUNCTION_BODY_LINES,
            format!("{name_text} body has {count} lines; maximum allowed is {max}"),
            self.path,
            name.span(),
        ));
    }

    fn check_function_docstring(&mut self, name: &Ident, attrs: &[Attribute]) {
        let policy = self.analyzer.config.function_docstring.policy;
        if policy == FunctionDocstringPolicy::Optional {
            return;
        }

        let has_docstring = attrs.iter().any(|attr| attr.path().is_ident("doc"));
        if policy == FunctionDocstringPolicy::Mandatory && has_docstring {
            return;
        }
        if policy == FunctionDocstringPolicy::Forbidden && !has_docstring {
            return;
        }

        let name_text = name.to_string();
        let message = if policy == FunctionDocstringPolicy::Forbidden {
            format!("{name_text} must not have a docstring")
        } else {
            format!("{name_text} must have a docstring")
        };
        self.diagnostics.push(Diagnostic::from_span(
            RULE_FUNCTION_DOCSTRING,
            message,
            self.path,
            name.span(),
        ));
    }

    fn check_function_parameter_casing(
        &mut self,
        inputs: &syn::punctuated::Punctuated<FnArg, syn::token::Comma>,
    ) {
        for input in inputs {
            if let FnArg::Typed(typed) = input {
                self.check_pattern_variable_casing(&typed.pat);
            }
        }
    }

    fn check_pattern_variable_casing(&mut self, pattern: &Pat) {
        for ident in pattern_identifiers(pattern) {
            self.check_identifier_casing(
                ident,
                CasingKind {
                    rule_id: RULE_VARIABLE_CASING,
                    name: "variable",
                    style: self.analyzer.config.casing.variables,
                    language_default: CasingStyle::SnakeCase,
                },
            );
        }
    }

    fn check_identifier_casing(&mut self, ident: &Ident, kind: CasingKind) {
        let rule = &self.analyzer.config.casing;
        if !rule.enabled {
            return;
        }

        let name = ident.to_string();
        if should_ignore_identifier_casing(rule, &name) {
            return;
        }

        let style = if kind.style == CasingStyle::LanguageDefault {
            kind.language_default
        } else {
            kind.style
        };
        if style == CasingStyle::Off || casing_style_matches(style, &name) {
            return;
        }

        self.diagnostics.push(Diagnostic::from_span(
            kind.rule_id,
            format!("{} {:?} must use {}", kind.name, name, style),
            self.path,
            ident.span(),
        ));
    }

    fn check_type_casing(&mut self, ident: &Ident) {
        self.check_identifier_casing(
            ident,
            CasingKind {
                rule_id: RULE_TYPE_CASING,
                name: "type",
                style: self.analyzer.config.casing.types,
                language_default: CasingStyle::UpperCamelCase,
            },
        );
    }

    fn check_constant_casing(&mut self, ident: &Ident) {
        self.check_identifier_casing(
            ident,
            CasingKind {
                rule_id: RULE_CONSTANT_CASING,
                name: "constant",
                style: self.analyzer.config.casing.constants,
                language_default: CasingStyle::SnakeUpperCase,
            },
        );
    }
}

impl<'ast> Visit<'ast> for RustVisitor<'_> {
    fn visit_item_fn(&mut self, node: &'ast ItemFn) {
        self.check_named_function(
            &node.sig.ident,
            &node.sig.inputs,
            Some(&node.block),
            &node.attrs,
        );
        visit::visit_item_fn(self, node);
    }

    fn visit_impl_item_fn(&mut self, node: &'ast ImplItemFn) {
        self.check_named_function(
            &node.sig.ident,
            &node.sig.inputs,
            Some(&node.block),
            &node.attrs,
        );
        visit::visit_impl_item_fn(self, node);
    }

    fn visit_trait_item_fn(&mut self, node: &'ast TraitItemFn) {
        self.check_named_function(
            &node.sig.ident,
            &node.sig.inputs,
            node.default.as_ref(),
            &node.attrs,
        );
        visit::visit_trait_item_fn(self, node);
    }

    fn visit_foreign_item_fn(&mut self, node: &'ast ForeignItemFn) {
        self.check_named_function(&node.sig.ident, &node.sig.inputs, None, &node.attrs);
        visit::visit_foreign_item_fn(self, node);
    }

    fn visit_local(&mut self, node: &'ast Local) {
        self.check_pattern_variable_casing(&node.pat);
        visit::visit_local(self, node);
    }

    fn visit_expr_for_loop(&mut self, node: &'ast ExprForLoop) {
        self.check_pattern_variable_casing(&node.pat);
        visit::visit_expr_for_loop(self, node);
    }

    fn visit_item_struct(&mut self, node: &'ast ItemStruct) {
        self.check_type_casing(&node.ident);
        visit::visit_item_struct(self, node);
    }

    fn visit_item_enum(&mut self, node: &'ast ItemEnum) {
        self.check_type_casing(&node.ident);
        visit::visit_item_enum(self, node);
    }

    fn visit_item_trait(&mut self, node: &'ast ItemTrait) {
        self.check_type_casing(&node.ident);
        visit::visit_item_trait(self, node);
    }

    fn visit_item_union(&mut self, node: &'ast ItemUnion) {
        self.check_type_casing(&node.ident);
        visit::visit_item_union(self, node);
    }

    fn visit_item_type(&mut self, node: &'ast ItemType) {
        self.check_type_casing(&node.ident);
        visit::visit_item_type(self, node);
    }

    fn visit_item_const(&mut self, node: &'ast ItemConst) {
        self.check_constant_casing(&node.ident);
        visit::visit_item_const(self, node);
    }

    fn visit_item_static(&mut self, node: &'ast ItemStatic) {
        self.check_constant_casing(&node.ident);
        visit::visit_item_static(self, node);
    }

    fn visit_impl_item_const(&mut self, node: &'ast ImplItemConst) {
        self.check_constant_casing(&node.ident);
        visit::visit_impl_item_const(self, node);
    }

    fn visit_trait_item_const(&mut self, node: &'ast TraitItemConst) {
        self.check_constant_casing(&node.ident);
        visit::visit_trait_item_const(self, node);
    }
}

fn function_body_line_count(body: &Block) -> usize {
    let span = body.brace_token.span;
    let start = span.open().start().line;
    let end = span.close().start().line;
    end.saturating_sub(start + 1)
}

fn source_line_count(source: &str) -> usize {
    if source.is_empty() {
        return 0;
    }
    let mut count = 1 + source
        .as_bytes()
        .iter()
        .filter(|byte| **byte == b'\n')
        .count();
    if source.ends_with('\n') {
        count -= 1;
    }
    count
}

fn find_source_file_header(source: &str) -> SourceFileHeader {
    let mut cursor = 0;
    let mut skipped_shebang = false;

    loop {
        cursor = skip_whitespace(source, cursor);
        if cursor >= source.len() {
            return SourceFileHeader {
                present: false,
                text: String::new(),
                offset: 0,
                first_code_offset: cursor,
            };
        }

        if !skipped_shebang
            && source[cursor..].starts_with("#!")
            && !source[cursor..].starts_with("#![")
        {
            skipped_shebang = true;
            cursor = skip_line(source, cursor);
            continue;
        }

        if source[cursor..].starts_with("//") {
            let (lines, end_offset) = read_line_comment_group(source, cursor);
            let text = normalized_header_text(&lines);
            if !text.is_empty() {
                return SourceFileHeader {
                    present: true,
                    text,
                    offset: cursor,
                    first_code_offset: end_offset,
                };
            }
            cursor = end_offset;
            continue;
        }

        if source[cursor..].starts_with("/*") {
            let (lines, end_offset) = read_block_comment(source, cursor);
            let text = normalized_header_text(&lines);
            if !text.is_empty() {
                return SourceFileHeader {
                    present: true,
                    text,
                    offset: cursor,
                    first_code_offset: end_offset,
                };
            }
            cursor = end_offset;
            continue;
        }

        return SourceFileHeader {
            present: false,
            text: String::new(),
            offset: 0,
            first_code_offset: cursor,
        };
    }
}

fn skip_whitespace(source: &str, mut cursor: usize) -> usize {
    let bytes = source.as_bytes();
    while cursor < bytes.len() && matches!(bytes[cursor], b' ' | b'\t' | b'\r' | b'\n') {
        cursor += 1;
    }
    cursor
}

fn skip_line(source: &str, mut cursor: usize) -> usize {
    let bytes = source.as_bytes();
    while cursor < bytes.len() && bytes[cursor] != b'\n' {
        cursor += 1;
    }
    if cursor < bytes.len() {
        cursor += 1;
    }
    cursor
}

fn read_line_comment_group(source: &str, offset: usize) -> (Vec<String>, usize) {
    let mut cursor = offset;
    let mut lines = Vec::new();

    while cursor < source.len() && source[cursor..].starts_with("//") {
        cursor += 2;
        let line_start = cursor;
        cursor = skip_line(source, cursor);
        let mut line_end = cursor;
        while line_end > line_start && matches!(source.as_bytes()[line_end - 1], b'\n' | b'\r') {
            line_end -= 1;
        }
        lines.push(source[line_start..line_end].to_string());
        cursor = skip_whitespace(source, cursor);
    }

    (lines, cursor)
}

fn read_block_comment(source: &str, offset: usize) -> (Vec<String>, usize) {
    let body_start = offset + 2;
    let mut cursor = body_start;
    while cursor + 1 < source.len() {
        if source.as_bytes()[cursor] == b'*' && source.as_bytes()[cursor + 1] == b'/' {
            let body = source[body_start..cursor].to_string();
            return (body.split('\n').map(str::to_string).collect(), cursor + 2);
        }
        cursor += 1;
    }

    let body = source[body_start..].to_string();
    (body.split('\n').map(str::to_string).collect(), source.len())
}

fn normalized_header_text(lines: &[String]) -> String {
    lines
        .iter()
        .filter_map(|line| {
            let normalized = normalize_header_line(line);
            if should_ignore_header_line(&normalized) {
                None
            } else {
                Some(normalized)
            }
        })
        .collect::<Vec<_>>()
        .join("\n")
        .trim()
        .to_string()
}

fn normalize_header_line(line: &str) -> String {
    let trimmed = line.trim();
    let trimmed = trimmed.strip_prefix('*').unwrap_or(trimmed);
    trimmed.trim().to_string()
}

fn should_ignore_header_line(line: &str) -> bool {
    line.is_empty()
        || (line.starts_with("Code generated ") && line.contains("DO NOT EDIT."))
        || line.starts_with("rustfmt::skip")
}

fn leading_indent(line: &str) -> &str {
    let end = line
        .bytes()
        .take_while(|byte| matches!(byte, b' ' | b'\t'))
        .count();
    &line[..end]
}

fn string_literal_lines(source: &str) -> HashSet<usize> {
    let mut lines = HashSet::new();
    let bytes = source.as_bytes();
    let mut cursor = 0;
    let mut line = 1;

    while cursor < bytes.len() {
        if bytes[cursor] == b'\n' {
            line += 1;
            cursor += 1;
            continue;
        }

        if cursor + 1 < bytes.len() && &bytes[cursor..cursor + 2] == b"//" {
            cursor = skip_line(source, cursor);
            line += 1;
            continue;
        }

        if cursor + 1 < bytes.len() && &bytes[cursor..cursor + 2] == b"/*" {
            cursor += 2;
            while cursor + 1 < bytes.len() {
                if bytes[cursor] == b'\n' {
                    line += 1;
                }
                if bytes[cursor] == b'*' && bytes[cursor + 1] == b'/' {
                    cursor += 2;
                    break;
                }
                cursor += 1;
            }
            continue;
        }

        if let Some((prefix_len, hashes)) = raw_string_prefix(bytes, cursor) {
            let start_line = line;
            cursor += prefix_len;
            while cursor < bytes.len() {
                if bytes[cursor] == b'\n' {
                    line += 1;
                    if line > start_line {
                        lines.insert(line);
                    }
                    cursor += 1;
                    continue;
                }
                if bytes[cursor] == b'"'
                    && cursor + 1 + hashes <= bytes.len()
                    && bytes[cursor + 1..cursor + 1 + hashes]
                        .iter()
                        .all(|byte| *byte == b'#')
                {
                    cursor += 1 + hashes;
                    break;
                }
                cursor += 1;
            }
            continue;
        }

        if bytes[cursor] == b'"' {
            let start_line = line;
            cursor += 1;
            let mut escaped = false;
            while cursor < bytes.len() {
                if bytes[cursor] == b'\n' {
                    line += 1;
                    if line > start_line {
                        lines.insert(line);
                    }
                    cursor += 1;
                    escaped = false;
                    continue;
                }
                if bytes[cursor] == b'"' && !escaped {
                    cursor += 1;
                    break;
                }
                escaped = bytes[cursor] == b'\\' && !escaped;
                if bytes[cursor] != b'\\' {
                    escaped = false;
                }
                cursor += 1;
            }
            continue;
        }

        cursor += 1;
    }

    lines
}

fn raw_string_prefix(bytes: &[u8], cursor: usize) -> Option<(usize, usize)> {
    let mut offset = cursor;
    if bytes.get(offset) == Some(&b'b') {
        offset += 1;
    }
    if bytes.get(offset) != Some(&b'r') {
        return None;
    }
    offset += 1;
    let hash_start = offset;
    while bytes.get(offset) == Some(&b'#') {
        offset += 1;
    }
    if bytes.get(offset) != Some(&b'"') {
        return None;
    }
    Some((offset - cursor + 1, offset - hash_start))
}

fn pattern_identifiers(pattern: &Pat) -> Vec<&Ident> {
    let mut idents = Vec::new();
    collect_pattern_identifiers(pattern, &mut idents);
    idents
}

fn collect_pattern_identifiers<'a>(pattern: &'a Pat, idents: &mut Vec<&'a Ident>) {
    match pattern {
        Pat::Ident(item) => {
            idents.push(&item.ident);
            if let Some((_, subpattern)) = &item.subpat {
                collect_pattern_identifiers(subpattern, idents);
            }
        }
        Pat::Or(item) => {
            for case in &item.cases {
                collect_pattern_identifiers(case, idents);
            }
        }
        Pat::Paren(item) => collect_pattern_identifiers(&item.pat, idents),
        Pat::Reference(item) => collect_pattern_identifiers(&item.pat, idents),
        Pat::Slice(item) => {
            for elem in &item.elems {
                collect_pattern_identifiers(elem, idents);
            }
        }
        Pat::Struct(item) => {
            for field in &item.fields {
                collect_pattern_identifiers(&field.pat, idents);
            }
        }
        Pat::Tuple(item) => {
            for elem in &item.elems {
                collect_pattern_identifiers(elem, idents);
            }
        }
        Pat::TupleStruct(item) => {
            for elem in &item.elems {
                collect_pattern_identifiers(elem, idents);
            }
        }
        Pat::Type(item) => collect_pattern_identifiers(&item.pat, idents),
        _ => {}
    }
}

fn casing_style_matches(style: CasingStyle, name: &str) -> bool {
    let pattern = match style {
        CasingStyle::CamelCase => r"^[a-z][A-Za-z0-9]*$",
        CasingStyle::UpperCamelCase => r"^[A-Z][A-Za-z0-9]*$",
        CasingStyle::SnakeCase => r"^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$",
        CasingStyle::SnakeUpperCase => r"^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$",
        CasingStyle::Off => return true,
        CasingStyle::LanguageDefault => return false,
    };
    Regex::new(pattern)
        .expect("valid casing pattern")
        .is_match(name)
}

fn should_ignore_identifier_casing(rule: &CasingRule, name: &str) -> bool {
    if name == "_" || rule.ignore_names.iter().any(|ignored| ignored == name) {
        return true;
    }

    rule.ignore_patterns
        .iter()
        .filter_map(|pattern| Regex::new(pattern).ok())
        .any(|pattern| pattern.is_match(name))
}
