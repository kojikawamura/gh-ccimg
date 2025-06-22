package util

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelVerbose, buf)

	if logger.level != LogLevelVerbose {
		t.Errorf("Level = %v, want %v", logger.level, LogLevelVerbose)
	}

	if logger.writer != buf {
		t.Errorf("Writer = %v, want %v", logger.writer, buf)
	}

	if logger.enableTime {
		t.Error("Time should be disabled by default")
	}
}

func TestNewLogger_NilWriter(t *testing.T) {
	logger := NewLogger(LogLevelNormal, nil)

	// Should default to os.Stderr, but we can't easily test that
	// Just verify it doesn't panic
	if logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestLogger_SetTimeEnabled(t *testing.T) {
	logger := NewLogger(LogLevelNormal, &bytes.Buffer{})
	
	logger.SetTimeEnabled(true)
	if !logger.enableTime {
		t.Error("Time should be enabled")
	}

	logger.SetTimeEnabled(false)
	if logger.enableTime {
		t.Error("Time should be disabled")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{
			name:     "error in quiet mode",
			logLevel: LogLevelQuiet,
			logFunc:  func(l *Logger) { l.Error("test error") },
			shouldLog: true,
		},
		{
			name:     "info in quiet mode",
			logLevel: LogLevelQuiet,
			logFunc:  func(l *Logger) { l.Info("test info") },
			shouldLog: false,
		},
		{
			name:     "info in normal mode",
			logLevel: LogLevelNormal,
			logFunc:  func(l *Logger) { l.Info("test info") },
			shouldLog: true,
		},
		{
			name:     "verbose in normal mode",
			logLevel: LogLevelNormal,
			logFunc:  func(l *Logger) { l.Verbose("test verbose") },
			shouldLog: false,
		},
		{
			name:     "verbose in verbose mode",
			logLevel: LogLevelVerbose,
			logFunc:  func(l *Logger) { l.Verbose("test verbose") },
			shouldLog: true,
		},
		{
			name:     "debug in verbose mode",
			logLevel: LogLevelVerbose,
			logFunc:  func(l *Logger) { l.Debug("test debug") },
			shouldLog: false,
		},
		{
			name:     "debug in debug mode",
			logLevel: LogLevelDebug,
			logFunc:  func(l *Logger) { l.Debug("test debug") },
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.logLevel, buf)

			tt.logFunc(logger)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldLog {
				t.Errorf("Expected log output: %v, got output: %v (%q)", tt.shouldLog, hasOutput, output)
			}
		})
	}
}

func TestLogger_ErrorMessages(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.Error("test error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Output should contain 'ERROR', got: %q", output)
	}
	if !strings.Contains(output, "test error message") {
		t.Errorf("Output should contain error message, got: %q", output)
	}
}

func TestLogger_Success(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.Success("operation completed")

	output := buf.String()
	if !strings.Contains(output, "SUCCESS") {
		t.Errorf("Output should contain 'SUCCESS', got: %q", output)
	}
	if !strings.Contains(output, "operation completed") {
		t.Errorf("Output should contain success message, got: %q", output)
	}
}

func TestLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("Output should contain 'WARN', got: %q", output)
	}
	if !strings.Contains(output, "warning message") {
		t.Errorf("Output should contain warning message, got: %q", output)
	}
}

func TestLogger_Plain(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.Plain("plain message")

	output := buf.String()
	expected := "plain message\n"
	if output != expected {
		t.Errorf("Plain() = %q, want %q", output, expected)
	}
}

func TestLogger_Progress(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.Progress("50%% complete")

	output := buf.String()
	if !strings.Contains(output, "50% complete") {
		t.Errorf("Output should contain progress message, got: %q", output)
	}
	// Progress should use \r but not \n
	if strings.Contains(output, "\n") {
		t.Errorf("Progress should not contain newline, got: %q", output)
	}
}

func TestLogger_ProgressFinish(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)

	logger.ProgressFinish()

	output := buf.String()
	if output != "\n" {
		t.Errorf("ProgressFinish() = %q, want %q", output, "\n")
	}
}

func TestLogger_GetLevel(t *testing.T) {
	logger := NewLogger(LogLevelVerbose, &bytes.Buffer{})

	if logger.GetLevel() != LogLevelVerbose {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), LogLevelVerbose)
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger := NewLogger(LogLevelNormal, &bytes.Buffer{})

	logger.SetLevel(LogLevelDebug)

	if logger.GetLevel() != LogLevelDebug {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), LogLevelDebug)
	}
}

func TestLogger_IsVerbose(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected bool
	}{
		{LogLevelQuiet, false},
		{LogLevelNormal, false},
		{LogLevelVerbose, true},
		{LogLevelDebug, true},
	}

	for _, tt := range tests {
		logger := NewLogger(tt.level, &bytes.Buffer{})
		result := logger.IsVerbose()
		if result != tt.expected {
			t.Errorf("IsVerbose() for level %v = %v, want %v", tt.level, result, tt.expected)
		}
	}
}

func TestLogger_IsQuiet(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected bool
	}{
		{LogLevelQuiet, true},
		{LogLevelNormal, false},
		{LogLevelVerbose, false},
		{LogLevelDebug, false},
	}

	for _, tt := range tests {
		logger := NewLogger(tt.level, &bytes.Buffer{})
		result := logger.IsQuiet()
		if result != tt.expected {
			t.Errorf("IsQuiet() for level %v = %v, want %v", tt.level, result, tt.expected)
		}
	}
}

func TestLogger_WithTime(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LogLevelNormal, buf)
	logger.SetTimeEnabled(true)

	logger.Info("test message")

	output := buf.String()
	// Should contain timestamp format [HH:MM:SS]
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Output should contain timestamp brackets, got: %q", output)
	}
}

// Test package-level functions
func TestPackageLevelFunctions(t *testing.T) {
	// Save original logger
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	// Set test logger
	buf := &bytes.Buffer{}
	defaultLogger = NewLogger(LogLevelNormal, buf)

	// Test package functions
	Error("test error")
	Warn("test warn")
	Info("test info")
	Success("test success")

	output := buf.String()
	expectedMessages := []string{"ERROR", "WARN", "INFO", "SUCCESS"}
	
	for _, expected := range expectedMessages {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain %q, got: %q", expected, output)
		}
	}
}

func TestSetDefaultLogLevel(t *testing.T) {
	// Save original logger
	originalLogger := defaultLogger
	defer func() {
		defaultLogger = originalLogger
	}()

	SetDefaultLogLevel(LogLevelDebug)

	if defaultLogger.GetLevel() != LogLevelDebug {
		t.Errorf("Default log level = %v, want %v", defaultLogger.GetLevel(), LogLevelDebug)
	}
}

func TestGetDefaultLogger(t *testing.T) {
	logger := GetDefaultLogger()
	if logger != defaultLogger {
		t.Error("GetDefaultLogger should return the default logger instance")
	}
}