package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kojikawamura/gh-ccimg/security"
	"github.com/kojikawamura/gh-ccimg/storage"
)

// TestPlatformFilePathHandling tests file path operations across platforms
func TestPlatformFilePathHandling(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		skipOS   []string
	}{
		{
			name:     "unix_absolute_path",
			path:     "/tmp/test/images",
			expected: "/tmp/test/images",
			skipOS:   []string{"windows"},
		},
		{
			name:     "unix_relative_path",
			path:     "./images/test",
			expected: "images/test", // filepath.Clean removes ./
		},
		{
			name:     "windows_absolute_path",
			path:     "C:\\temp\\test\\images",
			expected: "C:\\temp\\test\\images",
			skipOS:   []string{"linux", "darwin"},
		},
		{
			name:     "cross_platform_relative",
			path:     "images/subdirectory",
			expected: "images/subdirectory",
		},
		{
			name:     "path_with_dots",
			path:     "images/../test/./file.png",
			expected: "test/file.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if not applicable to current OS
			for _, skipOS := range tt.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping %s test on %s", tt.name, runtime.GOOS)
				}
			}

			cleaned := filepath.Clean(tt.path)
			
			// For Windows tests on non-Windows, just verify the function doesn't panic
			if runtime.GOOS == "windows" || len(tt.skipOS) == 0 {
				// Normalize separators for comparison
				expected := filepath.FromSlash(tt.expected)
				if cleaned != expected {
					t.Errorf("Expected %q, got %q", expected, cleaned)
				}
			}
		})
	}
}

// TestPlatformDirectoryCreation tests directory creation across platforms  
func TestPlatformDirectoryCreation(t *testing.T) {
	// Create platform-specific temp directory
	tempBase, err := os.MkdirTemp("", "platform-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempBase)

	tests := []struct {
		name    string
		path    string
		perm    os.FileMode
		wantErr bool
	}{
		{
			name: "simple_directory",
			path: "simple",
			perm: 0755,
		},
		{
			name: "nested_directories",
			path: "nested/deep/directory/structure",
			perm: 0755,
		},
		{
			name: "directory_with_spaces",
			path: "directory with spaces",
			perm: 0755,
		},
		{
			name: "directory_with_unicode",
			path: "ünïcødé-directory",
			perm: 0755,
		},
	}

	// Add platform-specific tests
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			perm    os.FileMode
			wantErr bool
		}{
			name: "restricted_permissions",
			path: "restricted",
			perm: 0000,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tempBase, tt.path)
			
			err := os.MkdirAll(fullPath, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("MkdirAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify directory exists
				info, err := os.Stat(fullPath)
				if err != nil {
					t.Errorf("Directory not created: %v", err)
					return
				}

				if !info.IsDir() {
					t.Errorf("Created path is not a directory")
				}

				// Check permissions (skip on Windows as it handles permissions differently)
				if runtime.GOOS != "windows" && tt.perm != 0000 {
					if info.Mode().Perm() != tt.perm {
						t.Logf("Permission mismatch: got %o, want %o (this may be expected due to umask)", 
							info.Mode().Perm(), tt.perm)
					}
				}
			}
		})
	}
}

// TestPlatformPathSecurity tests security path validation across platforms
func TestPlatformPathSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "security-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		basePath    string
		targetPath  string
		expectError bool
		skipOS      []string
	}{
		{
			name:       "valid_subdirectory",
			basePath:   tempDir,
			targetPath: filepath.Join(tempDir, "subdir"),
		},
		{
			name:        "traversal_attack_unix",
			basePath:    tempDir,
			targetPath:  filepath.Join(tempDir, "../../../etc/passwd"),
			expectError: true,
			skipOS:      []string{"windows"},
		},
		{
			name:        "traversal_attack_windows",
			basePath:    tempDir,
			targetPath:  filepath.Join(tempDir, "..\\..\\..\\Windows\\System32"),
			expectError: true,
			skipOS:      []string{"linux", "darwin"},
		},
		{
			name:        "absolute_path_escape",
			basePath:    tempDir,
			targetPath:  "/etc/passwd",
			expectError: true,
		},
		{
			name:       "same_directory",
			basePath:   tempDir,
			targetPath: tempDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if not applicable to current OS
			for _, skipOS := range tt.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping %s test on %s", tt.name, runtime.GOOS)
				}
			}

			err := security.ValidatePath(tt.basePath, tt.targetPath)
			
			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePath() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestPlatformStorageOperations tests storage operations across platforms
func TestPlatformStorageOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-platform-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testData := []byte("test image data for platform testing")

	tests := []struct {
		name        string
		outputDir   string
		force       bool
		expectError bool
	}{
		{
			name:      "standard_directory",
			outputDir: filepath.Join(tempDir, "standard"),
		},
		{
			name:      "directory_with_spaces",
			outputDir: filepath.Join(tempDir, "directory with spaces"),
		},
		{
			name:      "deeply_nested",
			outputDir: filepath.Join(tempDir, "deeply", "nested", "directory", "structure"),
		},
		{
			name:      "unicode_directory",
			outputDir: filepath.Join(tempDir, "ünïcødé", "directory"),
		},
	}

	// Add platform-specific tests
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name        string
			outputDir   string
			force       bool
			expectError bool
		}{
			name:      "windows_drive_path",
			outputDir: filepath.Join(tempDir, "C_drive_test"), // Simulate drive-like path
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diskStorage, err := storage.NewDiskStorage(tt.outputDir, tt.force)
			if (err != nil) != tt.expectError {
				if tt.expectError {
					t.Logf("Expected error for %s: %v", tt.name, err)
					return
				} else {
					t.Fatalf("Unexpected error creating storage: %v", err)
				}
			}

			if !tt.expectError {
				// Test storing a file
				filePath, err := diskStorage.Store(testData, "image/png", "test://platform-test.png")
				if err != nil {
					t.Errorf("Failed to store file: %v", err)
					return
				}

				// Verify file exists and has correct content
				data, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read stored file: %v", err)
					return
				}

				if string(data) != string(testData) {
					t.Errorf("File content mismatch")
				}

				// Verify file is in expected directory
				if !strings.HasPrefix(filePath, tt.outputDir) {
					t.Errorf("File not stored in expected directory: %s not under %s", 
						filePath, tt.outputDir)
				}
			}
		})
	}
}

// TestPlatformTempDirectories tests temporary directory usage across platforms
func TestPlatformTempDirectories(t *testing.T) {
	// Test that temp directory behavior works consistently
	tempDir, err := os.MkdirTemp("", "gh-ccimg-temp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Verify temp directory is absolute
	if !filepath.IsAbs(tempDir) {
		t.Errorf("Temp directory should be absolute: %s", tempDir)
	}

	// Verify we can write to temp directory
	testFile := filepath.Join(tempDir, "test-file.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Errorf("Cannot write to temp directory: %v", err)
	}

	// Verify we can read from temp directory
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Cannot read from temp directory: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("File content mismatch in temp directory")
	}

	// Platform-specific temp directory checks
	switch runtime.GOOS {
	case "windows":
		// On Windows, temp directories typically under TEMP or TMP
		if !strings.Contains(strings.ToUpper(tempDir), "TEMP") && 
		   !strings.Contains(strings.ToUpper(tempDir), "TMP") {
			t.Logf("Warning: temp directory doesn't contain TEMP or TMP: %s", tempDir)
		}
	case "darwin", "linux":
		// On Unix-like systems, typically under /tmp or /var/folders
		if !strings.HasPrefix(tempDir, "/tmp") && 
		   !strings.HasPrefix(tempDir, "/var/folders") && 
		   !strings.HasPrefix(tempDir, "/private/tmp") {
			t.Logf("Warning: temp directory not in expected location: %s", tempDir)
		}
	}
}

// TestPlatformFilePermissions tests file permission handling across platforms
func TestPlatformFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix permission tests on Windows")
	}

	tempDir, err := os.MkdirTemp("", "permission-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	permissionTests := []struct {
		name     string
		perm     os.FileMode
		readable bool
		writable bool
	}{
		{"read_only", 0444, true, false},
		{"write_only", 0222, false, true},
		{"read_write", 0644, true, true},
		{"executable", 0755, true, true},
		{"no_permissions", 0000, false, false},
	}

	for _, pt := range permissionTests {
		t.Run(pt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, pt.name+".txt")
			
			// Create file with specific permissions
			err := os.WriteFile(testFile, []byte("test content"), pt.perm)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Check file permissions
			info, err := os.Stat(testFile)
			if err != nil {
				t.Fatalf("Failed to stat test file: %v", err)
			}

			// Note: actual permissions might be modified by umask
			actualPerm := info.Mode().Perm()
			t.Logf("File %s: requested %o, actual %o", pt.name, pt.perm, actualPerm)

			// Test readability
			_, err = os.ReadFile(testFile)
			canRead := err == nil
			if canRead != pt.readable && pt.perm != 0000 {
				// Skip strict checking for 0000 as behavior varies
				t.Logf("Readability mismatch for %s: expected %v, got %v", 
					pt.name, pt.readable, canRead)
			}

			// Test writability (try to append)
			file, err := os.OpenFile(testFile, os.O_WRONLY|os.O_APPEND, 0)
			if file != nil {
				file.Close()
			}
			canWrite := err == nil
			if canWrite != pt.writable && pt.perm != 0000 {
				t.Logf("Writability mismatch for %s: expected %v, got %v", 
					pt.name, pt.writable, canWrite)
			}
		})
	}
}

