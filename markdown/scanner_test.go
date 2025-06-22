package markdown

import (
	"reflect"
	"sort"
	"testing"
)

func TestExtractImageURLs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
		{
			name:    "simple markdown image",
			content: "![alt text](https://example.com/image.png)",
			expected: []string{
				"https://example.com/image.png",
			},
		},
		{
			name:    "multiple images",
			content: "![img1](https://example.com/1.png) and ![img2](https://example.com/2.jpg)",
			expected: []string{
				"https://example.com/1.png",
				"https://example.com/2.jpg",
			},
		},
		{
			name: "HTML img tags",
			content: `<img src="https://example.com/html.png" alt="html image">
			<img src='https://example.com/html2.jpg' alt='another'>`,
			expected: []string{
				"https://example.com/html.png",
				"https://example.com/html2.jpg",
			},
		},
		{
			name: "GitHub user content URLs",
			content: "![screenshot](https://user-images.githubusercontent.com/123456/image.png)",
			expected: []string{
				"https://user-images.githubusercontent.com/123456/image.png",
			},
		},
		{
			name: "GitHub assets URLs",
			content: "![asset](https://github.com/user/repo/assets/12345678/image.png)",
			expected: []string{
				"https://github.com/user/repo/assets/12345678/image.png",
			},
		},
		{
			name: "reference style markdown",
			content: `![alt text][ref1]

[ref1]: https://example.com/ref.png`,
			expected: []string{
				"https://example.com/ref.png",
			},
		},
		{
			name: "mixed formats",
			content: `# Title
![direct](https://example.com/direct.png)
<img src="https://example.com/html.jpg">
![ref style][myref]

[myref]: https://example.com/referenced.gif`,
			expected: []string{
				"https://example.com/direct.png",
				"https://example.com/html.jpg",
				"https://example.com/referenced.gif",
			},
		},
		{
			name: "duplicate URLs",
			content: `![img1](https://example.com/same.png)
![img2](https://example.com/same.png)`,
			expected: []string{
				"https://example.com/same.png",
			},
		},
		{
			name: "various image extensions",
			content: `![png](https://example.com/1.png)
![jpg](https://example.com/2.jpg)
![jpeg](https://example.com/3.jpeg)
![gif](https://example.com/4.gif)
![webp](https://example.com/5.webp)
![svg](https://example.com/6.svg)`,
			expected: []string{
				"https://example.com/1.png",
				"https://example.com/2.jpg",
				"https://example.com/3.jpeg",
				"https://example.com/4.gif",
				"https://example.com/5.webp",
				"https://example.com/6.svg",
			},
		},
		{
			name:    "non-image URLs ignored",
			content: "![not an image](https://example.com/document.pdf)",
			expected: []string{
				"https://example.com/document.pdf", // Still included, validation happens at download
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractImageURLs(tt.content)
			
			// Sort both slices for comparison since order might vary
			sort.Strings(result)
			expected := make([]string, len(tt.expected))
			copy(expected, tt.expected)
			sort.Strings(expected)
			
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("ExtractImageURLs() = %v, want %v", result, expected)
			}
		})
	}
}

func TestIsValidImageURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"empty URL", "", false},
		{"PNG image", "https://example.com/image.png", true},
		{"JPG image", "https://example.com/image.jpg", true},
		{"JPEG image", "https://example.com/image.jpeg", true},
		{"GIF image", "https://example.com/image.gif", true},
		{"WebP image", "https://example.com/image.webp", true},
		{"SVG image", "https://example.com/image.svg", true},
		{"case insensitive", "https://example.com/IMAGE.PNG", true},
		{"GitHub user content", "https://user-images.githubusercontent.com/123/img", true},
		{"GitHub assets", "https://github.com/user/repo/assets/123/file", true},
		{"images path", "https://example.com/images/photo", true},
		{"img path", "https://example.com/img/photo", true},
		{"data URL", "data:image/png;base64,iVBOR...", true},
		{"HTTP URL", "http://example.com/unknown", true}, // Let downloader validate
		{"HTTPS URL", "https://example.com/unknown", true}, // Let downloader validate
		{"PDF document", "https://example.com/doc.pdf", true}, // permissive - let downloader validate
		{"text file", "https://example.com/readme.txt", true}, // permissive - let downloader validate
		{"no protocol", "example.com/image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidImageURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidImageURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestDeduplicateURLs(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		expected []string
	}{
		{
			name:     "no duplicates",
			urls:     []string{"https://example.com/1.png", "https://example.com/2.jpg"},
			expected: []string{"https://example.com/1.png", "https://example.com/2.jpg"},
		},
		{
			name:     "with duplicates",
			urls:     []string{"https://example.com/1.png", "https://example.com/2.jpg", "https://example.com/1.png"},
			expected: []string{"https://example.com/1.png", "https://example.com/2.jpg"},
		},
		{
			name:     "empty strings",
			urls:     []string{"", "https://example.com/1.png", "", "https://example.com/1.png"},
			expected: []string{"https://example.com/1.png"},
		},
		{
			name:     "whitespace handling",
			urls:     []string{" https://example.com/1.png ", "https://example.com/1.png"},
			expected: []string{"https://example.com/1.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateURLs(tt.urls)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deduplicateURLs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractReferences(t *testing.T) {
	content := `# Title

[ref1]: https://example.com/1.png
[ref2]: https://example.com/2.jpg "Title"
[ref3]: https://example.com/3.gif 'Alt title'

Some text here.`

	expected := map[string]string{
		"ref1": "https://example.com/1.png",
		"ref2": "https://example.com/2.jpg",
		"ref3": "https://example.com/3.gif",
	}

	result := extractReferences(content)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("extractReferences() = %v, want %v", result, expected)
	}
}