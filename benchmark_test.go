package main

import (
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

// Performance target constants based on PLAN.md
const (
	TARGET_SMALL_IMAGES_COUNT    = 10
	TARGET_SMALL_IMAGES_TIME     = 2 * time.Second
	TARGET_LARGE_IMAGES_COUNT    = 50
	TARGET_LARGE_IMAGES_TIME     = 10 * time.Second
	DEFAULT_CONCURRENT_DOWNLOADS = 5
)

// BenchmarkParseTarget tests URL parsing performance
func BenchmarkParseTarget(b *testing.B) {
	targets := []string{
		"owner/repo#123",
		"https://github.com/owner/repo/issues/123",
		"https://github.com/owner/repo/pull/456",
		"organization/very-long-project-name#789",
		"user/repo-with-dashes-and-numbers-123#999",
	}

	for _, target := range targets {
		b.Run(fmt.Sprintf("target_%s", strings.ReplaceAll(target, "/", "_")), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				owner, repo, num, err := github.ParseTarget(target)
				if err != nil {
					b.Fatalf("Parse failed: %v", err)
				}
				// Prevent compiler optimization
				_ = owner + repo + num
			}
		})
	}
}

// BenchmarkExtractImageURLs tests markdown parsing performance
func BenchmarkExtractImageURLs(b *testing.B) {
	// Load test markdown files
	testdataDir := "./testdata"
	
	markdownFiles := map[string]string{
		"simple":    filepath.Join(testdataDir, "markdown", "simple.md"),
		"complex":   filepath.Join(testdataDir, "markdown", "complex.md"),
		"malformed": filepath.Join(testdataDir, "markdown", "malformed.md"),
	}

	for name, filePath := range markdownFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			b.Skipf("Cannot read test file %s: %v", filePath, err)
			continue
		}

		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				urls := markdown.ExtractImageURLs(string(content))
				// Prevent compiler optimization
				_ = len(urls)
			}
		})
	}

	// Test with various sizes of markdown content
	sizeBenchmarks := []struct {
		name string
		size int
	}{
		{"small_1KB", 1024},
		{"medium_10KB", 10 * 1024},
		{"large_100KB", 100 * 1024},
		{"huge_1MB", 1024 * 1024},
	}

	for _, sb := range sizeBenchmarks {
		b.Run(sb.name, func(b *testing.B) {
			// Generate markdown content with images
			var content strings.Builder
			content.WriteString("# Performance Test Document\n\n")
			
			imageCount := sb.size / 100 // Roughly 1 image per 100 bytes
			for i := 0; i < imageCount; i++ {
				content.WriteString(fmt.Sprintf("![Image %d](https://example.com/image%d.png)\n", i, i))
			}
			
			// Fill remaining space with text
			remaining := sb.size - content.Len()
			if remaining > 0 {
				filler := strings.Repeat("Lorem ipsum dolor sit amet. ", remaining/28+1)
				content.WriteString(filler[:remaining])
			}

			markdownText := content.String()
			
			b.ResetTimer()
			b.SetBytes(int64(len(markdownText)))
			for i := 0; i < b.N; i++ {
				urls := markdown.ExtractImageURLs(markdownText)
				_ = len(urls)
			}
		})
	}
}

