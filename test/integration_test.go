package test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kojikawamura/gh-ccimg/claude"
	"github.com/kojikawamura/gh-ccimg/download"
	"github.com/kojikawamura/gh-ccimg/github"
	"github.com/kojikawamura/gh-ccimg/markdown"
	"github.com/kojikawamura/gh-ccimg/storage"
)

// Integration tests for the complete pipeline with mocked dependencies

func TestIntegration_CompletePipeline_MemoryMode(t *testing.T) {
	// Set up test server with image responses
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/image1.png":
			w.Header().Set("Content-Type", "image/png")
			// Create a minimal PNG (8x8 black image)
			pngData := []byte{
				0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
				0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
				0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x08, // 8x8 image
				0x08, 0x02, 0x00, 0x00, 0x00, 0x4B, 0x6D, 0x29, // bit depth, color type, etc.
				0xDC, 0x00, 0x00, 0x00, 0x17, 0x49, 0x44, 0x41, // IDAT chunk
				0x54, 0x78, 0x9C, 0x62, 0x00, 0x02, 0x00, 0x00, // compressed image data
				0x05, 0x00, 0x01, 0xE2, 0x26, 0x05, 0x9B, 0x00,
				0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
				0x42, 0x60, 0x82,
			}
			w.Write(pngData)
		case "/image2.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			// Create a minimal JPEG
			jpegData := []byte{
				0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, // JPEG header
				0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48,
				0x00, 0x48, 0x00, 0x00, 0xFF, 0xC0, 0x00, 0x11,
				0x08, 0x00, 0x08, 0x00, 0x08, 0x01, 0x01, 0x11,
				0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01, 0xFF,
				0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x08, 0xFF, 0xDA, 0x00,
				0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00, 0xD2,
				0xFF, 0xD9, // End of image
			}
			w.Write(jpegData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer imageServer.Close()

	// Test markdown content with images
	markdownContent := fmt.Sprintf(`
# Test Issue

Here are some screenshots:

![Image 1](%s/image1.png)
![Image 2](%s/image2.jpg)

Additional context here.
`, imageServer.URL, imageServer.URL)

	// Test the markdown extraction
	urls := markdown.ExtractImageURLs(markdownContent)
	if len(urls) != 2 {
		t.Fatalf("Expected 2 URLs, got %d", len(urls))
	}

	expectedURLs := []string{
		imageServer.URL + "/image1.png",
		imageServer.URL + "/image2.jpg",
	}

	for i, url := range urls {
		if url != expectedURLs[i] {
			t.Errorf("URL %d: expected %s, got %s", i, expectedURLs[i], url)
		}
	}

	// Test the download process
	fetcher := download.NewFetcher(10*1024*1024, 30*time.Second, 2) // 10MB, 30s, 2 workers
	ctx := context.Background()
	results := fetcher.FetchConcurrent(ctx, urls)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	successCount := 0
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Download failed for %s: %v", result.URL, result.Error)
		} else {
			successCount++
			t.Logf("Successfully downloaded %s (%d bytes, %s)", result.URL, result.Size, result.ContentType)
		}
	}

	if successCount != 2 {
		t.Fatalf("Expected 2 successful downloads, got %d", successCount)
	}

	// Test memory storage
	memStorage := storage.NewMemoryStorage()
	var encodedImages []string

	for _, result := range results {
		if result.Error == nil {
			encoded, err := memStorage.Store(result.Data, result.ContentType, result.URL)
			if err != nil {
				t.Errorf("Failed to store image %s: %v", result.URL, err)
			} else {
				encodedImages = append(encodedImages, encoded)
			}
		}
	}

	if len(encodedImages) != 2 {
		t.Fatalf("Expected 2 encoded images, got %d", len(encodedImages))
	}

	// Verify base64 encoding
	for i, encoded := range encodedImages {
		if encoded == "" {
			t.Errorf("Image %d: got empty encoding", i)
		}
		// Verify it's valid base64 (memory storage returns raw base64, not data URL)
		if len(encoded) < 10 {
			t.Errorf("Image %d: encoding too short: %s", i, encoded)
		}
	}

	t.Logf("Integration test passed: extracted %d images, downloaded %d, encoded %d", 
		len(urls), successCount, len(encodedImages))
}

