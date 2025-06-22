package claude

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecuteClaude executes the Claude CLI with the provided prompt and images
func ExecuteClaude(prompt string, images []string, continueFlag bool) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Build command arguments safely
	args := []string{}
	
	// Add continue flag if specified
	if continueFlag {
		args = append(args, "--continue")
	}

	// Add the prompt
	args = append(args, prompt)

	// Add images - support both base64 and file paths
	for _, image := range images {
		if image == "" {
			continue // Skip empty images
		}
		args = append(args, image)
	}

	// Execute claude command using exec.Command (no shell execution)
	cmd := exec.Command("claude", args...)
	
	// Set up output to go to stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute the command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("claude command failed with exit code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute claude command: %w", err)
	}

	return nil
}

// IsClaudeAvailable checks if the Claude CLI is available
func IsClaudeAvailable() error {
	// Check if claude command exists
	cmd := exec.Command("claude", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not found. Please install Claude CLI or check that it's in your PATH")
	}
	return nil
}

// BuildClaudeArgs builds the argument list for claude command
// This is useful for testing and validation
func BuildClaudeArgs(prompt string, images []string, continueFlag bool) []string {
	args := []string{}
	
	// Add continue flag if specified
	if continueFlag {
		args = append(args, "--continue")
	}

	// Add the prompt
	if prompt != "" {
		args = append(args, prompt)
	}

	// Add images - support both base64 and file paths
	for _, image := range images {
		if image != "" {
			args = append(args, image)
		}
	}

	return args
}

// ValidateClaudeInput validates the input parameters before execution
func ValidateClaudeInput(prompt string, images []string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	if len(images) == 0 {
		return fmt.Errorf("at least one image is required")
	}

	// Check for suspicious content in prompt (basic safety check)
	// Look for patterns that are more likely to be shell injection attempts
	suspicious := []string{
		"rm -rf",
		"sudo ",
		"eval(",
		"exec(",
		"$(", // command substitution
		"`",  // backtick command substitution
	}

	lowerPrompt := strings.ToLower(prompt)
	for _, sus := range suspicious {
		if strings.Contains(lowerPrompt, sus) {
			return fmt.Errorf("prompt contains potentially dangerous content: %s", sus)
		}
	}

	return nil
}

// SanitizePrompt performs basic sanitization on the prompt
func SanitizePrompt(prompt string) string {
	if prompt == "" {
		return ""
	}

	// Remove null bytes
	prompt = strings.ReplaceAll(prompt, "\x00", "")
	
	// Trim whitespace
	prompt = strings.TrimSpace(prompt)
	
	return prompt
}