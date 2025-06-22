package github

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Match patterns for different GitHub URL formats
	shortFormRegex = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9\-]{0,38})/([a-zA-Z0-9._\-]+)#(\d+)$`)
	issueURLRegex  = regexp.MustCompile(`^https://github\.com/([a-zA-Z0-9][a-zA-Z0-9\-]{0,38})/([a-zA-Z0-9._\-]+)/issues/(\d+)(?:[/?#].*)?$`)
	pullURLRegex   = regexp.MustCompile(`^https://github\.com/([a-zA-Z0-9][a-zA-Z0-9\-]{0,38})/([a-zA-Z0-9._\-]+)/pull/(\d+)(?:[/?#].*)?$`)
)

// ParseTarget parses a GitHub target into owner, repo, and issue/PR number.
// Supports three formats:
//   - OWNER/REPO#NUM
//   - https://github.com/OWNER/REPO/issues/NUM
//   - https://github.com/OWNER/REPO/pull/NUM
func ParseTarget(input string) (owner, repo, num string, err error) {
	if input == "" {
		return "", "", "", fmt.Errorf("target cannot be empty")
	}

	input = strings.TrimSpace(input)

	// Try short form: OWNER/REPO#NUM
	if matches := shortFormRegex.FindStringSubmatch(input); matches != nil {
		owner = matches[1]
		repo = matches[2]
		num = matches[3]
		
		if err := validateComponents(owner, repo, num); err != nil {
			return "", "", "", err
		}
		return owner, repo, num, nil
	}

	// Try issue URL: https://github.com/OWNER/REPO/issues/NUM
	if matches := issueURLRegex.FindStringSubmatch(input); matches != nil {
		owner = matches[1]
		repo = matches[2]
		num = matches[3]
		
		if err := validateComponents(owner, repo, num); err != nil {
			return "", "", "", err
		}
		return owner, repo, num, nil
	}

	// Try pull request URL: https://github.com/OWNER/REPO/pull/NUM
	if matches := pullURLRegex.FindStringSubmatch(input); matches != nil {
		owner = matches[1]
		repo = matches[2]
		num = matches[3]
		
		if err := validateComponents(owner, repo, num); err != nil {
			return "", "", "", err
		}
		return owner, repo, num, nil
	}

	return "", "", "", fmt.Errorf("invalid target format. Expected:\n  - OWNER/REPO#NUM\n  - https://github.com/OWNER/REPO/issues/NUM\n  - https://github.com/OWNER/REPO/pull/NUM\nGot: %s", input)
}

// validateComponents performs additional validation on parsed components
func validateComponents(owner, repo, num string) error {
	if owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	
	// Validate issue/PR number
	n, err := strconv.Atoi(num)
	if err != nil {
		return fmt.Errorf("invalid issue/PR number: %s", num)
	}
	if n <= 0 {
		return fmt.Errorf("issue/PR number must be positive, got: %d", n)
	}
	
	// Additional GitHub username/org validation
	if len(owner) > 39 {
		return fmt.Errorf("owner name too long (max 39 characters): %s", owner)
	}
	if len(repo) > 100 {
		return fmt.Errorf("repository name too long (max 100 characters): %s", repo)
	}
	
	return nil
}