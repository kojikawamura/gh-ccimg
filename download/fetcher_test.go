package download

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewFetcher(t *testing.T) {
	maxSize := int64(1024 * 1024) // 1MB
	timeout := 30 * time.Second
	concurrency := 5

	fetcher := NewFetcher(maxSize, timeout, concurrency)

	if fetcher == nil {
		t.Fatal("NewFetcher returned nil")
	}

	if fetcher.maxSize != maxSize {
		t.Errorf("maxSize = %d, want %d", fetcher.maxSize, maxSize)
	}
	if fetcher.timeout != timeout {
		t.Errorf("timeout = %v, want %v", fetcher.timeout, timeout)
	}
	if fetcher.concurrency != concurrency {
		t.Errorf("concurrency = %d, want %d", fetcher.concurrency, concurrency)
	}
}

func TestFetcher_FetchSingle_Success(t *testing.T) {
	// Create test server
	testData := []byte("fake image data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	fetcher := NewFetcher(1024*1024, 30*time.Second, 5)
	ctx := context.Background()

	result := fetcher.FetchSingle(ctx, server.URL)

	if result.Error != nil {
		t.Fatalf("FetchSingle failed: %v", result.Error)
	}

	if result.URL != server.URL {
		t.Errorf("URL = %q, want %q", result.URL, server.URL)
	}

	if !bytes.Equal(result.Data, testData) {
		t.Errorf("Data = %v, want %v", result.Data, testData)
	}

	if result.ContentType != "image/png" {
		t.Errorf("ContentType = %q, want %q", result.ContentType, "image/png")
	}

	if result.Size != int64(len(testData)) {
		t.Errorf("Size = %d, want %d", result.Size, len(testData))
	}
}

func TestFetcher_FetchSingle_InvalidContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not an image"))
	}))
	defer server.Close()

	fetcher := NewFetcher(1024*1024, 30*time.Second, 5)
	ctx := context.Background()

	result := fetcher.FetchSingle(ctx, server.URL)

	if result.Error == nil {
		t.Fatal("Expected error for invalid content type")
	}

	if !strings.Contains(result.Error.Error(), "invalid content type") {
		t.Errorf("Error = %v, want error containing 'invalid content type'", result.Error)
	}
}

func TestFetcher_FetchSingle_SizeLimit(t *testing.T) {
	// Create large data that exceeds limit
	largeData := make([]byte, 1024) // 1KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	// Set max size to 512 bytes (smaller than our test data)
	fetcher := NewFetcher(512, 30*time.Second, 5)
	ctx := context.Background()

	result := fetcher.FetchSingle(ctx, server.URL)

	if result.Error == nil {
		t.Fatal("Expected error for size limit exceeded")
	}

	if !strings.Contains(result.Error.Error(), "file too large") {
		t.Errorf("Error = %v, want error containing 'file too large'", result.Error)
	}
}

func TestFetcher_FetchSingle_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	fetcher := NewFetcher(1024*1024, 30*time.Second, 5)
	ctx := context.Background()

	result := fetcher.FetchSingle(ctx, server.URL)

	if result.Error == nil {
		t.Fatal("Expected error for HTTP 404")
	}

	if !strings.Contains(result.Error.Error(), "HTTP 404") {
		t.Errorf("Error = %v, want error containing 'HTTP 404'", result.Error)
	}
}

func TestFetcher_FetchSingle_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer server.Close()

	// Set very short timeout
	fetcher := NewFetcher(1024*1024, 50*time.Millisecond, 5)
	ctx := context.Background()

	result := fetcher.FetchSingle(ctx, server.URL)

	if result.Error == nil {
		t.Fatal("Expected timeout error")
	}

	// The error message might vary, but it should be a timeout-related error
	t.Logf("Timeout error (expected): %v", result.Error)
}

func TestFetcher_FetchConcurrent(t *testing.T) {
	// Create test server that returns different responses based on path
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		// Return different data based on path
		w.Write([]byte(fmt.Sprintf("data-%s", r.URL.Path)))
	}))
	defer server.Close()

	urls := []string{
		server.URL + "/1",
		server.URL + "/2",
		server.URL + "/3",
	}

	fetcher := NewFetcher(1024*1024, 30*time.Second, 2)
	ctx := context.Background()

	results := fetcher.FetchConcurrent(ctx, urls)

	if len(results) != len(urls) {
		t.Fatalf("Expected %d results, got %d", len(urls), len(results))
	}

	// Check that all downloads succeeded
	for i, result := range results {
		if result.Error != nil {
			t.Errorf("Result %d failed: %v", i, result.Error)
		}

		if result.ContentType != "image/png" {
			t.Errorf("Result %d ContentType = %q, want %q", i, result.ContentType, "image/png")
		}

		if len(result.Data) == 0 {
			t.Errorf("Result %d has empty data", i)
		}
	}

	// Verify that we got responses from all URLs
	urlsSeen := make(map[string]bool)
	for _, result := range results {
		urlsSeen[result.URL] = true
	}

	for _, url := range urls {
		if !urlsSeen[url] {
			t.Errorf("Missing result for URL: %s", url)
		}
	}
}

func TestFetcher_FetchConcurrent_EmptyURLs(t *testing.T) {
	fetcher := NewFetcher(1024*1024, 30*time.Second, 5)
	ctx := context.Background()

	results := fetcher.FetchConcurrent(ctx, []string{})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty URLs, got %d", len(results))
	}
}

func TestFetcher_FetchConcurrent_Context_Cancellation(t *testing.T) {
	// Create server with slow responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer server.Close()

	urls := []string{server.URL + "/1", server.URL + "/2"}

	fetcher := NewFetcher(1024*1024, 30*time.Second, 2)
	
	// Create context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := fetcher.FetchConcurrent(ctx, urls)

	// At least some results should have context cancellation errors
	var timeoutErrors int
	for _, result := range results {
		if result.Error != nil {
			timeoutErrors++
		}
	}

	if timeoutErrors == 0 {
		t.Error("Expected at least some timeout errors due to context cancellation")
	}
}