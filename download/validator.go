package download

import (
	"fmt"
	"strings"
)

// ValidateContentType checks if the content type is a valid image type
func ValidateContentType(contentType string) error {
	if contentType == "" {
		return fmt.Errorf("content-type header is missing")
	}

	// Convert to lowercase for comparison
	lower := strings.ToLower(contentType)
	
	// Strip any charset or other parameters
	if idx := strings.Index(lower, ";"); idx > 0 {
		lower = lower[:idx]
	}
	lower = strings.TrimSpace(lower)

	// List of accepted image MIME types
	validTypes := []string{
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
		"image/webp",
		"image/svg+xml",
		"image/bmp",
		"image/tiff",
		"image/x-icon",
		"image/vnd.microsoft.icon", // .ico files
	}

	for _, validType := range validTypes {
		if lower == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid content type for image: %s (expected image/*)", contentType)
}

// GetFileExtensionFromContentType returns the appropriate file extension for a content type
func GetFileExtensionFromContentType(contentType string) string {
	if contentType == "" {
		return ".bin"
	}

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
		return ".bin"
	}
}