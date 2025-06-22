package util

import (
	"fmt"
	"strings"
)

// ErrorType represents different types of errors in the application
type ErrorType int

const (
	// ErrorTypeGeneric represents a generic error
	ErrorTypeGeneric ErrorType = iota
	// ErrorTypeValidation represents input validation errors
	ErrorTypeValidation
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork
	// ErrorTypeFileSystem represents file system errors
	ErrorTypeFileSystem
	// ErrorTypeAuth represents authentication errors
	ErrorTypeAuth
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout
	// ErrorTypeSecurity represents security-related errors
	ErrorTypeSecurity
	// ErrorTypeClaude represents Claude integration errors
	ErrorTypeClaude
)

// AppError represents a structured application error
type AppError struct {
	Type        ErrorType
	Message     string
	Suggestion  string
	OriginalErr error
	Code        int
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.OriginalErr)
	}
	return e.Message
}

// Unwrap returns the original error
func (e *AppError) Unwrap() error {
	return e.OriginalErr
}

// String returns a formatted error message with suggestion
func (e *AppError) String() string {
	msg := e.Error()
	if e.Suggestion != "" {
		msg += "\nSuggestion: " + e.Suggestion
	}
	return msg
}

// NewAppError creates a new application error
func NewAppError(errType ErrorType, message string, originalErr error) *AppError {
	return &AppError{
		Type:        errType,
		Message:     message,
		OriginalErr: originalErr,
	}
}

// NewValidationError creates a validation error with suggestion
func NewValidationError(message, suggestion string) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Message:    message,
		Suggestion: suggestion,
		Code:       1,
	}
}

// NewNetworkError creates a network error with suggestion
func NewNetworkError(message string, originalErr error) *AppError {
	suggestion := "Check your internet connection and try again"
	
	// Add more specific suggestions based on the error type
	if originalErr != nil {
		errStr := strings.ToLower(originalErr.Error())
		if strings.Contains(errStr, "rate limit") {
			suggestion = "GitHub API rate limit exceeded. Wait a few minutes before retrying, or use a GitHub token with higher limits"
		} else if strings.Contains(errStr, "timeout") {
			suggestion = "Request timed out. Try increasing the timeout with --timeout flag or check your network connection"
		} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") {
			suggestion = "Authentication failed. Please run 'gh auth login' to authenticate with GitHub"
		} else if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
			suggestion = "Resource not found. Check that the repository and issue/PR number are correct and accessible"
		} else if strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "403") {
			suggestion = "Access forbidden. You may not have permission to access this repository or resource"
		}
	}
	
	return &AppError{
		Type:        ErrorTypeNetwork,
		Message:     message,
		Suggestion:  suggestion,
		OriginalErr: originalErr,
		Code:        2,
	}
}

// NewFileSystemError creates a file system error with suggestion
func NewFileSystemError(message string, originalErr error) *AppError {
	suggestion := "Check file permissions and available disk space"
	
	// Add more specific suggestions based on the error type
	if originalErr != nil {
		errStr := strings.ToLower(originalErr.Error())
		if strings.Contains(errStr, "permission denied") {
			suggestion = "Permission denied. Check that you have write access to the target directory or run with appropriate permissions"
		} else if strings.Contains(errStr, "no space left") {
			suggestion = "Insufficient disk space. Free up some space or choose a different output directory"
		} else if strings.Contains(errStr, "file exists") {
			suggestion = "File already exists. Use --force flag to overwrite existing files"
		} else if strings.Contains(errStr, "no such file or directory") {
			suggestion = "Directory does not exist. Create the directory first or use a valid output path"
		} else if strings.Contains(errStr, "is a directory") {
			suggestion = "Target is a directory. Specify a file path or use a different name"
		}
	}
	
	return &AppError{
		Type:        ErrorTypeFileSystem,
		Message:     message,
		Suggestion:  suggestion,
		OriginalErr: originalErr,
		Code:        3,
	}
}

// NewAuthError creates an authentication error with suggestion
func NewAuthError(message string) *AppError {
	suggestion := "Please run 'gh auth login' to authenticate with GitHub"
	return &AppError{
		Type:       ErrorTypeAuth,
		Message:    message,
		Suggestion: suggestion,
		Code:       4,
	}
}

// NewTimeoutError creates a timeout error with suggestion
func NewTimeoutError(message string) *AppError {
	suggestion := "Try increasing the timeout with --timeout flag (current default: 15s) or check your network connection. For large images, consider using --max-size to limit file sizes"
	return &AppError{
		Type:       ErrorTypeTimeout,
		Message:    message,
		Suggestion: suggestion,
		Code:       5,
	}
}

// NewSecurityError creates a security error with suggestion
func NewSecurityError(message string) *AppError {
	suggestion := "This operation was blocked for security reasons. Review the security warnings and ensure you trust the data being processed"
	return &AppError{
		Type:       ErrorTypeSecurity,
		Message:    message,
		Suggestion: suggestion,
		Code:       6,
	}
}

// NewClaudeError creates a Claude integration error with suggestion
func NewClaudeError(message string, originalErr error) *AppError {
	suggestion := "Check that Claude CLI is installed and accessible. Run 'claude --version' to verify installation"
	
	// Add more specific suggestions based on the error type
	if originalErr != nil {
		errStr := strings.ToLower(originalErr.Error())
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "command not found") {
			suggestion = "Claude CLI not found. Install it from https://claude.ai/code or remove the --send flag"
		} else if strings.Contains(errStr, "permission denied") {
			suggestion = "Permission denied accessing Claude CLI. Check that the claude command is executable"
		} else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "unauthorized") {
			suggestion = "Claude authentication failed. Run 'claude auth login' or check your API credentials"
		} else if strings.Contains(errStr, "timeout") {
			suggestion = "Claude request timed out. The images may be too large or the service may be temporarily unavailable"
		} else if strings.Contains(errStr, "rate limit") {
			suggestion = "Claude rate limit exceeded. Wait a few minutes before retrying"
		}
	}
	
	return &AppError{
		Type:        ErrorTypeClaude,
		Message:     message,
		Suggestion:  suggestion,
		OriginalErr: originalErr,
		Code:        7,
	}
}

// GetExitCode returns the appropriate exit code for an error
func GetExitCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return 1 // Generic error exit code
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeValidation
	}
	return false
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ErrorTypeAuth
	}
	return false
}