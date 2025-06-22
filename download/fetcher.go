package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Result represents the result of downloading a single URL
type Result struct {
	URL         string
	Data        []byte
	ContentType string
	Size        int64
	Error       error
}

// Fetcher handles concurrent image downloading with guards
type Fetcher struct {
	client      *http.Client
	maxSize     int64
	timeout     time.Duration
	concurrency int
	reporter    Reporter
	maxRetries  int
	baseDelay   time.Duration
}

// NewFetcher creates a new fetcher with the specified limits
func NewFetcher(maxSize int64, timeout time.Duration, concurrency int) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		maxSize:     maxSize,
		timeout:     timeout,
		concurrency: concurrency,
		reporter:    NewNoOpReporter(), // Default to no-op
		maxRetries:  3,                  // Default 3 retries
		baseDelay:   500 * time.Millisecond, // Default 500ms base delay
	}
}

// SetReporter sets the progress reporter
func (f *Fetcher) SetReporter(reporter Reporter) {
	f.reporter = reporter
}

// FetchConcurrent downloads multiple URLs concurrently
func (f *Fetcher) FetchConcurrent(ctx context.Context, urls []string) []Result {
	if len(urls) == 0 {
		return []Result{}
	}

	f.reporter.Start(len(urls))
	defer f.reporter.Finish()

	// Create channels for work distribution
	urlChan := make(chan string, len(urls))
	resultChan := make(chan Result, len(urls))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < f.concurrency; i++ {
		wg.Add(1)
		go f.worker(ctx, &wg, urlChan, resultChan)
	}

	// Send URLs to workers
	for _, url := range urls {
		urlChan <- url
	}
	close(urlChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []Result
	completed := 0
	for result := range resultChan {
		results = append(results, result)
		completed++
		f.reporter.Update(completed, result.URL, result.Error == nil, result.Error)
	}

	return results
}

// worker is a worker goroutine that processes URLs from the channel
func (f *Fetcher) worker(ctx context.Context, wg *sync.WaitGroup, urlChan <-chan string, resultChan chan<- Result) {
	defer wg.Done()

	for url := range urlChan {
		select {
		case <-ctx.Done():
			resultChan <- Result{
				URL:   url,
				Error: ctx.Err(),
			}
			return
		default:
			result := f.fetchSingle(ctx, url)
			resultChan <- result
		}
	}
}

// fetchSingle downloads a single URL with size and content-type validation and retry logic
func (f *Fetcher) fetchSingle(ctx context.Context, url string) Result {
	result := Result{URL: url}

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		// Create request with context
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			result.Error = fmt.Errorf("failed to create request: %w", err)
			return result // Don't retry on request creation errors
		}

		// Set user agent
		req.Header.Set("User-Agent", "gh-ccimg/1.0")

		// Perform request
		resp, err := f.client.Do(req)
		if err != nil {
			if attempt < f.maxRetries && f.isRetryableError(err) {
				delay := f.calculateBackoffDelay(attempt)
				time.Sleep(delay)
				continue
			}
			result.Error = fmt.Errorf("HTTP request failed after %d attempts: %w", attempt+1, err)
			return result
		}
		defer resp.Body.Close()

		// Check HTTP status
		if resp.StatusCode != http.StatusOK {
			if attempt < f.maxRetries && f.isRetryableStatusCode(resp.StatusCode) {
				resp.Body.Close()
				delay := f.calculateBackoffDelay(attempt)
				time.Sleep(delay)
				continue
			}
			result.Error = fmt.Errorf("HTTP %d: %s (after %d attempts)", resp.StatusCode, resp.Status, attempt+1)
			return result
		}

		// Get and validate content type
		contentType := resp.Header.Get("Content-Type")
		if err := ValidateContentType(contentType); err != nil {
			result.Error = err
			return result // Don't retry on content type validation errors
		}
		result.ContentType = contentType

		// Check content length if available
		if resp.ContentLength > 0 {
			if resp.ContentLength > f.maxSize {
				result.Error = fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, f.maxSize)
				return result // Don't retry on size validation errors
			}
		}

		// Read body with size limit
		limitedReader := &io.LimitedReader{
			R: resp.Body,
			N: f.maxSize + 1, // +1 to detect if we exceed limit
		}

		data, err := io.ReadAll(limitedReader)
		if err != nil {
			if attempt < f.maxRetries {
				resp.Body.Close()
				delay := f.calculateBackoffDelay(attempt)
				time.Sleep(delay)
				continue
			}
			result.Error = fmt.Errorf("failed to read response body after %d attempts: %w", attempt+1, err)
			return result
		}

		// Check if we exceeded size limit
		if int64(len(data)) > f.maxSize {
			result.Error = fmt.Errorf("file too large: %d bytes (max %d)", len(data), f.maxSize)
			return result // Don't retry on size validation errors
		}

		// Success - return the result
		result.Data = data
		result.Size = int64(len(data))
		return result
	}

	// This should never be reached, but just in case
	result.Error = fmt.Errorf("unexpected error in retry loop")
	return result
}

// FetchSingle downloads a single URL (convenience method)
func (f *Fetcher) FetchSingle(ctx context.Context, url string) Result {
	return f.fetchSingle(ctx, url)
}

// isRetryableError determines if an error should trigger a retry
func (f *Fetcher) isRetryableError(err error) bool {
	// Retry on network errors, timeouts, temporary failures
	if err == nil {
		return false
	}
	
	// Simple heuristic: retry on common network errors
	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no such host",
		"i/o timeout",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}
	
	return false
}

// isRetryableStatusCode determines if an HTTP status code should trigger a retry
func (f *Fetcher) isRetryableStatusCode(statusCode int) bool {
	// Retry on server errors (5xx) and rate limiting (429)
	switch statusCode {
	case 429: // Too Many Requests (rate limiting)
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	default:
		return false
	}
}

// calculateBackoffDelay calculates exponential backoff delay
func (f *Fetcher) calculateBackoffDelay(attempt int) time.Duration {
	// Exponential backoff: base_delay * 2^attempt with jitter
	delay := f.baseDelay * time.Duration(1<<uint(attempt))
	
	// Add some jitter (up to 25% of the delay)
	jitter := time.Duration(delay.Nanoseconds() / 4) // 25% jitter
	if jitter > 0 {
		delay += time.Duration(attempt * int(jitter.Nanoseconds()) % int(jitter.Nanoseconds()))
	}
	
	// Cap at 10 seconds maximum
	maxDelay := 10 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}