// BenchmarkConcurrentDownload tests download throughput and scalability
func BenchmarkConcurrentDownload(b *testing.B) {
	// Create test server with controllable response times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different image sizes
		var size int
		switch {
		case strings.Contains(r.URL.Path, "small"):
			size = 1024 // 1KB
		case strings.Contains(r.URL.Path, "medium"):
			size = 100 * 1024 // 100KB
		case strings.Contains(r.URL.Path, "large"):
			size = 1024 * 1024 // 1MB
		default:
			size = 10 * 1024 // 10KB default
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		
		// Generate simple test data
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		w.Write(data)
	}))
	defer server.Close()

	downloadTests := []struct {
		name           string
		imageCount     int
		imageSize      string
		concurrency    int
		expectedTime   time.Duration
	}{
		{"1_image_small", 1, "small", 1, 100 * time.Millisecond},
		{"5_images_small", 5, "small", 5, 200 * time.Millisecond},
		{"10_images_small", 10, "small", 5, TARGET_SMALL_IMAGES_TIME},
		{"20_images_medium", 20, "medium", 5, 4 * time.Second},
		{"50_images_medium", 50, "medium", 5, TARGET_LARGE_IMAGES_TIME},
		{"10_images_large", 10, "large", 5, 3 * time.Second},
		{"concurrent_1", 10, "medium", 1, 15 * time.Second},
		{"concurrent_3", 10, "medium", 3, 8 * time.Second},
		{"concurrent_5", 10, "medium", 5, 5 * time.Second},
		{"concurrent_10", 10, "medium", 10, 3 * time.Second},
	}

	for _, dt := range downloadTests {
		b.Run(dt.name, func(b *testing.B) {
			// Generate URLs
			urls := make([]string, dt.imageCount)
			for i := 0; i < dt.imageCount; i++ {
				urls[i] = fmt.Sprintf("%s/%s_image_%d.png", server.URL, dt.imageSize, i)
			}

			fetcher := download.NewFetcher(10*1024*1024, 30*time.Second, dt.concurrency)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()
				results := fetcher.FetchConcurrent(ctx, urls)
				elapsed := time.Since(start)

				// Verify all downloads succeeded
				successCount := 0
				totalBytes := int64(0)
				for _, result := range results {
					if result.Error == nil {
						successCount++
						totalBytes += result.Size
					}
				}

				if successCount != dt.imageCount {
					b.Fatalf("Expected %d successful downloads, got %d", dt.imageCount, successCount)
				}

				b.SetBytes(totalBytes)
				
				// Report if significantly over target time
				if elapsed > dt.expectedTime*2 {
					b.Logf("Warning: Download took %v, expected around %v", elapsed, dt.expectedTime)
				}
			}
		})
	}
}

// BenchmarkStoreMemory tests base64 encoding performance
func BenchmarkStoreMemory(b *testing.B) {
	memStorage := storage.NewMemoryStorage()

	testSizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"5MB", 5 * 1024 * 1024},
	}

	for _, ts := range testSizes {
		b.Run(ts.name, func(b *testing.B) {
			// Generate test data
			data := make([]byte, ts.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			b.ResetTimer()
			b.SetBytes(int64(ts.size))
			for i := 0; i < b.N; i++ {
				encoded, err := memStorage.Store(data, "image/png", 
					fmt.Sprintf("test://image%d.png", i))
				if err != nil {
					b.Fatalf("Store failed: %v", err)
				}
				// Prevent compiler optimization
				_ = len(encoded)
			}
		})
	}
}

// BenchmarkStoreDisk tests file I/O performance
func BenchmarkStoreDisk(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark-storage-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	diskStorage, err := storage.NewDiskStorage(tempDir, true)
	if err != nil {
		b.Fatalf("Failed to create disk storage: %v", err)
	}

	testSizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"5MB", 5 * 1024 * 1024},
	}

	for _, ts := range testSizes {
		b.Run(ts.name, func(b *testing.B) {
			// Generate test data
			data := make([]byte, ts.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			b.ResetTimer()
			b.SetBytes(int64(ts.size))
			for i := 0; i < b.N; i++ {
				filePath, err := diskStorage.Store(data, "image/png", 
					fmt.Sprintf("test://image%d.png", i))
				if err != nil {
					b.Fatalf("Store failed: %v", err)
				}
				// Prevent compiler optimization
				_ = len(filePath)
			}
		})
	}
}

