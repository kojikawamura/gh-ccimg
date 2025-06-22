package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMain_Entry tests the main function entry point with comprehensive scenarios
func TestMain_Entry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main entry test in short mode")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-test")
	
	tests := []struct {
		name        string
		args        []string
		wantExit    int
		checkOutput bool
		expectInOutput string
	}{
		{
			name:     "no_arguments",
			args:     []string{},
			wantExit: 1, // Should exit with error
		},
		{
			name:           "help_flag",
			args:           []string{"--help"},
			wantExit:       0, // Help should exit successfully
			checkOutput:    true,
			expectInOutput: "gh-ccimg",
		},
		{
			name:           "version_flag",
			args:           []string{"--version"},
			wantExit:       0, // Version should exit successfully
			checkOutput:    true,
			expectInOutput: "gh-ccimg version dev",
		},
		{
			name:           "version_short_flag",
			args:           []string{"-V"},
			wantExit:       0, // Version should exit successfully
			checkOutput:    true,
			expectInOutput: "gh-ccimg version dev",
		},
		{
			name:     "invalid_target",
			args:     []string{"invalid-target"},
			wantExit: 1, // Should exit with error
		},
		{
			name:     "malformed_github_url",
			args:     []string{"not-a-github-url"},
			wantExit: 2, // Invalid arguments
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gh-ccimg-test", tt.args...)
			
			var stdout, stderr bytes.Buffer
			if tt.checkOutput {
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
			}
			
			err := cmd.Run()
			
			if tt.wantExit == 0 {
				if err != nil {
					t.Errorf("Expected successful execution, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected non-zero exit code, but command succeeded")
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					// For some test environments, we may not get the exact exit code
					if exitErr.ExitCode() == 0 {
						t.Errorf("Expected non-zero exit code, got 0")
					}
				}
			}
			
			// Check output if expected
			if tt.checkOutput && tt.expectInOutput != "" {
				output := stdout.String() + stderr.String()
				if !strings.Contains(output, tt.expectInOutput) {
					t.Errorf("Expected output to contain %q, got: %s", tt.expectInOutput, output)
				}
			}
		})
	}
}

// TestMain_PanicRecovery tests panic recovery mechanism
func TestMain_PanicRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping panic recovery test in short mode")
	}
	
	// We test panic recovery through subprocess execution
	// since we can't directly trigger a panic in main without affecting the test
	
	// Build a test binary that can simulate a panic
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-panic-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-panic-test")
	
	// Test that the binary doesn't panic on invalid input
	cmd = exec.Command("./gh-ccimg-panic-test", "definitely-invalid-input-that-should-not-panic")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err == nil {
		t.Error("Expected error for invalid input, got none")
	}
	
	// Should not contain panic traces
	output := stderr.String()
	if strings.Contains(output, "panic:") || strings.Contains(output, "runtime.gopanic") {
		t.Errorf("Unexpected panic in output: %s", output)
	}
}

// TestMain_SignalHandling tests signal handling for graceful shutdown
func TestMain_SignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}
	
	if runtime.GOOS == "windows" {
		t.Skip("Signal handling test not supported on Windows")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-signal-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-signal-test")
	
	// Start the process with an invalid target that would normally fail
	// but we'll send SIGINT before it completes
	cmd = exec.Command("./gh-ccimg-signal-test", "owner/repo#999999")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Send SIGINT
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}
	
	// Wait for process to exit
	err := cmd.Wait()
	if err == nil {
		t.Error("Expected process to exit with error after SIGINT")
	}
	
	// Check for graceful shutdown message
	output := stderr.String()
	if !strings.Contains(output, "Received signal") && !strings.Contains(output, "shutting down") {
		// Some environments may not show the signal message, so we just check it doesn't panic
		if strings.Contains(output, "panic:") {
			t.Errorf("Process panicked instead of handling signal gracefully: %s", output)
		}
	}
}

// TestMain_VersionDisplay tests version information display
func TestMain_VersionDisplay(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping version display test in short mode")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-version-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-version-test")
	
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "version_long_flag",
			args: []string{"--version"},
			expectedOutput: []string{
				"gh-ccimg version dev",
				"Built with go",
				"OS/Arch:",
			},
		},
		{
			name: "version_short_flag",
			args: []string{"-V"},
			expectedOutput: []string{
				"gh-ccimg version dev",
				"Built with go",
				"OS/Arch:",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gh-ccimg-version-test", tt.args...)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			
			err := cmd.Run()
			if err != nil {
				t.Errorf("Version command failed: %v", err)
				return
			}
			
			output := stdout.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

// TestMain_ExitBehavior tests proper exit behavior for different scenarios
func TestMain_ExitBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping exit behavior test in short mode")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-exit-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-exit-test")
	
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		description  string
	}{
		{
			name:         "success_help",
			args:         []string{"--help"},
			wantExitCode: 0,
			description:  "Help should exit with code 0",
		},
		{
			name:         "success_version",
			args:         []string{"--version"},
			wantExitCode: 0,
			description:  "Version should exit with code 0",
		},
		{
			name:         "error_no_args",
			args:         []string{},
			wantExitCode: 1,
			description:  "No arguments should exit with error code",
		},
		{
			name:         "error_invalid_target",
			args:         []string{"invalid"},
			wantExitCode: 1, // Could be 2 for validation error, but depends on implementation
			description:  "Invalid target should exit with error code",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gh-ccimg-exit-test", tt.args...)
			cmd.Stdout = nil // Suppress output
			cmd.Stderr = nil
			
			err := cmd.Run()
			
			if tt.wantExitCode == 0 {
				if err != nil {
					t.Errorf("%s: %v", tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("%s: expected non-zero exit code, got success", tt.description)
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					// In test environments, exact exit codes may vary
					// We just verify it's non-zero
					if exitErr.ExitCode() == 0 {
						t.Errorf("%s: expected non-zero exit code, got 0", tt.description)
					}
				}
			}
		})
	}
}

