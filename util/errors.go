package util

import (
	"fmt"
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
	suggestion := "Try increasing the timeout with --timeout flag or check your network connection"
	return &AppError{
		Type:       ErrorTypeTimeout,
		Message:    message,
		Suggestion: suggestion,
		Code:       5,
	}
}

// NewSecurityError creates a security error with suggestion
func NewSecurityError(message string) *AppError {
	suggestion := "This operation was blocked for security reasons"
	return &AppError{
		Type:       ErrorTypeSecurity,
		Message:    message,
		Suggestion: suggestion,
		Code:       6,
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