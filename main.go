package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/kojikawamura/gh-ccimg/cmd"
)

// Version information - set during build
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %v\n", r)
			fmt.Fprintf(os.Stderr, "Runtime: %s\n", runtime.Version())
			fmt.Fprintf(os.Stderr, "Please report this issue at: https://github.com/kojikawamura/gh-ccimg/issues\n")
			os.Exit(7) // Exit code 7 for panic recovery
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the command in a goroutine so we can handle signals
	errorChan := make(chan error, 1)
	go func() {
		errorChan <- cmd.Execute()
	}()

	// Wait for either command completion or signal
	select {
	case err := <-errorChan:
		if err != nil {
			// Check if it's one of our custom error types for proper exit codes
			if exitErr, ok := err.(interface{ ExitCode() int }); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1) // General error
		}
		// Success
		os.Exit(0)
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, shutting down gracefully...\n", sig)
		os.Exit(130) // 128 + SIGINT(2) = 130
	}
}

// ShowVersion displays version information
func ShowVersion() {
	fmt.Printf("gh-ccimg version %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
	fmt.Printf("Built: %s\n", BuildTime)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}
