package util

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// LogLevelQuiet suppresses all output except errors
	LogLevelQuiet LogLevel = iota
	// LogLevelNormal shows standard output
	LogLevelNormal
	// LogLevelVerbose shows detailed output
	LogLevelVerbose
	// LogLevelDebug shows debug information
	LogLevelDebug
)

// Logger handles colorized stderr output with different log levels
type Logger struct {
	level      LogLevel
	writer     io.Writer
	enableTime bool
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel, writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stderr
	}
	return &Logger{
		level:      level,
		writer:     writer,
		enableTime: false,
	}
}

// SetTimeEnabled enables or disables timestamps in log output
func (l *Logger) SetTimeEnabled(enabled bool) {
	l.enableTime = enabled
}

// Error logs an error message (always shown unless quiet)
func (l *Logger) Error(format string, args ...interface{}) {
	l.logWithColor("ERROR", "\033[31m", format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		l.logWithColor("WARN", "\033[33m", format, args...)
	}
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		l.logWithColor("INFO", "\033[36m", format, args...)
	}
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		l.logWithColor("SUCCESS", "\033[32m", format, args...)
	}
}

// Verbose logs a verbose message
func (l *Logger) Verbose(format string, args ...interface{}) {
	if l.level >= LogLevelVerbose {
		l.logWithColor("VERBOSE", "\033[37m", format, args...)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level >= LogLevelDebug {
		l.logWithColor("DEBUG", "\033[35m", format, args...)
	}
}

// Progress logs a progress message (overwrites previous line)
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "\r%s", message)
	}
}

// ProgressFinish finishes a progress line with a newline
func (l *Logger) ProgressFinish() {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, "\n")
	}
}

// logWithColor logs a message with the specified color
func (l *Logger) logWithColor(level, color, format string, args ...interface{}) {
	if l.level == LogLevelQuiet && level != "ERROR" {
		return
	}

	message := fmt.Sprintf(format, args...)
	
	var prefix string
	if l.enableTime {
		timestamp := time.Now().Format("15:04:05")
		prefix = fmt.Sprintf("[%s] ", timestamp)
	}

	// Check if output supports colors (simple check for stderr)
	if l.writer == os.Stderr {
		fmt.Fprintf(l.writer, "%s%s%s%s\033[0m\n", prefix, color, level, message)
	} else {
		fmt.Fprintf(l.writer, "%s%s: %s\n", prefix, level, message)
	}
}

// Plain logs a message without any formatting or level
func (l *Logger) Plain(format string, args ...interface{}) {
	if l.level >= LogLevelNormal {
		fmt.Fprintf(l.writer, format+"\n", args...)
	}
}

// ErrorPlain logs an error message without formatting
func (l *Logger) ErrorPlain(format string, args ...interface{}) {
	fmt.Fprintf(l.writer, format+"\n", args...)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// IsVerbose returns true if verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.level >= LogLevelVerbose
}

// IsQuiet returns true if quiet mode is enabled
func (l *Logger) IsQuiet() bool {
	return l.level == LogLevelQuiet
}

// Global logger instance
var defaultLogger = NewLogger(LogLevelNormal, os.Stderr)

// Package-level convenience functions

// Error logs an error using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Warn logs a warning using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Info logs info using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Success logs success using the default logger
func Success(format string, args ...interface{}) {
	defaultLogger.Success(format, args...)
}

// Verbose logs verbose using the default logger
func Verbose(format string, args ...interface{}) {
	defaultLogger.Verbose(format, args...)
}

// Debug logs debug using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// SetDefaultLogLevel sets the default logger level
func SetDefaultLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// GetDefaultLogger returns the default logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}