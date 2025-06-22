package security

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kojikawamura/gh-ccimg/download"
)

// TestSecurityIntegration tests various security attack vectors
func TestSecurityIntegration(t *testing.T) {
	t.Run("MaliciousContentTypeAttack", testMaliciousContentTypeAttack)
	t.Run("FileBombAttack", testFileBombAttack)
	t.Run("SlowLorisAttack", testSlowLorisAttack)
	t.Run("PathTraversalInURL", testPathTraversalInURL)
	t.Run("RedirectAttack", testRedirectAttack)
	t.Run("XXEAttack", testXXEAttack)
	t.Run("JavaScriptInjection", testJavaScriptInjection)
}

// testMaliciousContentTypeAttack tests content-type spoofing
func testMaliciousContentTypeAttack(t *testing.T) {
	attacks := []struct {
		name        string
		contentType string
		body        string
		shouldBlock bool
	}{
		{
			name:        "executable disguised as PNG",
			contentType: "image/png",
			body:        "\x00\x00\x00\x00IHDR\x00\x00\x00\x00MZP\x00\x00\x00", // PE header disguised
			shouldBlock: false, // Should pass content-type but fail on actual content
		},
		{
			name:        "script disguised as JPEG",
			contentType: "image/jpeg",
			body:        "<script>alert('xss')</script>",
			shouldBlock: false, // Content-type validation allows, but content is wrong
		},
		{
			name:        "malicious SVG",
			contentType: "image/svg+xml",
			body: `<?xml version="1.0"?>
			<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
			<svg xmlns="http://www.w3.org/2000/svg">
				<script>alert('XSS')</script>
			</svg>`,
			shouldBlock: false, // SVG is valid content-type but contains scripts
		},
		{
			name:        "binary executable",
			contentType: "application/x-msdownload",
			body:        "MZP\x00\x00\x00\x00executable content",
			shouldBlock: true, // Should be blocked by content-type validation
		},
	}

	for _, attack := range attacks {
		t.Run(attack.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", attack.contentType)
				w.Write([]byte(attack.body))
			}))
			defer server.Close()

			fetcher := download.NewFetcher(1024*1024, 5*time.Second, 1)
			result := fetcher.FetchSingle(context.Background(), server.URL)

			if attack.shouldBlock && result.Error == nil {
				t.Errorf("Attack %s should have been blocked but was not", attack.name)
			}
			if !attack.shouldBlock && result.Error != nil && strings.Contains(result.Error.Error(), "content-type") {
				t.Errorf("Attack %s was blocked by content-type but shouldn't have been", attack.name)
			}
		})
	}
}

// testFileBombAttack tests protection against large file attacks
func testFileBombAttack(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		maxSize  int64
		shouldBlock bool
	}{
		{
			name:     "normal size",
			fileSize: 1024,     // 1KB
			maxSize:  1024*1024, // 1MB
			shouldBlock: false,
		},
		{
			name:     "exactly at limit",
			fileSize: 1024*1024, // 1MB
			maxSize:  1024*1024, // 1MB
			shouldBlock: false,
		},
		{
			name:     "slightly over limit",
			fileSize: 1024*1024 + 1, // 1MB + 1 byte
			maxSize:  1024*1024,     // 1MB
			shouldBlock: true,
		},
		{
			name:     "massive file bomb",
			fileSize: 100*1024*1024, // 100MB
			maxSize:  1024*1024,     // 1MB limit
			shouldBlock: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Content-Length", fmt.Sprintf("%d", test.fileSize))
				
				// Write data in chunks to simulate streaming
				written := int64(0)
				chunkSize := int64(8192) // 8KB chunks
				for written < test.fileSize {
					remaining := test.fileSize - written
					if remaining < chunkSize {
						chunkSize = remaining
					}
					data := make([]byte, chunkSize)
					w.Write(data)
					written += chunkSize
					
					// Stop if we've written enough to trigger the limit
					if written > test.maxSize+1 {
						break
					}
				}
			}))
			defer server.Close()

			fetcher := download.NewFetcher(test.maxSize, 10*time.Second, 1)
			result := fetcher.FetchSingle(context.Background(), server.URL)

			if test.shouldBlock && result.Error == nil {
				t.Errorf("Large file attack should have been blocked but was not. File size: %d, limit: %d", test.fileSize, test.maxSize)
			}
			if !test.shouldBlock && result.Error != nil {
				t.Errorf("Normal file was blocked: %v", result.Error)
			}
		})
	}
}

