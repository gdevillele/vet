package pathpattern

import (
	"path"
	"path/filepath"
	"strings"
)

// Match reports whether a slash-separated path matches a vet glob such as "**/*.py", "dir/**", or "dir/...".
func Match(pattern string, filePath string) bool {
	normalizedPattern := normalizePattern(pattern)
	normalizedPath := normalizePattern(filePath)

	if normalizedPattern == "" {
		return false
	}

	if normalizedPattern == "..." {
		return true
	}
	if strings.HasSuffix(normalizedPattern, "/...") {
		prefix := strings.TrimSuffix(normalizedPattern, "/...")
		return normalizedPath == prefix || strings.HasPrefix(normalizedPath, prefix+"/")
	}
	if strings.HasSuffix(normalizedPattern, "/**") {
		prefix := strings.TrimSuffix(normalizedPattern, "/**")
		return normalizedPath == prefix || strings.HasPrefix(normalizedPath, prefix+"/")
	}
	if strings.HasPrefix(normalizedPattern, "**/") {
		suffixPattern := strings.TrimPrefix(normalizedPattern, "**/")
		if Match(suffixPattern, normalizedPath) {
			return true
		}
		parts := strings.Split(normalizedPath, "/")
		for index := 1; index < len(parts); index++ {
			if Match(suffixPattern, strings.Join(parts[index:], "/")) {
				return true
			}
		}
		return false
	}

	if matches, err := path.Match(normalizedPattern, normalizedPath); err == nil && matches {
		return true
	}
	if !strings.Contains(normalizedPattern, "/") {
		if matches, err := path.Match(normalizedPattern, path.Base(normalizedPath)); err == nil && matches {
			return true
		}
	}

	return false
}

func normalizePattern(value string) string {
	result := filepath.ToSlash(value)
	for strings.HasPrefix(result, "./") {
		result = strings.TrimPrefix(result, "./")
	}
	return strings.TrimSuffix(result, "/")
}
