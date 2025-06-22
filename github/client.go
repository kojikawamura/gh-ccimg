package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Issue represents a GitHub issue or pull request
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
}

// Comment represents a GitHub issue/PR comment
type Comment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Client handles GitHub API interactions via gh CLI
type Client struct {
	timeout    time.Duration
	maxRetries int
	baseDelay  time.Duration
}

// NewClient creates a new GitHub client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		timeout:    timeout,
		maxRetries: 3,                        // Default 3 retries
		baseDelay:  1 * time.Second,          // Default 1s base delay for GitHub API
	}
}

// FetchIssue retrieves an issue or pull request from GitHub with retry logic
func (c *Client) FetchIssue(owner, repo, num string) (*Issue, error) {
	if owner == "" || repo == "" || num == "" {
		return nil, fmt.Errorf("owner, repo, and number are required")
	}

	apiPath := fmt.Sprintf("repos/%s/%s/issues/%s", owner, repo, num)
	
	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		cmd := exec.Command("gh", "api", apiPath)
		
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr := string(exitErr.Stderr)
				
				// Don't retry on authentication or not found errors
				if strings.Contains(stderr, "Not Found") || strings.Contains(stderr, "404") {
					return nil, fmt.Errorf("issue/PR %s not found in %s/%s", num, owner, repo)
				}
				if strings.Contains(stderr, "Bad credentials") || strings.Contains(stderr, "401") {
					return nil, fmt.Errorf("authentication failed. Please run 'gh auth login'")
				}
				
				// Retry on rate limiting or server errors
				if attempt < c.maxRetries && c.isRetryableGitHubError(stderr) {
					delay := c.calculateBackoffDelay(attempt)
					time.Sleep(delay)
					continue
				}
				
				return nil, fmt.Errorf("GitHub API error after %d attempts: %s", attempt+1, stderr)
			}
			
			// Retry on general execution errors
			if attempt < c.maxRetries {
				delay := c.calculateBackoffDelay(attempt)
				time.Sleep(delay)
				continue
			}
			
			return nil, fmt.Errorf("failed to execute gh command after %d attempts: %w", attempt+1, err)
		}

		var issue Issue
		if err := json.Unmarshal(output, &issue); err != nil {
			return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
		}

		return &issue, nil
	}

	return nil, fmt.Errorf("unexpected error in retry loop")
}

// FetchComments retrieves all comments for an issue or pull request with retry logic
func (c *Client) FetchComments(owner, repo, num string) ([]*Comment, error) {
	if owner == "" || repo == "" || num == "" {
		return nil, fmt.Errorf("owner, repo, and number are required")
	}

	apiPath := fmt.Sprintf("repos/%s/%s/issues/%s/comments", owner, repo, num)
	
	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		cmd := exec.Command("gh", "api", "--paginate", apiPath)
		
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr := string(exitErr.Stderr)
				
				// Don't retry on authentication or not found errors
				if strings.Contains(stderr, "Not Found") || strings.Contains(stderr, "404") {
					return nil, fmt.Errorf("issue/PR %s not found in %s/%s", num, owner, repo)
				}
				if strings.Contains(stderr, "Bad credentials") || strings.Contains(stderr, "401") {
					return nil, fmt.Errorf("authentication failed. Please run 'gh auth login'")
				}
				
				// Retry on rate limiting or server errors
				if attempt < c.maxRetries && c.isRetryableGitHubError(stderr) {
					delay := c.calculateBackoffDelay(attempt)
					time.Sleep(delay)
					continue
				}
				
				return nil, fmt.Errorf("GitHub API error after %d attempts: %s", attempt+1, stderr)
			}
			
			// Retry on general execution errors
			if attempt < c.maxRetries {
				delay := c.calculateBackoffDelay(attempt)
				time.Sleep(delay)
				continue
			}
			
			return nil, fmt.Errorf("failed to execute gh command after %d attempts: %w", attempt+1, err)
		}

		var comments []*Comment
		if err := json.Unmarshal(output, &comments); err != nil {
			return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
		}

		return comments, nil
	}

	return nil, fmt.Errorf("unexpected error in retry loop")
}

// IsGHCliAvailable checks if gh CLI is available and authenticated
func IsGHCliAvailable() error {
	// Check if gh command exists
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not found. Please install GitHub CLI: https://cli.github.com/")
	}

	// Check if authenticated
	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not authenticated. Please run 'gh auth login'")
	}

	return nil
}

// isRetryableGitHubError determines if a GitHub API error should trigger a retry
func (c *Client) isRetryableGitHubError(stderr string) bool {
	errorStr := strings.ToLower(stderr)
	
	// Retry on rate limiting and server errors
	retryableErrors := []string{
		"rate limit",
		"api rate limit",
		"secondary rate limit",
		"server error",
		"internal server error",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
		"timeout",
		"temporary failure",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}
	
	return false
}

// calculateBackoffDelay calculates exponential backoff delay for GitHub API
func (c *Client) calculateBackoffDelay(attempt int) time.Duration {
	// Exponential backoff: base_delay * 2^attempt
	delay := c.baseDelay * time.Duration(1<<uint(attempt))
	
	// Add some jitter (up to 25% of the delay)
	jitter := time.Duration(delay.Nanoseconds() / 4) // 25% jitter
	if jitter > 0 {
		delay += time.Duration(attempt * int(jitter.Nanoseconds()) % int(jitter.Nanoseconds()))
	}
	
	// Cap at 30 seconds maximum for GitHub API
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}