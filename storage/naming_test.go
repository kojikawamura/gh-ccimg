package storage

import (
	"testing"
)

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name      string
		index     int
		extension string
		expected  string
	}{
		{"first PNG", 0, ".png", "img-01.png"},
		{"second JPEG", 1, ".jpg", "img-02.jpg"},
		{"tenth GIF", 9, ".gif", "img-10.gif"},
		{"extension without dot", 0, "png", "img-01.png"},
		{"empty extension", 0, "", "img-01.bin"},
		{"large index", 99, ".webp", "img-100.webp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFilename(tt.index, tt.extension)
			if result != tt.expected {
				t.Errorf("GenerateFilename(%d, %q) = %q, want %q", tt.index, tt.extension, result, tt.expected)
			}
		})
	}
}

func TestExtractExtensionFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"PNG URL", "https://example.com/image.png", ".png"},
		{"JPEG URL", "https://example.com/photo.jpg", ".jpg"},
		{"GIF URL", "https://example.com/animation.gif", ".gif"},
		{"WebP URL", "https://example.com/modern.webp", ".webp"},
		{"SVG URL", "https://example.com/vector.svg", ".svg"},
		{"uppercase extension", "https://example.com/IMAGE.PNG", ".png"},
		{"with query params", "https://example.com/image.png?size=large", ".png"},
		{"with fragment", "https://example.com/image.jpg#section", ".jpg"},
		{"with both params and fragment", "https://example.com/image.gif?v=1#top", ".gif"},
		{"no extension", "https://example.com/image", ""},
		{"invalid extension", "https://example.com/document.pdf", ""},
		{"empty URL", "", ""},
		{"complex path", "https://cdn.example.com/assets/images/photo.jpeg", ".jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractExtensionFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractExtensionFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestDetermineExtension(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		url         string
		expected    string
	}{
		{
			"content type priority",
			"image/png",
			"https://example.com/image.jpg",
			".png", // Should prefer content type over URL
		},
		{
			"URL fallback",
			"",
			"https://example.com/image.jpg",
			".jpg", // Should use URL when no content type
		},
		{
			"content type with params",
			"image/jpeg; charset=utf-8",
			"https://example.com/image.png",
			".jpg", // Should prefer content type over URL
		},
		{
			"invalid content type, valid URL",
			"text/plain",
			"https://example.com/image.png",
			".png", // Should fall back to URL
		},
		{
			"both empty",
			"",
			"",
			".bin", // Should use default
		},
		{
			"invalid both",
			"application/pdf",
			"https://example.com/document.pdf",
			".bin", // Should use default when both invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineExtension(tt.contentType, tt.url)
			if result != tt.expected {
				t.Errorf("DetermineExtension(%q, %q) = %q, want %q", tt.contentType, tt.url, result, tt.expected)
			}
		})
	}
}

func TestGetExtensionFromContentType(t *testing.T) {
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
		{"uppercase", "IMAGE/PNG", ".png"},
		{"with charset", "image/jpeg; charset=utf-8", ".jpg"},
		{"with boundary", "image/png; boundary=something", ".png"},
		{"unknown type", "application/pdf", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("getExtensionFromContentType(%q) = %q, want %q", tt.contentType, result, tt.expected)
			}
		})
	}
}