// BenchmarkClaudeCommandBuilding tests Claude CLI command construction
func BenchmarkClaudeCommandBuilding(b *testing.B) {
	testCases := []struct {
		name         string
		prompt       string
		imageCount   int
		continueFlag bool
	}{
		{"simple_prompt_1_image", "Analyze this image", 1, false},
		{"simple_prompt_5_images", "Analyze these images", 5, false},
		{"complex_prompt_10_images", "Please analyze these screenshots and provide detailed feedback on the user interface design, noting any accessibility issues, visual hierarchy problems, and suggestions for improvement", 10, false},
		{"continue_session_5_images", "Continue the analysis", 5, true},
		{"many_images", "Analyze all images", 50, false},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate test images (base64 data URLs)
			images := make([]string, tc.imageCount)
			for i := 0; i < tc.imageCount; i++ {
				images[i] = fmt.Sprintf("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAA%d", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				args := claude.BuildClaudeArgs(tc.prompt, images, tc.continueFlag)
				// Prevent compiler optimization
				_ = len(args)
			}
		})
	}
}

// BenchmarkCompleteWorkflow tests end-to-end performance
func BenchmarkCompleteWorkflow(b *testing.B) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// Small test image for speed
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		w.Write(data)
	}))
	defer server.Close()

	workflowTests := []struct {
		name        string
		imageCount  int
		storageMode string
		targetTime  time.Duration
	}{
		{"5_images_memory", 5, "memory", 1 * time.Second},
		{"10_images_memory", 10, "memory", TARGET_SMALL_IMAGES_TIME},
		{"10_images_disk", 10, "disk", TARGET_SMALL_IMAGES_TIME + 500*time.Millisecond},
		{"25_images_memory", 25, "memory", 5 * time.Second},
		{"50_images_memory", 50, "memory", TARGET_LARGE_IMAGES_TIME},
	}

	for _, wt := range workflowTests {
		b.Run(wt.name, func(b *testing.B) {
			// Generate markdown with images
			var markdownContent strings.Builder
			markdownContent.WriteString("# Performance Test\n\n")
			for i := 0; i < wt.imageCount; i++ {
				markdownContent.WriteString(fmt.Sprintf("![Image %d](%s/image%d.png)\n", 
					i, server.URL, i))
			}

			var tempDir string
			if wt.storageMode == "disk" {
				var err error
				tempDir, err = os.MkdirTemp("", "benchmark-workflow-*")
				if err != nil {
					b.Fatalf("Failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()

				// Step 1: Extract URLs
				urls := markdown.ExtractImageURLs(markdownContent.String())

				// Step 2: Download images
				fetcher := download.NewFetcher(10*1024*1024, 30*time.Second, DEFAULT_CONCURRENT_DOWNLOADS)
				ctx := context.Background()
				results := fetcher.FetchConcurrent(ctx, urls)

				// Step 3: Store images
				var totalBytes int64
				successCount := 0
				
				if wt.storageMode == "memory" {
					memStorage := storage.NewMemoryStorage()
					for _, result := range results {
						if result.Error == nil {
							_, err := memStorage.Store(result.Data, result.ContentType, result.URL)
							if err == nil {
								successCount++
								totalBytes += result.Size
							}
						}
					}
				} else {
					diskStorage, _ := storage.NewDiskStorage(tempDir, true)
					for _, result := range results {
						if result.Error == nil {
							_, err := diskStorage.Store(result.Data, result.ContentType, result.URL)
							if err == nil {
								successCount++
								totalBytes += result.Size
							}
						}
					}
				}

				elapsed := time.Since(start)
				
				if successCount != wt.imageCount {
					b.Fatalf("Expected %d successful operations, got %d", wt.imageCount, successCount)
				}

				b.SetBytes(totalBytes)

				// Check if we're meeting performance targets
				if elapsed > wt.targetTime*2 {
					b.Logf("Warning: Workflow took %v, target was %v", elapsed, wt.targetTime)
				}
			}
		})
	}
}

// BenchmarkScalability tests system behavior under increasing load
func BenchmarkScalability(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// Variable size based on URL path
		size := 10 * 1024 // 10KB default
		if strings.Contains(r.URL.Path, "small") {
			size = 1024 // 1KB
		} else if strings.Contains(r.URL.Path, "large") {
			size = 100 * 1024 // 100KB
		}
		
		data := make([]byte, size)
		w.Write(data)
	}))
	defer server.Close()

	scaleTests := []int{1, 5, 10, 20, 50, 100}
	
	for _, imageCount := range scaleTests {
		b.Run(fmt.Sprintf("%d_images", imageCount), func(b *testing.B) {
			// Generate URLs
			urls := make([]string, imageCount)
			for i := 0; i < imageCount; i++ {
				size := "small"
				if i%3 == 0 {
					size = "large"
				}
				urls[i] = fmt.Sprintf("%s/%s_image_%d.png", server.URL, size, i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fetcher := download.NewFetcher(50*1024*1024, 60*time.Second, DEFAULT_CONCURRENT_DOWNLOADS)
				ctx := context.Background()
				
				start := time.Now()
				results := fetcher.FetchConcurrent(ctx, urls)
				elapsed := time.Since(start)

				successCount := 0
				totalBytes := int64(0)
				for _, result := range results {
					if result.Error == nil {
						successCount++
						totalBytes += result.Size
					}
				}

				b.SetBytes(totalBytes)
				
				// Report throughput
				if elapsed > 0 {
					throughput := float64(totalBytes) / elapsed.Seconds() / 1024 / 1024 // MB/s
					b.ReportMetric(throughput, "MB/s")
					b.ReportMetric(float64(successCount)/elapsed.Seconds(), "images/s")
				}
			}
		})
	}
}

