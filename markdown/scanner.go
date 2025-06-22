package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ExtractImageURLs extracts all image URLs from markdown content
// Uses goldmark AST parser for accurate parsing
func ExtractImageURLs(content string) []string {
	if content == "" {
		return []string{}
	}

	var urls []string
	
	// Create goldmark parser
	md := goldmark.New()
	
	// Parse markdown to AST
	source := []byte(content)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)
	
	// Walk the AST to find image nodes
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		
		// Handle image nodes
		if img, ok := node.(*ast.Image); ok {
			url := string(img.Destination)
			if url != "" && isValidImageURL(url) {
				urls = append(urls, url)
			}
		}
		
		return ast.WalkContinue, nil
	})
	
	// Also try fallback regex patterns for malformed markdown
	fallbackURLs := extractWithPatterns(content)
	urls = append(urls, fallbackURLs...)
	
	// Deduplicate URLs
	return deduplicateURLs(urls)
}

// isValidImageURL checks if a URL looks like an image URL
func isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	
	// Must be a proper URL
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "data:image/") {
		return false
	}
	
	// If it starts with data: it might be a data URL
	if strings.HasPrefix(lower, "data:image/") {
		return true
	}
	
	// Check for common image extensions
	imageExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".tiff"}
	for _, ext := range imageExtensions {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	
	// Check for common image hosting patterns
	imageHosts := []string{
		"github.com/",
		"githubusercontent.com/",
		"imgur.com/",
		"i.imgur.com/",
		"imagehosting",
		"images/",
		"img/",
	}
	for _, host := range imageHosts {
		if strings.Contains(lower, host) {
			return true
		}
	}
	
	// If it's a URL without clear indicators, include it (let the downloader validate)
	// This is a permissive approach - we'll let the downloader do final validation
	return true
}

// deduplicateURLs removes duplicate URLs while preserving order
func deduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, url := range urls {
		if url == "" {
			continue
		}
		
		// Normalize URL for comparison
		normalized := strings.TrimSpace(url)
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	
	return result
}