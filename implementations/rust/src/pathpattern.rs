use std::path::Path;

/// Reports whether a slash-separated path matches a vet glob such as "**/*.rs", "dir/**", or "dir/...".
pub fn matches(pattern: &str, file_path: &str) -> bool {
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
        if matches(suffix_pattern, &normalized_path) {
            return true;
        }
        let parts = normalized_path.split('/').collect::<Vec<_>>();
        for index in 1..parts.len() {
            if matches(suffix_pattern, &parts[index..].join("/")) {
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
