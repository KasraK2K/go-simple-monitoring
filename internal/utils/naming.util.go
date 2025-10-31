package utils

import "strings"

// IsEmptyOrWhitespace checks if a string is empty or contains only whitespace
func IsEmptyOrWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}

// SanitizeFilesystemName converts a string into a filesystem-friendly slug.
func SanitizeFilesystemName(input string) string {
	if IsEmptyOrWhitespace(input) {
		return ""
	}
	name := strings.TrimSpace(input)

	name = strings.ToLower(name)

	var builder strings.Builder
	builder.Grow(len(name))

	lastWasDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastWasDash = false
		case r == '-' || r == '_':
			builder.WriteRune(r)
			lastWasDash = false
		default:
			if !lastWasDash {
				builder.WriteRune('-')
				lastWasDash = true
			}
		}
	}

	sanitized := strings.Trim(builder.String(), "-_")
	if sanitized == "" {
		return "server"
	}

	return sanitized
}

// SanitizeTableName converts a string into a SQLite-friendly table identifier.
func SanitizeTableName(input string) string {
	if IsEmptyOrWhitespace(input) {
		return ""
	}
	name := strings.TrimSpace(input)

	name = strings.ToLower(name)

	var builder strings.Builder
	builder.Grow(len(name) + 2)

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r == '_':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	sanitized := strings.Trim(builder.String(), "_")
	if sanitized == "" {
		sanitized = "server_log"
	}

	if sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "t_" + sanitized
	}

	return sanitized
}