// testSlowLorisAttack tests protection against slow response attacks
func testSlowLorisAttack(t *testing.T) {
	tests := []struct {
		name           string
		delayPerChunk  time.Duration
		chunks         int
		timeout        time.Duration
		shouldTimeout  bool
	}{
		{
			name:          "normal speed",
			delayPerChunk: 10 * time.Millisecond,
			chunks:        10,
			timeout:       5 * time.Second,
			shouldTimeout: false,
		},
		{
			name:          "slow but within timeout",
			delayPerChunk: 100 * time.Millisecond,
			chunks:        10,
			timeout:       5 * time.Second,
			shouldTimeout: false,
		},
		{
			name:          "slow loris attack",
			delayPerChunk: 1 * time.Second,
			chunks:        10,
			timeout:       3 * time.Second,
			shouldTimeout: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				
				for i := 0; i < test.chunks; i++ {
					time.Sleep(test.delayPerChunk)
					w.Write([]byte("data chunk "))
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}))
			defer server.Close()

			fetcher := download.NewFetcher(1024*1024, test.timeout, 1)
			result := fetcher.FetchSingle(context.Background(), server.URL)

			if test.shouldTimeout && result.Error == nil {
				t.Errorf("Slow loris attack should have timed out but didn't")
			}
			if !test.shouldTimeout && result.Error != nil {
				t.Errorf("Normal slow response was blocked: %v", result.Error)
			}
		})
	}
}

// testPathTraversalInURL tests that malicious URLs don't cause path traversal
func testPathTraversalInURL(t *testing.T) {
	maliciousURLs := []string{
		"http://example.com/../../../etc/passwd",
		"http://example.com/..%2F..%2F..%2Fetc%2Fpasswd",
		"http://example.com/%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"file:///etc/passwd",
		"file://C:/windows/system32/config/sam",
	}

	for _, url := range maliciousURLs {
		t.Run(fmt.Sprintf("URL: %s", url), func(t *testing.T) {
			fetcher := download.NewFetcher(1024*1024, 5*time.Second, 1)
			result := fetcher.FetchSingle(context.Background(), url)

			// These should fail for various reasons (invalid scheme, network error, etc.)
			// The important thing is they don't succeed in accessing local files
			if result.Error == nil {
				t.Errorf("Malicious URL should have failed but succeeded: %s", url)
			}
		})
	}
}

// testRedirectAttack tests protection against malicious redirects
func testRedirectAttack(t *testing.T) {
	// Create a server that redirects to localhost/internal URLs
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to potentially dangerous internal URLs
		dangerousURLs := []string{
			"http://localhost:22/", // SSH
			"http://127.0.0.1:3306/", // MySQL
			"http://169.254.169.254/", // AWS metadata
			"file:///etc/passwd",
		}
		
		if len(dangerousURLs) > 0 {
			http.Redirect(w, r, dangerousURLs[0], http.StatusMovedPermanently)
		}
	}))
	defer redirectServer.Close()

	fetcher := download.NewFetcher(1024*1024, 5*time.Second, 1)
	result := fetcher.FetchSingle(context.Background(), redirectServer.URL)

	// The redirect should fail (connection refused, invalid scheme, etc.)
	// Important: it shouldn't succeed in accessing internal services
	if result.Error == nil {
		t.Errorf("Redirect attack should have failed but succeeded")
	}
}

// testXXEAttack tests XML External Entity attacks (primarily for SVG)
func testXXEAttack(t *testing.T) {
	xxePayload := `<?xml version="1.0" encoding="UTF-8"?>
	<!DOCTYPE svg [
	  <!ENTITY xxe SYSTEM "file:///etc/passwd">
	]>
	<svg xmlns="http://www.w3.org/2000/svg">
	  <text>&xxe;</text>
	</svg>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(xxePayload))
	}))
	defer server.Close()

	fetcher := download.NewFetcher(1024*1024, 5*time.Second, 1)
	result := fetcher.FetchSingle(context.Background(), server.URL)

	// The download should succeed (it's valid SVG content-type)
	// But the XXE attack should not be processed since we're just downloading, not parsing
	if result.Error != nil {
		t.Errorf("XXE SVG download failed: %v", result.Error)
	}

	// Verify the content doesn't contain actual file contents
	if strings.Contains(string(result.Data), "root:") {
		t.Errorf("XXE attack succeeded - file contents found in downloaded data")
	}
}

// testJavaScriptInjection tests that downloaded content doesn't execute scripts
func testJavaScriptInjection(t *testing.T) {
	jsPayloads := []struct {
		name        string
		contentType string
		payload     string
	}{
		{
			name:        "HTML with script",
			contentType: "image/png", // Spoofed content-type
			payload:     `<html><script>document.write('PWNED')</script></html>`,
		},
		{
			name:        "SVG with script",
			contentType: "image/svg+xml",
			payload: `<svg xmlns="http://www.w3.org/2000/svg">
				<script>alert('XSS')</script>
			</svg>`,
		},
	}

	for _, test := range jsPayloads {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				w.Write([]byte(test.payload))
			}))
			defer server.Close()

			fetcher := download.NewFetcher(1024*1024, 5*time.Second, 1)
			result := fetcher.FetchSingle(context.Background(), server.URL)

			// The download itself should work (we're just downloading bytes)
			expectedToPass := test.contentType == "image/svg+xml" || test.contentType == "image/png"
			if expectedToPass && result.Error != nil && strings.Contains(result.Error.Error(), "content-type") {
				t.Errorf("JS injection test failed content-type validation: %v", result.Error)
			}

			// Most importantly, verify the content is just stored as bytes, not executed
			if result.Data != nil && string(result.Data) != test.payload {
				t.Errorf("Downloaded content was modified, potential execution detected")
			}
		})
	}
}