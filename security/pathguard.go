package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidatePath ensures that target paths remain within the base directory
// and prevents directory traversal attacks
func ValidatePath(base, target string) error {
	if base == "" {
		return fmt.Errorf("base path cannot be empty")
	}
	if target == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	// Clean both paths to resolve any . and .. elements
	cleanBase := filepath.Clean(base)
	cleanTarget := filepath.Clean(target)

	// Convert to absolute paths
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute base path: %w", err)
	}

	absTarget, err := filepath.Abs(cleanTarget)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute target path: %w", err)
	}

	// Ensure base path ends with separator for proper prefix checking
	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}

	// Check if target is within base directory
	if !strings.HasPrefix(absTarget+string(filepath.Separator), absBase) {
		return fmt.Errorf("path traversal detected: target path %q is outside base directory %q", target, base)
	}

	return nil
}

// ValidateOutputPath validates an output path for writing files
func ValidateOutputPath(outputDir, filename string) error {
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check for suspicious characters in filename
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename contains directory traversal sequence: %s", filename)
	}

	// Check for absolute paths in filename
	if filepath.IsAbs(filename) {
		return fmt.Errorf("filename cannot be an absolute path: %s", filename)
	}

	// Construct target path
	targetPath := filepath.Join(outputDir, filename)

	// Validate using the main validation function
	return ValidatePath(outputDir, targetPath)
}

// SanitizeFilename removes or replaces dangerous characters from a filename
func SanitizeFilename(filename string) string {
	if filename == "" {
		return "unnamed"
	}

	// Replace dangerous characters with underscores
	dangerous := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	result := filename

	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove leading/trailing dots and spaces
	result = strings.Trim(result, ". ")

	// Ensure it's not empty after sanitization
	if result == "" || result == strings.Repeat("_", len(result)) {
		result = "unnamed"
	}

	// Limit length to reasonable maximum
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// IsPathSafe performs additional safety checks on a path
func IsPathSafe(path string) bool {
	if path == "" {
		return false
	}

	// Clean the path
	clean := filepath.Clean(path)

	// Check for suspicious patterns
	suspicious := []string{
		"..",
		"~",
		"$",
	}

	for _, pattern := range suspicious {
		if strings.Contains(clean, pattern) {
			return false
		}
	}

	// Check for absolute paths (might be suspicious in some contexts)
	if filepath.IsAbs(clean) {
		return false
	}

	return true
}