package util

import (
	"errors"
	"testing"
)

func TestNewAppError(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := NewAppError(ErrorTypeNetwork, "network failed", originalErr)

	if appErr.Type != ErrorTypeNetwork {
		t.Errorf("Type = %v, want %v", appErr.Type, ErrorTypeNetwork)
	}

	if appErr.Message != "network failed" {
		t.Errorf("Message = %q, want %q", appErr.Message, "network failed")
	}

	if appErr.OriginalErr != originalErr {
		t.Errorf("OriginalErr = %v, want %v", appErr.OriginalErr, originalErr)
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name        string
		appErr      *AppError
		expected    string
	}{
		{
			name: "with original error",
			appErr: &AppError{
				Message:     "something failed",
				OriginalErr: errors.New("original"),
			},
			expected: "something failed: original",
		},
		{
			name: "without original error",
			appErr: &AppError{
				Message: "something failed",
			},
			expected: "something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.appErr.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAppError_String(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name: "with suggestion",
			appErr: &AppError{
				Message:    "something failed",
				Suggestion: "try this fix",
			},
			expected: "something failed\nSuggestion: try this fix",
		},
		{
			name: "without suggestion",
			appErr: &AppError{
				Message: "something failed",
			},
			expected: "something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.appErr.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original")
	appErr := &AppError{
		OriginalErr: originalErr,
	}

	unwrapped := appErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// Test with nil original error
	appErr.OriginalErr = nil
	unwrapped = appErr.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input", "check your parameters")

	if err.Type != ErrorTypeValidation {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeValidation)
	}

	if err.Message != "invalid input" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid input")
	}

	if err.Suggestion != "check your parameters" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "check your parameters")
	}

	if err.Code != 1 {
		t.Errorf("Code = %d, want 1", err.Code)
	}
}

func TestNewNetworkError(t *testing.T) {
	originalErr := errors.New("connection failed")
	err := NewNetworkError("network error", originalErr)

	if err.Type != ErrorTypeNetwork {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeNetwork)
	}

	if err.Code != 2 {
		t.Errorf("Code = %d, want 2", err.Code)
	}

	if err.OriginalErr != originalErr {
		t.Errorf("OriginalErr = %v, want %v", err.OriginalErr, originalErr)
	}

	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty for network errors")
	}
}

func TestNewAuthError(t *testing.T) {
	err := NewAuthError("authentication failed")

	if err.Type != ErrorTypeAuth {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeAuth)
	}

	if err.Code != 4 {
		t.Errorf("Code = %d, want 4", err.Code)
	}

	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty for auth errors")
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "app error with code",
			err:      NewValidationError("test", "test"),
			expected: 1,
		},
		{
			name:     "network error",
			err:      NewNetworkError("test", nil),
			expected: 2,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetExitCode(tt.err)
			if result != tt.expected {
				t.Errorf("GetExitCode() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network error",
			err:      NewNetworkError("test", nil),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", "test"),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      NewValidationError("test", "test"),
			expected: true,
		},
		{
			name:     "network error",
			err:      NewNetworkError("test", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "auth error",
			err:      NewAuthError("test"),
			expected: true,
		},
		{
			name:     "network error",
			err:      NewNetworkError("test", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", result, tt.expected)
			}
		})
	}
}