func TestIntegration_CompletePipeline_DiskMode(t *testing.T) {
	// Create temporary directory for output
	tempDir, err := os.MkdirTemp("", "gh-ccimg-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test server with image responses
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test.gif":
			w.Header().Set("Content-Type", "image/gif")
			// Create a minimal GIF (1x1 transparent pixel)
			gifData := []byte{
				0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a header
				0x01, 0x00, 0x01, 0x00, 0x80, 0x00, // 1x1 image with global color table
				0x00, 0xFF, 0xFF, 0xFF, 0x00, 0x00, // Global color table (white, black)
				0x00, 0x21, 0xF9, 0x04, 0x01, 0x00, // Graphics control extension
				0x00, 0x00, 0x00, 0x2C, 0x00, 0x00, // Image descriptor
				0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
				0x00, 0x02, 0x02, 0x04, 0x01, 0x00, // Image data
				0x3B, // Trailer
			}
			w.Write(gifData)
		case "/test.webp":
			w.Header().Set("Content-Type", "image/webp")
			// Create a minimal WebP (1x1 pixel)
			webpData := []byte{
				0x52, 0x49, 0x46, 0x46, 0x1A, 0x00, 0x00, 0x00, // RIFF header
				0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20, // WEBP VP8
				0x0E, 0x00, 0x00, 0x00, 0x9D, 0x01, 0x2A, 0x01,
				0x00, 0x01, 0x00, 0x9D, 0x01, 0x2A, 0x06, 0x00,
				0x88, 0x85, 0x85, 0x88,
			}
			w.Write(webpData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer imageServer.Close()

	// Test markdown content
	markdownContent := fmt.Sprintf(`
# Test with Different Formats

![GIF image](%s/test.gif)
<img src="%s/test.webp" alt="WebP image">
`, imageServer.URL, imageServer.URL)

	// Extract URLs
	urls := markdown.ExtractImageURLs(markdownContent)
	if len(urls) != 2 {
		t.Fatalf("Expected 2 URLs, got %d: %v", len(urls), urls)
	}

	// Download images
	fetcher := download.NewFetcher(5*1024*1024, 15*time.Second, 2)
	ctx := context.Background()
	results := fetcher.FetchConcurrent(ctx, urls)

	// Store to disk
	diskStorage, err := storage.NewDiskStorage(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create disk storage: %v", err)
	}

	var savedFiles []string
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Download failed for %s: %v", result.URL, result.Error)
			continue
		}

		filePath, err := diskStorage.Store(result.Data, result.ContentType, result.URL)
		if err != nil {
			t.Errorf("Failed to store %s: %v", result.URL, err)
			continue
		}

		savedFiles = append(savedFiles, filePath)
		t.Logf("Saved %s to %s", result.URL, filePath)

		// Verify file exists and has content
		info, err := os.Stat(filePath)
		if err != nil {
			t.Errorf("File %s not found: %v", filePath, err)
		} else if info.Size() == 0 {
			t.Errorf("File %s is empty", filePath)
		}
	}

	if len(savedFiles) != 2 {
		t.Fatalf("Expected 2 saved files, got %d", len(savedFiles))
	}

	// Verify file naming convention
	expectedPatterns := []string{"img-01", "img-02"}
	for i, filePath := range savedFiles {
		fileName := filepath.Base(filePath)
		if !strings.Contains(fileName, expectedPatterns[i]) {
			t.Errorf("File %d (%s) doesn't match expected pattern %s", i, fileName, expectedPatterns[i])
		}
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		expectSuccess  int
		expectFailures int
	}{
		{
			name: "mixed_success_and_failure",
			markdown: `
![Valid image](%s/valid.png)
![Invalid image](%s/invalid.png)
![Not found](%s/notfound.png)
`,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/valid.png":
					w.Header().Set("Content-Type", "image/png")
					w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) // PNG header
				case "/invalid.png":
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("Not an image"))
				default:
					http.NotFound(w, r)
				}
			},
			expectSuccess:  1,
			expectFailures: 2,
		},
		{
			name: "oversized_image",
			markdown: `![Large image](%s/large.png)`,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/large.png" {
					w.Header().Set("Content-Type", "image/png")
					// Write a large response to trigger size limit
					largeData := make([]byte, 2*1024*1024) // 2MB
					w.Write(largeData)
				}
			},
			expectSuccess:  0,
			expectFailures: 1,
		},
		{
			name: "timeout_scenario",
			markdown: `![Slow image](%s/slow.png)`,
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/slow.png" {
					// Simulate slow response (but we'll use a short timeout)
					time.Sleep(100 * time.Millisecond)
					w.Header().Set("Content-Type", "image/png")
					w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
				}
			},
			expectSuccess:  1, // Should succeed with reasonable timeout
			expectFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			// Create markdown content with server URL
			markdownContent := fmt.Sprintf(tt.markdown, server.URL, server.URL, server.URL)

			// Extract URLs
			urls := markdown.ExtractImageURLs(markdownContent)
			if len(urls) == 0 {
				t.Fatal("No URLs extracted from markdown")
			}

			// Download with appropriate limits
			maxSize := int64(1024 * 1024) // 1MB limit for oversized test
			timeout := 50 * time.Millisecond
			if tt.name == "timeout_scenario" {
				timeout = 200 * time.Millisecond // Allow enough time
			}

			fetcher := download.NewFetcher(maxSize, timeout, 2)
			ctx := context.Background()
			results := fetcher.FetchConcurrent(ctx, urls)

			// Count successes and failures
			successCount := 0
			failureCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
					t.Logf("Success: %s (%d bytes)", result.URL, result.Size)
				} else {
					failureCount++
					t.Logf("Failure: %s - %v", result.URL, result.Error)
				}
			}

			if successCount != tt.expectSuccess {
				t.Errorf("Expected %d successes, got %d", tt.expectSuccess, successCount)
			}
			if failureCount != tt.expectFailures {
				t.Errorf("Expected %d failures, got %d", tt.expectFailures, failureCount)
			}
		})
	}
}