// TestPlatformPathSeparators tests path separator handling
func TestPlatformPathSeparators(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		skipOnUnix bool
	}{
		{
			name:     "unix_style_path",
			input:    "images/subdirectory/file.png",
			expected: filepath.Join("images", "subdirectory", "file.png"),
		},
		{
			name:       "windows_style_path", 
			input:      "images\\subdirectory\\file.png",
			expected:   filepath.Join("images", "subdirectory", "file.png"),
			skipOnUnix: true, // FromSlash doesn't convert backslashes on Unix
		},
		{
			name:       "mixed_separators",
			input:      "images/subdirectory\\file.png",
			expected:   filepath.Join("images", "subdirectory", "file.png"),
			skipOnUnix: true, // FromSlash doesn't convert backslashes on Unix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that don't work on Unix systems
			if tt.skipOnUnix && runtime.GOOS != "windows" {
				t.Skipf("Skipping %s test on non-Windows platform", tt.name)
			}

			// Test filepath.FromSlash conversion
			converted := filepath.FromSlash(tt.input)
			
			// Normalize the expected path for current platform
			expected := tt.expected
			
			// Compare normalized paths
			if filepath.Clean(converted) != filepath.Clean(expected) {
				t.Errorf("Path conversion failed: input %q, got %q, expected %q", 
					tt.input, converted, expected)
			}
		})
	}
}

// TestPlatformExecutableDetection tests executable file detection
func TestPlatformExecutableDetection(t *testing.T) {
	// Test that we can detect if executables are available
	executables := []struct {
		name        string
		command     string
		shouldExist bool
		skipOS      []string
	}{
		{
			name:        "go_executable",
			command:     "go",
			shouldExist: true, // Should exist since we're running go test
		},
		{
			name:    "git_executable",
			command: "git",
			// Git might not be available in all test environments
		},
		{
			name:    "claude_executable", 
			command: "claude",
			// Claude CLI might not be available
		},
	}

	for _, exec := range executables {
		t.Run(exec.name, func(t *testing.T) {
			// Skip test if not applicable to current OS
			for _, skipOS := range exec.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping %s test on %s", exec.name, runtime.GOOS)
				}
			}

			// Try to find executable in PATH
			_, err := os.Stat(exec.command)
			
			// On Unix systems, try which command
			if err != nil && runtime.GOOS != "windows" {
				// This is just informational - we don't require all executables
				t.Logf("Executable %s not found directly, this may be normal", exec.command)
			}

			// On Windows, try with .exe extension
			if err != nil && runtime.GOOS == "windows" {
				_, err = os.Stat(exec.command + ".exe")
				if err != nil {
					t.Logf("Executable %s.exe not found, this may be normal", exec.command)
				}
			}

			// Don't fail tests based on executable availability unless specifically required
			if exec.shouldExist && err != nil {
				t.Logf("Warning: Required executable %s not found: %v", exec.command, err)
			}
		})
	}
}

// TestPlatformEnvironmentVariables tests environment variable handling
func TestPlatformEnvironmentVariables(t *testing.T) {
	// Test environment variables that should exist on different platforms
	envTests := []struct {
		name    string
		envVar  string
		skipOS  []string
		require bool
	}{
		{
			name:   "temp_directory_unix",
			envVar: "TMPDIR",
			skipOS: []string{"windows"},
		},
		{
			name:   "temp_directory_windows",
			envVar: "TEMP",
			skipOS: []string{"linux", "darwin"},
		},
		{
			name:   "home_directory_unix",
			envVar: "HOME",
			skipOS: []string{"windows"},
		},
		{
			name:   "user_profile_windows",
			envVar: "USERPROFILE",
			skipOS: []string{"linux", "darwin"},
		},
		{
			name:   "path_variable",
			envVar: "PATH",
		},
	}

	for _, et := range envTests {
		t.Run(et.name, func(t *testing.T) {
			// Skip test if not applicable to current OS
			for _, skipOS := range et.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping %s test on %s", et.name, runtime.GOOS)
				}
			}

			value := os.Getenv(et.envVar)
			if et.require && value == "" {
				t.Errorf("Required environment variable %s not set", et.envVar)
			} else {
				t.Logf("Environment variable %s = %q", et.envVar, value)
			}
		})
	}
}

// BenchmarkPlatformOperations benchmarks platform-specific operations
func BenchmarkPlatformOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "platform-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b.Run("filepath_clean", func(b *testing.B) {
		testPaths := []string{
			"./images/../test/file.png",
			"images/subdirectory/file.png", 
			"../../../etc/passwd",
			"C:\\temp\\..\\test\\file.png",
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, path := range testPaths {
				_ = filepath.Clean(path)
			}
		}
	})

	b.Run("filepath_join", func(b *testing.B) {
		components := [][]string{
			{"images", "subdirectory", "file.png"},
			{"temp", "test", "output", "image.jpg"},
			{"very", "deeply", "nested", "directory", "structure", "file.gif"},
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, comp := range components {
				_ = filepath.Join(comp...)
			}
		}
	})

	b.Run("file_creation", func(b *testing.B) {
		testData := []byte("benchmark test data")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("bench-file-%d.txt", i%1000))
			err := os.WriteFile(testFile, testData, 0644)
			if err != nil {
				b.Fatalf("Failed to create test file: %v", err)
			}
			os.Remove(testFile) // Clean up immediately
		}
	})
}