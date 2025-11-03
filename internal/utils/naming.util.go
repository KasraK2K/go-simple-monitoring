package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ValidateLogPath validates and sanitizes a log directory path for security
func ValidateLogPath(logPath string) (string, error) {
	if IsEmptyOrWhitespace(logPath) {
		return "", fmt.Errorf("log path cannot be empty")
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(logPath)
	
	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Ensure path doesn't contain dangerous patterns
	if strings.Contains(absPath, "..") {
		return "", fmt.Errorf("path contains directory traversal pattern")
	}

	// Get current working directory to define safe boundaries
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Define allowed base directories (relative to working directory)
	allowedBases := []string{
		filepath.Join(wd, "logs"),
		filepath.Join(wd, "data"),
		filepath.Join(wd, "storage"),
		"/tmp/go-monitoring",
		"/var/log/go-monitoring",
	}

	// Check if the path is within allowed boundaries
	pathAllowed := false
	for _, base := range allowedBases {
		baseAbs, err := filepath.Abs(base)
		if err != nil {
			continue
		}
		
		// Check if absPath is within or equal to the allowed base
		rel, err := filepath.Rel(baseAbs, absPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		return "", fmt.Errorf("path %s is not within allowed directories", absPath)
	}

	return absPath, nil
}

// ValidateServerDirName validates and sanitizes server directory names
func ValidateServerDirName(serverName string) (string, error) {
	if IsEmptyOrWhitespace(serverName) {
		return "", fmt.Errorf("server name cannot be empty")
	}

	// Sanitize the name
	sanitized := SanitizeFilesystemName(serverName)
	if sanitized == "" {
		return "", fmt.Errorf("server name resulted in empty string after sanitization")
	}

	// Additional security checks
	if len(sanitized) > 100 {
		return "", fmt.Errorf("server name too long (max 100 characters)")
	}

	// Check for reserved names
	reservedNames := []string{"con", "prn", "aux", "nul", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9"}
	for _, reserved := range reservedNames {
		if strings.EqualFold(sanitized, reserved) {
			return "server_" + sanitized, nil
		}
	}

	return sanitized, nil
}

// CreateSecureDirectory creates a directory with proper validation and permissions
func CreateSecureDirectory(dirPath string) error {
	// Validate the directory path
	validatedPath, err := ValidateLogPath(dirPath)
	if err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// Check if directory already exists
	if info, err := os.Stat(validatedPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", validatedPath)
		}
		// Directory already exists, check permissions
		if info.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("directory is not writable: %s", validatedPath)
		}
		return nil
	}

	// Create directory with restrictive permissions
	if err := os.MkdirAll(validatedPath, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", validatedPath, err)
	}

	LogInfo("created secure directory: %s", validatedPath)
	return nil
}