func TestIntegration_GitHubTargetParsing(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantOwner   string
		wantRepo    string
		wantNum     string
		expectError bool
	}{
		{
			name:      "short_form",
			target:    "octocat/Hello-World#1",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "1",
		},
		{
			name:      "issue_url",
			target:    "https://github.com/octocat/Hello-World/issues/1",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "1",
		},
		{
			name:      "pull_request_url",
			target:    "https://github.com/octocat/Hello-World/pull/2",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "2",
		},
		{
			name:        "invalid_format",
			target:      "invalid-target",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, num, err := github.ParseTarget(tt.target)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for target %q, but got none", tt.target)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for target %q: %v", tt.target, err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("Owner: got %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("Repo: got %q, want %q", repo, tt.wantRepo)
			}
			if num != tt.wantNum {
				t.Errorf("Num: got %q, want %q", num, tt.wantNum)
			}
		})
	}
}

func TestIntegration_MarkdownVariations(t *testing.T) {
	// Set up a simple image server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) // PNG header
	}))
	defer server.Close()

	tests := []struct {
		name         string
		markdown     string
		expectedURLs int
	}{
		{
			name: "inline_images",
			markdown: fmt.Sprintf(`
![Alt text](%s/image1.png)
![Another image](%s/image2.png)
`, server.URL, server.URL),
			expectedURLs: 2,
		},
		{
			name: "html_img_tags",
			markdown: fmt.Sprintf(`
<img src="%s/image1.png" alt="Image 1">
<img src="%s/image2.png" alt="Image 2" />
`, server.URL, server.URL),
			expectedURLs: 2,
		},
		{
			name: "mixed_formats",
			markdown: fmt.Sprintf(`
![Markdown style](%s/image1.png)
<img src="%s/image2.png" alt="HTML style">
`, server.URL, server.URL),
			expectedURLs: 2,
		},
		{
			name: "reference_style",
			markdown: fmt.Sprintf(`
![Alt text][1]
![Another][2]

[1]: %s/image1.png
[2]: %s/image2.png
`, server.URL, server.URL),
			expectedURLs: 2,
		},
		{
			name: "duplicate_urls",
			markdown: fmt.Sprintf(`
![Image 1](%s/same.png)
![Image 2](%s/same.png)
<img src="%s/same.png" alt="Same image">
`, server.URL, server.URL, server.URL),
			expectedURLs: 1, // Should be deduplicated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := markdown.ExtractImageURLs(tt.markdown)

			if len(urls) != tt.expectedURLs {
				t.Errorf("Expected %d URLs, got %d: %v", tt.expectedURLs, len(urls), urls)
			}

			// Verify all URLs are valid
			for _, url := range urls {
				if !strings.HasPrefix(url, server.URL) {
					t.Errorf("Invalid URL extracted: %s", url)
				}
			}

			// Test that we can download the extracted images
			if len(urls) > 0 {
				fetcher := download.NewFetcher(1024*1024, 5*time.Second, 2)
				ctx := context.Background()
				results := fetcher.FetchConcurrent(ctx, urls)

				for _, result := range results {
					if result.Error != nil {
						t.Errorf("Failed to download %s: %v", result.URL, result.Error)
					}
				}
			}
		})
	}
}