// BenchmarkMemoryUsage tests memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	// Test with large number of small images vs small number of large images
	memoryTests := []struct {
		name       string
		imageCount int
		imageSize  int
	}{
		{"many_small", 100, 1024},      // 100 x 1KB = 100KB total
		{"few_large", 10, 10*1024},     // 10 x 10KB = 100KB total  
		{"moderate", 50, 2*1024},       // 50 x 2KB = 100KB total
	}

	for _, mt := range memoryTests {
		b.Run(mt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				memStorage := storage.NewMemoryStorage()
				
				totalBytes := int64(0)
				for j := 0; j < mt.imageCount; j++ {
					data := make([]byte, mt.imageSize)
					for k := range data {
						data[k] = byte(k % 256)
					}
					
					_, err := memStorage.Store(data, "image/png", 
						fmt.Sprintf("test://image%d.png", j))
					if err != nil {
						b.Fatalf("Store failed: %v", err)
					}
					totalBytes += int64(mt.imageSize)
				}

				b.SetBytes(totalBytes)
				
				// Report memory efficiency metric
				memoryUsage := memStorage.EstimateMemoryUsage()
				efficiency := float64(totalBytes) / float64(memoryUsage) * 100
				b.ReportMetric(efficiency, "efficiency_%")
			}
		})
	}
}

// BenchmarkRegressionProtection ensures performance doesn't degrade
func BenchmarkRegressionProtection(b *testing.B) {
	// These benchmarks establish baseline performance expectations
	
	b.Run("baseline_parse_target", func(b *testing.B) {
		target := "owner/repo#123"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			github.ParseTarget(target)
		}
	})

	b.Run("baseline_extract_urls", func(b *testing.B) {
		markdownContent := "![Image](https://example.com/test.png)"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			markdown.ExtractImageURLs(markdownContent)
		}
	})

	b.Run("baseline_memory_store", func(b *testing.B) {
		memStorage := storage.NewMemoryStorage()
		data := make([]byte, 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			memStorage.Store(data, "image/png", fmt.Sprintf("test://image%d.png", i))
		}
	})
}