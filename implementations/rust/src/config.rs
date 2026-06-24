use regex::Regex;
use serde::Deserialize;
use std::{fmt, fs};

pub const DEFAULT_MAX_FUNCTION_PARAMETERS: i32 = 1;

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Config {
    pub max_function_parameters: MaxFunctionParametersRule,
    pub source_file_header: SourceFileHeaderRule,
    pub source_file_lines: SourceFileLinesRule,
    pub function_body_lines: FunctionBodyLinesRule,
    pub function_docstring: FunctionDocstringRule,
    pub indent: IndentRule,
    pub casing: CasingRule,
    pub file_selection: FileSelection,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MaxFunctionParametersRule {
    pub enabled: bool,
    pub max: i32,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct SourceFileHeaderRule {
    pub required: bool,
    pub min_length: i32,
    pub max_length: i32,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct SourceFileLinesRule {
    pub max: i32,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FunctionBodyLinesRule {
    pub max: i32,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq)]
#[serde(rename_all = "kebab-case")]
pub enum FunctionDocstringPolicy {
    Forbidden,
    Optional,
    Mandatory,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FunctionDocstringRule {
    pub policy: FunctionDocstringPolicy,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq)]
#[serde(rename_all = "kebab-case")]
pub enum IndentType {
    Tabs,
    Spaces,
    LanguageDefault,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct IndentRule {
    pub r#type: IndentType,
    pub width: i32,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq)]
pub enum CasingStyle {
    #[serde(rename = "off")]
    Off,
    #[serde(rename = "language-default")]
    LanguageDefault,
    #[serde(rename = "camelCase")]
    CamelCase,
    #[serde(rename = "UpperCamelCase")]
    UpperCamelCase,
    #[serde(rename = "snake_case")]
    SnakeCase,
    #[serde(rename = "SNAKE_CASE_FULL_CAPS")]
    SnakeUpperCase,
}

impl fmt::Display for CasingStyle {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let value = match self {
            CasingStyle::Off => "off",
            CasingStyle::LanguageDefault => "language-default",
            CasingStyle::CamelCase => "camelCase",
            CasingStyle::UpperCamelCase => "UpperCamelCase",
            CasingStyle::SnakeCase => "snake_case",
            CasingStyle::SnakeUpperCase => "SNAKE_CASE_FULL_CAPS",
        };
        f.write_str(value)
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CasingRule {
    pub enabled: bool,
    pub functions: CasingStyle,
    pub variables: CasingStyle,
    pub types: CasingStyle,
    pub constants: CasingStyle,
    pub ignore_names: Vec<String>,
    pub ignore_patterns: Vec<String>,
}

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct FileSelection {
    pub files: Vec<String>,
    pub exclude: Vec<String>,
}

#[derive(Clone, Debug)]
pub struct LoadFileRequest {
    pub path: String,
    pub base: Config,
    pub language: Option<String>,
}

#[derive(Debug)]
pub enum ConfigError {
    Message(String),
}

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ConfigError::Message(message) => f.write_str(message),
        }
    }
}

impl std::error::Error for ConfigError {}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct ConfigFile {
    version: Option<i32>,
    #[serde(default)]
    rules: RulesFile,
    #[serde(default)]
    languages: std::collections::BTreeMap<String, LanguageFile>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
struct LanguageFile {
    #[serde(default)]
    files: Vec<String>,
    #[serde(default)]
    exclude: Vec<String>,
    #[serde(default)]
    rules: RulesFile,
}

#[derive(Debug, Default, Deserialize)]
#[serde(deny_unknown_fields)]
struct RulesFile {
    #[serde(rename = "max-function-parameters")]
    max_function_parameters: Option<MaxFunctionParametersFile>,
    #[serde(rename = "source-file-header")]
    source_file_header: Option<SourceFileHeaderFile>,
    #[serde(rename = "max-source-file-lines")]
    source_file_lines: Option<SourceFileLinesFile>,
    #[serde(rename = "max-function-body-lines")]
    function_body_lines: Option<FunctionBodyLinesFile>,
    #[serde(rename = "function-docstring")]
    function_docstring: Option<FunctionDocstringFile>,
    indent: Option<IndentFile>,
    casing: Option<CasingFile>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct MaxFunctionParametersFile {
    enabled: Option<bool>,
    max: Option<i32>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct SourceFileHeaderFile {
    required: Option<bool>,
    #[serde(rename = "min-length")]
    min_length: Option<i32>,
    #[serde(rename = "max-length")]
    max_length: Option<i32>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct SourceFileLinesFile {
    max: Option<i32>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct FunctionBodyLinesFile {
    max: Option<i32>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct FunctionDocstringFile {
    policy: Option<FunctionDocstringPolicy>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct IndentFile {
    r#type: Option<IndentType>,
    width: Option<i32>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CasingFile {
    enabled: Option<bool>,
    functions: Option<CasingStyle>,
    variables: Option<CasingStyle>,
    types: Option<CasingStyle>,
    constants: Option<CasingStyle>,
    #[serde(rename = "ignore-names")]
    ignore_names: Option<Vec<String>>,
    #[serde(rename = "ignore-patterns")]
    ignore_patterns: Option<Vec<String>>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            max_function_parameters: MaxFunctionParametersRule {
                enabled: true,
                max: DEFAULT_MAX_FUNCTION_PARAMETERS,
            },
            source_file_header: SourceFileHeaderRule {
                required: false,
                min_length: 0,
                max_length: 0,
            },
            source_file_lines: SourceFileLinesRule { max: 0 },
            function_body_lines: FunctionBodyLinesRule { max: 0 },
            function_docstring: FunctionDocstringRule {
                policy: FunctionDocstringPolicy::Optional,
            },
            indent: IndentRule {
                r#type: IndentType::LanguageDefault,
                width: 0,
            },
            casing: CasingRule {
                enabled: false,
                functions: CasingStyle::LanguageDefault,
                variables: CasingStyle::LanguageDefault,
                types: CasingStyle::LanguageDefault,
                constants: CasingStyle::LanguageDefault,
                ignore_names: Vec::new(),
                ignore_patterns: Vec::new(),
            },
            file_selection: FileSelection::default(),
        }
    }
}

pub fn load_file(request: LoadFileRequest) -> Result<Config, ConfigError> {
    let yaml = fs::read_to_string(&request.path)
        .map_err(|err| ConfigError::Message(format!("read config {:?}: {}", request.path, err)))?;
    let document: ConfigFile = serde_yaml::from_str(&yaml)
        .map_err(|err| ConfigError::Message(format!("parse config {:?}: {}", request.path, err)))?;

    if let Some(version) = document.version {
        if version != 1 {
            return Err(ConfigError::Message(format!(
                "config {:?} uses unsupported version {}",
                request.path, version
            )));
        }
    }

    let mut result = apply_rules(request.base, &document.rules);
    if let Some(language) = request.language.as_deref() {
        if let Some(language_file) = document.languages.get(language) {
            result.file_selection = FileSelection {
                files: language_file.files.clone(),
                exclude: language_file.exclude.clone(),
            };
            result = apply_rules(result, &language_file.rules);
        }
    }

    validate(&result)?;
    Ok(result)
}

fn apply_rules(mut config: Config, rules: &RulesFile) -> Config {
    if let Some(rule) = &rules.max_function_parameters {
        if let Some(enabled) = rule.enabled {
            config.max_function_parameters.enabled = enabled;
        }
        if let Some(max) = rule.max {
            config.max_function_parameters.max = max;
        }
    }

    if let Some(rule) = &rules.source_file_header {
        if let Some(required) = rule.required {
            config.source_file_header.required = required;
        }
        if let Some(min_length) = rule.min_length {
            config.source_file_header.min_length = min_length;
        }
        if let Some(max_length) = rule.max_length {
            config.source_file_header.max_length = max_length;
        }
    }

    if let Some(rule) = &rules.source_file_lines {
        if let Some(max) = rule.max {
            config.source_file_lines.max = max;
        }
    }

    if let Some(rule) = &rules.function_body_lines {
        if let Some(max) = rule.max {
            config.function_body_lines.max = max;
        }
    }

    if let Some(rule) = &rules.function_docstring {
        if let Some(policy) = rule.policy {
            config.function_docstring.policy = policy;
        }
    }

    if let Some(rule) = &rules.indent {
        if let Some(indent_type) = rule.r#type {
            config.indent.r#type = indent_type;
        }
        if let Some(width) = rule.width {
            config.indent.width = width;
        }
    }

    if let Some(rule) = &rules.casing {
        if let Some(enabled) = rule.enabled {
            config.casing.enabled = enabled;
        }
        if let Some(functions) = rule.functions {
            config.casing.functions = functions;
        }
        if let Some(variables) = rule.variables {
            config.casing.variables = variables;
        }
        if let Some(types) = rule.types {
            config.casing.types = types;
        }
        if let Some(constants) = rule.constants {
            config.casing.constants = constants;
        }
        if let Some(ignore_names) = &rule.ignore_names {
            config.casing.ignore_names = ignore_names.clone();
        }
        if let Some(ignore_patterns) = &rule.ignore_patterns {
            config.casing.ignore_patterns = ignore_patterns.clone();
        }
    }

    config
}

pub fn validate(config: &Config) -> Result<(), ConfigError> {
    if config.max_function_parameters.max < 0 {
        return Err(invalid(
            "max-function-parameters.max must be zero or greater",
        ));
    }
    if config.source_file_header.min_length < 0 {
        return Err(invalid(
            "source-file-header.min-length must be zero or greater",
        ));
    }
    if config.source_file_header.max_length < 0 {
        return Err(invalid(
            "source-file-header.max-length must be zero or greater",
        ));
    }
    if config.source_file_header.min_length > 0
        && config.source_file_header.max_length > 0
        && config.source_file_header.max_length < config.source_file_header.min_length
    {
        return Err(invalid(
            "source-file-header.max-length must be greater than or equal to source-file-header.min-length",
        ));
    }
    if config.source_file_lines.max < 0 {
        return Err(invalid("max-source-file-lines.max must be zero or greater"));
    }
    if config.function_body_lines.max < 0 {
        return Err(invalid(
            "max-function-body-lines.max must be zero or greater",
        ));
    }
    if config.indent.width < 0 {
        return Err(invalid("indent.width must be zero or greater"));
    }
    for pattern in &config.casing.ignore_patterns {
        Regex::new(pattern).map_err(|err| {
            ConfigError::Message(format!(
                "casing.ignore-patterns contains invalid regex {:?}: {}",
                pattern, err
            ))
        })?;
    }

    Ok(())
}

fn invalid(message: impl Into<String>) -> ConfigError {
    ConfigError::Message(message.into())
}
