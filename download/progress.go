package download

import (
	"fmt"
	"io"
	"time"
)

// Reporter interface for progress reporting
type Reporter interface {
	Start(total int)
	Update(completed int, url string, success bool, err error)
	Finish()
}

// ConsoleReporter implements console-based progress reporting
type ConsoleReporter struct {
	writer  io.Writer
	verbose bool
	total   int
	start   time.Time
}

// NewConsoleReporter creates a new console reporter
func NewConsoleReporter(writer io.Writer, verbose bool) *ConsoleReporter {
	return &ConsoleReporter{
		writer:  writer,
		verbose: verbose,
	}
}

// Start initializes the progress reporting
func (r *ConsoleReporter) Start(total int) {
	r.total = total
	r.start = time.Now()
	
	if r.verbose {
		fmt.Fprintf(r.writer, "Starting download of %d images...\n", total)
	} else if total > 1 {
		fmt.Fprintf(r.writer, "Downloading %d images...\n", total)
	}
}

// Update reports progress for a single URL
func (r *ConsoleReporter) Update(completed int, url string, success bool, err error) {
	if r.verbose {
		if success {
			fmt.Fprintf(r.writer, "✓ [%d/%d] Downloaded: %s\n", completed, r.total, url)
		} else {
			fmt.Fprintf(r.writer, "✗ [%d/%d] Failed: %s - %v\n", completed, r.total, url, err)
		}
	} else {
		// Simple progress for non-verbose mode
		if r.total > 1 {
			fmt.Fprintf(r.writer, "Progress: %d/%d\r", completed, r.total)
		}
	}
}

// Finish completes the progress reporting
func (r *ConsoleReporter) Finish() {
	duration := time.Since(r.start)
	
	if r.verbose {
		fmt.Fprintf(r.writer, "Download completed in %v\n", duration.Round(time.Millisecond))
	} else if r.total > 1 {
		fmt.Fprintf(r.writer, "\nCompleted in %v\n", duration.Round(time.Millisecond))
	}
}

// NoOpReporter is a reporter that does nothing (for testing)
type NoOpReporter struct{}

// NewNoOpReporter creates a no-op reporter
func NewNoOpReporter() *NoOpReporter {
	return &NoOpReporter{}
}

// Start does nothing
func (r *NoOpReporter) Start(total int) {}

// Update does nothing
func (r *NoOpReporter) Update(completed int, url string, success bool, err error) {}

// Finish does nothing
func (r *NoOpReporter) Finish() {}