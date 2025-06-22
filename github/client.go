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
	timeout time.Duration
}

// NewClient creates a new GitHub client
func NewClient(timeout time.Duration) *Client {
	return &Client{
		timeout: timeout,
	}
}

// FetchIssue retrieves an issue or pull request from GitHub
func (c *Client) FetchIssue(owner, repo, num string) (*Issue, error) {
	if owner == "" || repo == "" || num == "" {
		return nil, fmt.Errorf("owner, repo, and number are required")
	}

	// Use gh api to fetch the issue/PR
	apiPath := fmt.Sprintf("repos/%s/%s/issues/%s", owner, repo, num)
	
	cmd := exec.Command("gh", "api", apiPath)
	if c.timeout > 0 {
		// Note: exec.Command doesn't directly support timeout, but gh CLI has built-in timeouts
		// For now, we rely on gh's default timeout behavior
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Not Found") || strings.Contains(stderr, "404") {
				return nil, fmt.Errorf("issue/PR %s not found in %s/%s", num, owner, repo)
			}
			if strings.Contains(stderr, "Bad credentials") || strings.Contains(stderr, "401") {
				return nil, fmt.Errorf("authentication failed. Please run 'gh auth login'")
			}
			return nil, fmt.Errorf("GitHub API error: %s", stderr)
		}
		return nil, fmt.Errorf("failed to execute gh command: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return &issue, nil
}

// FetchComments retrieves all comments for an issue or pull request
func (c *Client) FetchComments(owner, repo, num string) ([]*Comment, error) {
	if owner == "" || repo == "" || num == "" {
		return nil, fmt.Errorf("owner, repo, and number are required")
	}

	// Use gh api with pagination to fetch all comments
	apiPath := fmt.Sprintf("repos/%s/%s/issues/%s/comments", owner, repo, num)
	
	cmd := exec.Command("gh", "api", "--paginate", apiPath)
	if c.timeout > 0 {
		// Note: exec.Command doesn't directly support timeout, but gh CLI has built-in timeouts
		// For now, we rely on gh's default timeout behavior
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Not Found") || strings.Contains(stderr, "404") {
				return nil, fmt.Errorf("issue/PR %s not found in %s/%s", num, owner, repo)
			}
			if strings.Contains(stderr, "Bad credentials") || strings.Contains(stderr, "401") {
				return nil, fmt.Errorf("authentication failed. Please run 'gh auth login'")
			}
			return nil, fmt.Errorf("GitHub API error: %s", stderr)
		}
		return nil, fmt.Errorf("failed to execute gh command: %w", err)
	}

	var comments []*Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return comments, nil
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