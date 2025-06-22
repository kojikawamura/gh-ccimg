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