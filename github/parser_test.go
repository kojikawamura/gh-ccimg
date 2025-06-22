package github

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOwner   string
		wantRepo    string
		wantNum     string
		wantErr     bool
		errContains string
	}{
		// Valid short form
		{
			name:      "short form valid",
			input:     "octocat/Hello-World#123",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "123",
			wantErr:   false,
		},
		// Valid issue URLs
		{
			name:      "issue URL basic",
			input:     "https://github.com/octocat/Hello-World/issues/123",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "123",
			wantErr:   false,
		},
		{
			name:      "issue URL with query params",
			input:     "https://github.com/octocat/Hello-World/issues/123?tab=comments",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "123",
			wantErr:   false,
		},
		// Valid pull request URLs
		{
			name:      "pull URL basic",
			input:     "https://github.com/octocat/Hello-World/pull/456",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "456",
			wantErr:   false,
		},
		{
			name:      "pull URL with fragment",
			input:     "https://github.com/octocat/Hello-World/pull/456#discussion_r12345",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "456",
			wantErr:   false,
		},
		// Edge cases - valid
		{
			name:      "numeric repo name",
			input:     "user/123#1",
			wantOwner: "user",
			wantRepo:  "123",
			wantNum:   "1",
			wantErr:   false,
		},
		{
			name:      "repo with dots and dashes",
			input:     "user/repo.name-test#999",
			wantOwner: "user",
			wantRepo:  "repo.name-test",
			wantNum:   "999",
			wantErr:   false,
		},
		// Invalid cases
		{
			name:        "empty input",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "missing hash",
			input:       "octocat/Hello-World123",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "missing issue number",
			input:       "octocat/Hello-World#",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "zero issue number",
			input:       "octocat/Hello-World#0",
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:        "negative issue number",
			input:       "octocat/Hello-World#-1",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "non-numeric issue number",
			input:       "octocat/Hello-World#abc",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "invalid URL domain",
			input:       "https://gitlab.com/octocat/Hello-World/issues/123",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "malformed URL",
			input:       "https://github.com/octocat",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "empty owner",
			input:       "/Hello-World#123",
			wantErr:     true,
			errContains: "invalid target format",
		},
		{
			name:        "empty repo",
			input:       "octocat/#123",
			wantErr:     true,
			errContains: "invalid target format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, num, err := ParseTarget(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTarget() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("ParseTarget() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseTarget() unexpected error = %v", err)
				return
			}
			
			if owner != tt.wantOwner {
				t.Errorf("ParseTarget() owner = %v, want %v", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("ParseTarget() repo = %v, want %v", repo, tt.wantRepo)
			}
			if num != tt.wantNum {
				t.Errorf("ParseTarget() num = %v, want %v", num, tt.wantNum)
			}
		})
	}
}

func TestParseTargetWithWhitespace(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantNum   string
	}{
		{
			input:     "  octocat/Hello-World#123  ",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "123",
		},
		{
			input:     "\t\nhttps://github.com/octocat/Hello-World/issues/123\n\t",
			wantOwner: "octocat",
			wantRepo:  "Hello-World",
			wantNum:   "123",
		},
	}

	for i, tt := range tests {
		owner, repo, num, err := ParseTarget(tt.input)
		if err != nil {
			t.Errorf("test %d: unexpected error = %v", i, err)
			continue
		}
		
		if owner != tt.wantOwner || repo != tt.wantRepo || num != tt.wantNum {
			t.Errorf("test %d: got (%v, %v, %v), want (%v, %v, %v)", 
				i, owner, repo, num, tt.wantOwner, tt.wantRepo, tt.wantNum)
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}