package markdown

import (
	"regexp"
	"strings"
)

var (
	// Regex patterns for fallback image URL extraction
	// These handle cases where markdown might be malformed or goldmark misses something
	
	// Standard markdown image pattern: ![alt](url)
	markdownImageRegex = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
	
	// HTML img tag pattern: <img src="url">
	htmlImgRegex = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	
	// Reference-style markdown images: [alt]: url
	referenceRegex = regexp.MustCompile(`^\s*\[[^\]]+\]:\s*([^\s]+)`)
	
	// GitHub-specific patterns
	// GitHub attachment URLs: https://github.com/user/repo/assets/...
	githubAssetRegex = regexp.MustCompile(`https://github\.com/[^/]+/[^/]+/assets/[^\s)]+`)
	
	// GitHub user content URLs: https://user-images.githubusercontent.com/...
	githubUserContentRegex = regexp.MustCompile(`https://[^/]*githubusercontent\.com/[^\s)]+`)
	
	// General HTTP(S) URLs that might be images
	// Only match URLs that contain common image indicators
	httpImageRegex = regexp.MustCompile(`https?://[^\s)]+(?:\.(?:png|jpg|jpeg|gif|webp|svg|bmp|tiff)|/(?:images?|img|assets|uploads)/[^\s)]+)`)
)

// extractWithPatterns uses regex patterns to extract image URLs as fallback
func extractWithPatterns(content string) []string {
	var urls []string
	
	// Extract using each pattern
	patterns := []*regexp.Regexp{
		markdownImageRegex,
		htmlImgRegex,
		githubAssetRegex,
		githubUserContentRegex,
		httpImageRegex,
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				url := strings.TrimSpace(match[1])
				if url != "" && isValidImageURL(url) {
					urls = append(urls, url)
				}
			} else if len(match) > 0 {
				// For patterns that capture the whole URL (like githubAssetRegex)
				url := strings.TrimSpace(match[0])
				if url != "" && isValidImageURL(url) {
					urls = append(urls, url)
				}
			}
		}
	}
	
	// Handle reference-style markdown
	// First pass: collect reference definitions
	references := extractReferences(content)
	
	// Second pass: find reference usages and resolve them
	refUsageRegex := regexp.MustCompile(`!\[[^\]]*\]\[([^\]]+)\]`)
	refMatches := refUsageRegex.FindAllStringSubmatch(content, -1)
	for _, match := range refMatches {
		if len(match) > 1 {
			refKey := strings.ToLower(strings.TrimSpace(match[1]))
			if url, exists := references[refKey]; exists && isValidImageURL(url) {
				urls = append(urls, url)
			}
		}
	}
	
	return urls
}

// extractReferences extracts reference-style markdown definitions
func extractReferences(content string) map[string]string {
	references := make(map[string]string)
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := referenceRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			// Extract reference key and URL
			parts := strings.SplitN(line, "]:", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(parts[0], "[")))
				url := strings.TrimSpace(parts[1])
				
				// Remove optional title from URL
				if spaceIdx := strings.Index(url, " "); spaceIdx > 0 {
					url = url[:spaceIdx]
				}
				if tabIdx := strings.Index(url, "\t"); tabIdx > 0 {
					url = url[:tabIdx]
				}
				
				// Remove quotes if present
				url = strings.Trim(url, `"'`)
				
				if url != "" {
					references[key] = url
				}
			}
		}
	}
	
	return references
}