package test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojikawamura/gh-ccimg/markdown"
)

// TestTestDataStructure verifies the testdata directory structure and files
func TestTestDataStructure(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Check main directories exist
	dirs := []string{
		"markdown",
		"images", 
		"responses",
		"fixtures/golden",
		"fixtures/temp",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(testdataDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Required testdata directory missing: %s", dirPath)
		}
	}

	// Check required markdown files exist
	markdownFiles := []string{
		"simple.md",
		"complex.md", 
		"malformed.md",
		"empty.md",
	}

	for _, file := range markdownFiles {
		filePath := filepath.Join(testdataDir, "markdown", file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Required markdown file missing: %s", filePath)
		}
	}

	// Check required response files exist
	responseFiles := []string{
		"issue.json",
		"comments.json",
		"error.json",
	}

	for _, file := range responseFiles {
		filePath := filepath.Join(testdataDir, "responses", file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Required response file missing: %s", filePath)
		}
	}

	// Check required image files exist
	imageFiles := []string{
		"test-image.png",
		"test-image.jpg",
		"invalid.txt",
	}

	for _, file := range imageFiles {
		filePath := filepath.Join(testdataDir, "images", file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Required image file missing: %s", filePath)
		}
	}

	// Check required golden files exist
	goldenFiles := []string{
		"simple_urls.txt",
		"complex_urls.txt",
	}

	for _, file := range goldenFiles {
		filePath := filepath.Join(testdataDir, "fixtures", "golden", file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Required golden file missing: %s", filePath)
		}
	}
}

// TestMarkdownFilesWithGoldenData tests markdown parsing against expected results
func TestMarkdownFilesWithGoldenData(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		markdownFile string
		goldenFile   string
	}{
		{"simple.md", "simple_urls.txt"},
		{"complex.md", "complex_urls.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.markdownFile, func(t *testing.T) {
			// Read markdown file
			markdownPath := filepath.Join(testdataDir, "markdown", tt.markdownFile)
			content, err := os.ReadFile(markdownPath)
			if err != nil {
				t.Fatalf("Failed to read markdown file %s: %v", markdownPath, err)
			}

			// Extract URLs
			extractedURLs := markdown.ExtractImageURLs(string(content))

			// Read expected URLs from golden file
			goldenPath := filepath.Join(testdataDir, "fixtures", "golden", tt.goldenFile)
			file, err := os.Open(goldenPath)
			if err != nil {
				t.Fatalf("Failed to open golden file %s: %v", goldenPath, err)
			}
			defer file.Close()

			var expectedURLs []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					expectedURLs = append(expectedURLs, line)
				}
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Error reading golden file %s: %v", goldenPath, err)
			}

			// Compare extracted URLs with expected URLs
			if len(extractedURLs) != len(expectedURLs) {
				t.Errorf("URL count mismatch for %s: got %d, expected %d", 
					tt.markdownFile, len(extractedURLs), len(expectedURLs))
				t.Logf("Extracted URLs: %v", extractedURLs)
				t.Logf("Expected URLs: %v", expectedURLs)
				return
			}

			// Check each URL
			for i, expected := range expectedURLs {
				if i >= len(extractedURLs) {
					t.Errorf("Missing URL at index %d: expected %s", i, expected)
					continue
				}
				if extractedURLs[i] != expected {
					t.Errorf("URL mismatch at index %d: got %s, expected %s", 
						i, extractedURLs[i], expected)
				}
			}
		})
	}
}

// TestEmptyMarkdownFile tests handling of empty markdown
func TestEmptyMarkdownFile(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	emptyPath := filepath.Join(testdataDir, "markdown", "empty.md")

	content, err := os.ReadFile(emptyPath)
	if err != nil {
		t.Fatalf("Failed to read empty markdown file: %v", err)
	}

	urls := markdown.ExtractImageURLs(string(content))
	if len(urls) != 0 {
		t.Errorf("Expected 0 URLs from empty file, got %d: %v", len(urls), urls)
	}
}

// TestMalformedMarkdownFile tests handling of malformed markdown
func TestMalformedMarkdownFile(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	malformedPath := filepath.Join(testdataDir, "markdown", "malformed.md")

	content, err := os.ReadFile(malformedPath)
	if err != nil {
		t.Fatalf("Failed to read malformed markdown file: %v", err)
	}

	// Should not panic and should extract some valid URLs
	urls := markdown.ExtractImageURLs(string(content))
	t.Logf("Extracted %d URLs from malformed markdown", len(urls))

	// Verify that valid URLs are still extracted despite malformed content
	validURLsFound := 0
	for _, url := range urls {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			validURLsFound++
		}
	}

	if validURLsFound == 0 {
		t.Errorf("Expected to find some valid URLs even in malformed markdown")
	}
}

