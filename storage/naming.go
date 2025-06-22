package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateFilename generates a sequential filename with proper extension
// Format: img-01.png, img-02.jpg, etc.
func GenerateFilename(index int, extension string) string {
	// Ensure extension starts with dot
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	
	// Use default extension if none provided
	if extension == "" {
		extension = ".bin"
	}
	
	return fmt.Sprintf("img-%02d%s", index+1, extension)
}

// ExtractExtensionFromURL attempts to extract file extension from URL
func ExtractExtensionFromURL(url string) string {
	if url == "" {
		return ""
	}
	
	// Remove query parameters and fragments
	if idx := strings.Index(url, "?"); idx > 0 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "#"); idx > 0 {
		url = url[:idx]
	}
	
	// Get the extension
	ext := filepath.Ext(url)
	if ext == "" {
		return ""
	}
	
	// Convert to lowercase and validate it's a reasonable image extension
	ext = strings.ToLower(ext)
	validExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".ico"}
	
	for _, valid := range validExtensions {
		if ext == valid {
			return ext
		}
	}
	
	// Return empty if not a recognized image extension
	return ""
}

// DetermineExtension determines the best extension to use
// Priority: contentType > URL > default
func DetermineExtension(contentType, url string) string {
	// First try to get extension from content type
	if contentType != "" {
		if ext := getExtensionFromContentType(contentType); ext != "" {
			return ext
		}
	}
	
	// Then try to extract from URL
	if url != "" {
		if ext := ExtractExtensionFromURL(url); ext != "" {
			return ext
		}
	}
	
	// Default fallback
	return ".bin"
}

// getExtensionFromContentType converts content type to file extension
func getExtensionFromContentType(contentType string) string {
	lower := strings.ToLower(contentType)
	
	// Strip any charset or other parameters
	if idx := strings.Index(lower, ";"); idx > 0 {
		lower = lower[:idx]
	}
	lower = strings.TrimSpace(lower)

	switch lower {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ""
	}
}