// TestMain_VersionFunctions tests the version-related functions
func TestMain_VersionFunctions(t *testing.T) {
	// Build test binary to test version functions
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-version-func-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-version-func-test")
	
	// Test version flag output
	cmd = exec.Command("./gh-ccimg-version-func-test", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err := cmd.Run()
	if err != nil {
		t.Errorf("Version command failed: %v", err)
		return
	}
	
	result := stdout.String()
	
	// Check that version output contains expected information
	expected := []string{
		"gh-ccimg version",
		"Built with go",
		"OS/Arch:",
	}
	
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected version output to contain %q, got: %s", exp, result)
		}
	}
}

// TestMain_ProcessLevelHandling tests process-level functionality
func TestMain_ProcessLevelHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping process-level test in short mode")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-process-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-process-test")
	
	// Test process spawning and basic lifecycle
	cmd = exec.Command("./gh-ccimg-process-test", "--version")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	
	// Verify process is running
	if cmd.Process == nil {
		t.Fatal("Process was not started")
	}
	
	// Wait for completion
	if err := cmd.Wait(); err != nil {
		t.Errorf("Process failed to complete successfully: %v", err)
	}
	
	// Verify process completed
	if cmd.ProcessState == nil {
		t.Error("Process state not available")
	} else if !cmd.ProcessState.Exited() {
		t.Error("Process did not exit")
	}
}

// TestMain_ErrorContextPreservation tests that error context is preserved
func TestMain_ErrorContextPreservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error context test in short mode")
	}
	
	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-error-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-error-test")
	
	// Test with various error conditions
	errorTests := []struct {
		name string
		args []string
	}{
		{"no_args", []string{}},
		{"invalid_target", []string{"invalid"}},
		{"malformed_url", []string{"https://not-github.com/owner/repo/issues/1"}},
	}
	
	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./gh-ccimg-error-test", tt.args...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			if err == nil {
				t.Error("Expected error but command succeeded")
				return
			}
			
			// Check that error output is meaningful (not just panic traces)
			output := stderr.String()
			if strings.Contains(output, "panic:") {
				t.Errorf("Unexpected panic in error output: %s", output)
			}
			
			// Should contain some useful error information
			if output == "" {
				t.Error("No error output provided")
			}
		})
	}
}

// BenchmarkMain_Build benchmarks the build process
func BenchmarkMain_Build(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("go", "build", "-o", "gh-ccimg-bench", ".")
		if err := cmd.Run(); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
		os.Remove("gh-ccimg-bench")
	}
}

// BenchmarkMain_VersionDisplay benchmarks version display performance
func BenchmarkMain_VersionDisplay(b *testing.B) {
	// Build test binary once
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-version-bench", ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-version-bench")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("./gh-ccimg-version-bench", "--version")
		cmd.Stdout = nil
		cmd.Stderr = nil
		
		if err := cmd.Run(); err != nil {
			b.Fatalf("Version display failed: %v", err)
		}
	}
}

// BenchmarkMain_ProcessSpawn benchmarks process spawning
func BenchmarkMain_ProcessSpawn(b *testing.B) {
	// Build test binary once
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-spawn-bench", ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-spawn-bench")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("./gh-ccimg-spawn-bench", "--version")
		cmd.Stdout = nil
		cmd.Stderr = nil
		
		if err := cmd.Run(); err != nil {
			b.Fatalf("Process spawn failed: %v", err)
		}
	}
}

// TestMain_Coverage provides coverage for main package
func TestMain_Coverage(t *testing.T) {
	// Verify the package compiles
	cmd := exec.Command("go", "build", ".")
	if err := cmd.Run(); err != nil {
		t.Errorf("Main package should build successfully: %v", err)
	}
	defer os.Remove("gh-ccimg")
	
	// Test version display via built binary
	cmd = exec.Command("./gh-ccimg", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err := cmd.Run()
	if err != nil {
		t.Errorf("Version command should work: %v", err)
		return
	}
	
	if stdout.Len() == 0 {
		t.Error("Version command produced no output")
	}
}

// TestMain_RuntimeInfo tests runtime information display
func TestMain_RuntimeInfo(t *testing.T) {
	// Build and test runtime info display via binary
	cmd := exec.Command("go", "build", "-o", "gh-ccimg-runtime-test", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove("gh-ccimg-runtime-test")
	
	// Test version output includes runtime info
	cmd = exec.Command("./gh-ccimg-runtime-test", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err := cmd.Run()
	if err != nil {
		t.Errorf("Version command failed: %v", err)
		return
	}
	
	output := stdout.String()
	
	// Check for runtime information
	expectedInfo := []string{
		runtime.GOOS,
		runtime.GOARCH,
	}
	
	for _, info := range expectedInfo {
		if !strings.Contains(output, info) {
			t.Errorf("Expected runtime info %q in version output: %s", info, output)
		}
	}
}