// TestImageFiles tests that test image files are valid
func TestImageFiles(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	imageTests := []struct {
		filename    string
		expectedMin int // Minimum expected file size
		shouldExist bool
	}{
		{"test-image.png", 50, true},  // PNG should be at least 50 bytes
		{"test-image.jpg", 50, true},  // JPEG should be at least 50 bytes
		{"invalid.txt", 10, true},     // Text file should exist
	}

	for _, tt := range imageTests {
		t.Run(tt.filename, func(t *testing.T) {
			filePath := filepath.Join(testdataDir, "images", tt.filename)
			
			info, err := os.Stat(filePath)
			if tt.shouldExist {
				if err != nil {
					t.Fatalf("Expected file %s to exist: %v", filePath, err)
				}
				if info.Size() < int64(tt.expectedMin) {
					t.Errorf("File %s too small: %d bytes (expected at least %d)", 
						tt.filename, info.Size(), tt.expectedMin)
				}
			} else {
				if err == nil {
					t.Errorf("Expected file %s to not exist", filePath)
				}
			}
		})
	}
}

// TestResponseFiles tests that JSON response files are valid
func TestResponseFiles(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	responseFiles := []string{
		"issue.json",
		"comments.json", 
		"error.json",
	}

	for _, filename := range responseFiles {
		t.Run(filename, func(t *testing.T) {
			filePath := filepath.Join(testdataDir, "responses", filename)
			
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read response file %s: %v", filePath, err)
			}

			// Verify it's valid JSON by attempting to parse
			if len(content) == 0 {
				t.Errorf("Response file %s is empty", filename)
			}

			// Basic JSON validation - should start with { or [
			trimmed := strings.TrimSpace(string(content))
			if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
				t.Errorf("Response file %s doesn't appear to be valid JSON", filename)
			}
		})
	}
}

// TestGoldenFiles tests that golden files are properly formatted
func TestGoldenFiles(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	goldenFiles := []string{
		"simple_urls.txt",
		"complex_urls.txt",
	}

	for _, filename := range goldenFiles {
		t.Run(filename, func(t *testing.T) {
			filePath := filepath.Join(testdataDir, "fixtures", "golden", filename)
			
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open golden file %s: %v", filePath, err)
			}
			defer file.Close()

			var urls []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					urls = append(urls, line)
				}
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Error reading golden file %s: %v", filePath, err)
			}

			if len(urls) == 0 {
				t.Errorf("Golden file %s contains no URLs", filename)
			}

			// Verify URLs are well-formed
			for i, url := range urls {
				if !strings.HasPrefix(url, "http://") && 
				   !strings.HasPrefix(url, "https://") && 
				   !strings.HasPrefix(url, "//") {
					t.Errorf("Golden file %s line %d: invalid URL format: %s", 
						filename, i+1, url)
				}
			}
		})
	}
}

// TestTempDirectory tests that temp directory is writable
func TestTempDirectory(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	tempDir := filepath.Join(testdataDir, "fixtures", "temp")

	// Try to create a temporary file
	tempFile := filepath.Join(tempDir, "test-write.tmp")
	
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Cannot write to temp directory %s: %v", tempDir, err)
	}

	// Clean up
	os.Remove(tempFile)
}

// BenchmarkTestDataLoad benchmarks loading test data
func BenchmarkTestDataLoad(b *testing.B) {
	testdataDir := filepath.Join("..", "testdata")

	b.Run("load_simple_markdown", func(b *testing.B) {
		filePath := filepath.Join(testdataDir, "markdown", "simple.md")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := os.ReadFile(filePath)
			if err != nil {
				b.Fatalf("Failed to read file: %v", err)
			}
			_ = markdown.ExtractImageURLs(string(content))
		}
	})

	b.Run("load_complex_markdown", func(b *testing.B) {
		filePath := filepath.Join(testdataDir, "markdown", "complex.md")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := os.ReadFile(filePath)
			if err != nil {
				b.Fatalf("Failed to read file: %v", err)
			}
			_ = markdown.ExtractImageURLs(string(content))
		}
	})
}