package download

import (
	"testing"
)

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		// Valid image types
		{"PNG", "image/png", false},
		{"JPEG", "image/jpeg", false},
		{"JPG", "image/jpg", false},
		{"GIF", "image/gif", false},
		{"WebP", "image/webp", false},
		{"SVG", "image/svg+xml", false},
		{"BMP", "image/bmp", false},
		{"TIFF", "image/tiff", false},
		{"ICO", "image/x-icon", false},
		{"ICO alt", "image/vnd.microsoft.icon", false},

		// Case insensitive
		{"PNG uppercase", "IMAGE/PNG", false},
		{"JPEG mixed case", "Image/Jpeg", false},

		// With charset parameters
		{"PNG with charset", "image/png; charset=utf-8", false},
		{"JPEG with boundary", "image/jpeg; boundary=something", false},

		// Invalid types
		{"empty", "", true},
		{"text", "text/plain", true},
		{"HTML", "text/html", true},
		{"JSON", "application/json", true},
		{"PDF", "application/pdf", true},
		{"video", "video/mp4", true},
		{"audio", "audio/mp3", true},

		// Edge cases
		{"whitespace", "  image/png  ", false},
		{"semicolon only", "image/png;", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentType(tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentType(%q) error = %v, wantErr %v", tt.contentType, err, tt.wantErr)
			}
		})
	}
}

func TestGetFileExtensionFromContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{"PNG", "image/png", ".png"},
		{"JPEG", "image/jpeg", ".jpg"},
		{"JPG", "image/jpg", ".jpg"},
		{"GIF", "image/gif", ".gif"},
		{"WebP", "image/webp", ".webp"},
		{"SVG", "image/svg+xml", ".svg"},
		{"BMP", "image/bmp", ".bmp"},
		{"TIFF", "image/tiff", ".tiff"},
		{"ICO", "image/x-icon", ".ico"},
		{"ICO alt", "image/vnd.microsoft.icon", ".ico"},

		// Case insensitive
		{"PNG uppercase", "IMAGE/PNG", ".png"},

		// With parameters
		{"PNG with charset", "image/png; charset=utf-8", ".png"},

		// Unknown types
		{"empty", "", ".bin"},
		{"unknown", "application/octet-stream", ".bin"},
		{"text", "text/plain", ".bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFileExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("GetFileExtensionFromContentType(%q) = %q, want %q", tt.contentType, result, tt.expected)
			}
		})
	}
}