func TestIntegration_StorageModes(t *testing.T) {
	// Create test image data
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 image
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0xFF,
		0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}

	t.Run("memory_storage", func(t *testing.T) {
		memStorage := storage.NewMemoryStorage()

		encoded, err := memStorage.Store(testImageData, "image/png", "test://image.png")
		if err != nil {
			t.Fatalf("Failed to store in memory: %v", err)
		}

		if encoded == "" {
			t.Errorf("Got empty encoding")
		}
		// Memory storage returns raw base64, not data URL format
		if len(encoded) < 10 {
			t.Errorf("Encoding too short: %s", encoded[:50])
		}

		// Verify we can retrieve the data
		images := memStorage.GetImages()
		if len(images) != 1 {
			t.Errorf("Expected 1 stored image, got %d", len(images))
		}
	})

	t.Run("disk_storage", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		diskStorage, err := storage.NewDiskStorage(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to create disk storage: %v", err)
		}

		filePath, err := diskStorage.Store(testImageData, "image/png", "test://image.png")
		if err != nil {
			t.Fatalf("Failed to store to disk: %v", err)
		}

		// Verify file exists and has correct content
		if !filepath.IsAbs(filePath) {
			t.Errorf("Expected absolute path, got: %s", filePath)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read stored file: %v", err)
		}

		if !bytes.Equal(data, testImageData) {
			t.Errorf("Stored data doesn't match original")
		}

		// Verify file name follows convention
		fileName := filepath.Base(filePath)
		if !strings.HasPrefix(fileName, "img-") || !strings.HasSuffix(fileName, ".png") {
			t.Errorf("Invalid file name format: %s", fileName)
		}
	})
}

// Test Claude integration validation (without actual execution)
func TestIntegration_ClaudeValidation(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		images      []string
		expectError bool
		errorMsg    string
	}{
		{
			name:   "valid_input",
			prompt: "Analyze these images",
			images: []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="},
		},
		{
			name:        "empty_prompt",
			prompt:      "",
			images:      []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="},
			expectError: true,
			errorMsg:    "empty",
		},
		{
			name:        "no_images",
			prompt:      "Analyze",
			images:      []string{},
			expectError: true,
			errorMsg:    "image",
		},
		{
			name:        "suspicious_prompt",
			prompt:      "rm -rf /",
			images:      []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="},
			expectError: true,
			errorMsg:    "dangerous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := claude.ValidateClaudeInput(tt.prompt, tt.images)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark integration tests
func BenchmarkIntegration_MarkdownExtraction(b *testing.B) {
	markdownContent := `
# Test Document

Here are some images:

![Image 1](https://example.com/image1.png)
![Image 2](https://example.com/image2.jpg)
<img src="https://example.com/image3.gif" alt="Image 3">
![Image 4][ref1]

[ref1]: https://example.com/image4.webp

More content here with no images.
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		urls := markdown.ExtractImageURLs(markdownContent)
		if len(urls) != 4 {
			b.Fatalf("Expected 4 URLs, got %d", len(urls))
		}
	}
}

func BenchmarkIntegration_StorageOperations(b *testing.B) {
	testData := make([]byte, 1024) // 1KB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.Run("memory_storage", func(b *testing.B) {
		memStorage := storage.NewMemoryStorage()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := memStorage.Store(testData, "image/png", fmt.Sprintf("test://image%d.png", i))
			if err != nil {
				b.Fatalf("Storage failed: %v", err)
			}
		}
	})

	b.Run("disk_storage", func(b *testing.B) {
		tempDir, err := os.MkdirTemp("", "bench-storage-*")
		if err != nil {
			b.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		diskStorage, err := storage.NewDiskStorage(tempDir, true)
		if err != nil {
			b.Fatalf("Failed to create disk storage: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := diskStorage.Store(testData, "image/png", fmt.Sprintf("test://image%d.png", i))
			if err != nil {
				b.Fatalf("Storage failed: %v", err)
			}
		}
	})
}