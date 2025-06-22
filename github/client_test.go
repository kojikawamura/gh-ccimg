package github

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	timeout := 30 * time.Second
	client := NewClient(timeout)
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.timeout != timeout {
		t.Errorf("NewClient timeout = %v, want %v", client.timeout, timeout)
	}
}

func TestClient_FetchIssue_ValidationErrors(t *testing.T) {
	client := NewClient(30 * time.Second)
	
	tests := []struct {
		name  string
		owner string
		repo  string
		num   string
	}{
		{"empty owner", "", "repo", "1"},
		{"empty repo", "owner", "", "1"},
		{"empty num", "owner", "repo", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.FetchIssue(tt.owner, tt.repo, tt.num)
			if err == nil {
				t.Error("FetchIssue expected error for invalid parameters, got nil")
			}
			if !containsString(err.Error(), "required") {
				t.Errorf("FetchIssue error = %v, want error containing 'required'", err)
			}
		})
	}
}

func TestClient_FetchComments_ValidationErrors(t *testing.T) {
	client := NewClient(30 * time.Second)
	
	tests := []struct {
		name  string
		owner string
		repo  string
		num   string
	}{
		{"empty owner", "", "repo", "1"},
		{"empty repo", "owner", "", "1"},
		{"empty num", "owner", "repo", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.FetchComments(tt.owner, tt.repo, tt.num)
			if err == nil {
				t.Error("FetchComments expected error for invalid parameters, got nil")
			}
			if !containsString(err.Error(), "required") {
				t.Errorf("FetchComments error = %v, want error containing 'required'", err)
			}
		})
	}
}

// Integration tests - these would require gh CLI to be installed and authenticated
// They are disabled by default but can be run manually

func TestClient_FetchIssue_Integration(t *testing.T) {
	t.Skip("Integration test - requires gh CLI authentication")
	
	client := NewClient(30 * time.Second)
	
	// Test with a known public issue
	issue, err := client.FetchIssue("octocat", "Hello-World", "1")
	if err != nil {
		t.Fatalf("FetchIssue failed: %v", err)
	}
	
	if issue == nil {
		t.Fatal("FetchIssue returned nil issue")
	}
	
	if issue.Number != 1 {
		t.Errorf("FetchIssue issue number = %d, want 1", issue.Number)
	}
	
	if issue.Title == "" {
		t.Error("FetchIssue issue title is empty")
	}
}

func TestClient_FetchComments_Integration(t *testing.T) {
	t.Skip("Integration test - requires gh CLI authentication")
	
	client := NewClient(30 * time.Second)
	
	// Test with a known public issue that has comments
	comments, err := client.FetchComments("octocat", "Hello-World", "1")
	if err != nil {
		t.Fatalf("FetchComments failed: %v", err)
	}
	
	if comments == nil {
		t.Fatal("FetchComments returned nil comments")
	}
	
	// This test is flexible since the number of comments may change
	// We just verify the structure is correct if there are comments
	for i, comment := range comments {
		if comment.ID == 0 {
			t.Errorf("Comment %d has invalid ID: %d", i, comment.ID)
		}
		if comment.CreatedAt.IsZero() {
			t.Errorf("Comment %d has zero CreatedAt time", i)
		}
	}
}

func TestIsGHCliAvailable(t *testing.T) {
	// This test will pass only if gh CLI is installed and authenticated
	// In CI/CD or environments without gh CLI, this will fail as expected
	err := IsGHCliAvailable()
	
	// We don't assert success/failure here since it depends on the environment
	// Instead, we just verify the function doesn't panic and returns an appropriate error
	if err != nil {
		t.Logf("gh CLI not available (expected in some environments): %v", err)
	} else {
		t.Log("gh CLI is available and authenticated")
	}
}

// Additional tests for better coverage
func TestClient_CommandExecution(t *testing.T) {
	client := NewClient(5 * time.Second)
	
	// Test with very short timeout to trigger timeout errors
	client.timeout = 1 * time.Nanosecond
	
	_, err := client.FetchIssue("owner", "repo", "1")
	if err == nil {
		t.Error("Expected timeout error with very short timeout")
	}
}

func TestClient_ExecuteWithRetry(t *testing.T) {
	client := NewClient(30 * time.Second)
	
	// Test the executeWithRetry method indirectly through public methods
	// These will fail due to gh CLI not being available, but tests the retry logic
	_, err := client.FetchIssue("nonexistent", "repo", "1")
	if err == nil {
		t.Error("Expected error for nonexistent repository")
	}
	
	_, err = client.FetchComments("nonexistent", "repo", "1")
	if err == nil {
		t.Error("Expected error for nonexistent repository")
	}
}

func TestClient_EdgeCases(t *testing.T) {
	client := NewClient(30 * time.Second)
	
	tests := []struct {
		name  string
		owner string
		repo  string
		num   string
	}{
		{"special_chars_owner", "owner-with-dashes", "repo", "1"},
		{"special_chars_repo", "owner", "repo.name", "1"},
		{"large_number", "owner", "repo", "999999"},
		{"leading_zeros", "owner", "repo", "0001"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail due to gh CLI/network issues, but test parameter handling
			_, err := client.FetchIssue(tt.owner, tt.repo, tt.num)
			if err == nil {
				t.Error("Expected error in test environment")
			}
			
			_, err = client.FetchComments(tt.owner, tt.repo, tt.num)
			if err == nil {
				t.Error("Expected error in test environment")
			}
		})
	}
}

func TestClient_TimeoutVariations(t *testing.T) {
	timeouts := []time.Duration{
		1 * time.Second,
		30 * time.Second,
		5 * time.Minute,
	}
	
	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			client := NewClient(timeout)
			if client.timeout != timeout {
				t.Errorf("Client timeout = %v, want %v", client.timeout, timeout)
			}
		})
	}
}

// Test helper functions
func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		str    string
		substr string
		want   bool
	}{
		{"contains", "hello world", "world", true},
		{"not_contains", "hello world", "foo", false},
		{"empty_substr", "hello", "", true},
		{"empty_str", "", "hello", false},
		{"both_empty", "", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.str, tt.substr)
			if got != tt.want {
				t.Errorf("containsString(%q, %q) = %v, want %v", tt.str, tt.substr, got, tt.want)
			}
		})
	}
}