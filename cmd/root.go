package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kojikawamura/gh-ccimg/claude"
	"github.com/kojikawamura/gh-ccimg/download"
	"github.com/kojikawamura/gh-ccimg/github"
	"github.com/kojikawamura/gh-ccimg/markdown"
	"github.com/kojikawamura/gh-ccimg/security"
	"github.com/kojikawamura/gh-ccimg/storage"
	"github.com/kojikawamura/gh-ccimg/util"
)

var (
	outDir      string
	sendPrompt  string
	continueCmd bool
	maxSize     int64
	timeout     int
	force       bool
	verbose     bool
	quiet       bool
)

var rootCmd = &cobra.Command{
	Use:   "gh-ccimg <issue_url_or_target>",
	Short: "Extract images from GitHub issues and pull requests",
	Long: `gh-ccimg extracts all images from GitHub issues and pull requests,
with optional direct integration to Claude Code for AI-powered analysis.

Examples:
  gh-ccimg OWNER/REPO#123
  gh-ccimg https://github.com/OWNER/REPO/issues/123
  gh-ccimg OWNER/REPO#123 --out ./images
  gh-ccimg OWNER/REPO#123 --send "Analyze these screenshots"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up logging based on flags
		setupLogging()
		
		target := args[0]
		util.Info("Processing target: %s", target)
		
		// Step 1: Parse target
		util.Verbose("Parsing target URL/string...")
		owner, repo, num, err := github.ParseTarget(target)
		if err != nil {
			return util.NewValidationError(fmt.Sprintf("Invalid target format: %s", target), 
				"Use format: OWNER/REPO#NUM or https://github.com/OWNER/REPO/issues/NUM")
		}
		util.Verbose("Parsed: %s/%s#%s", owner, repo, num)

		// Step 2: Check prerequisites
		if err := checkPrerequisites(); err != nil {
			return err
		}

		// Step 3: Fetch GitHub data
		util.Info("Fetching GitHub data...")
		client := github.NewClient(time.Duration(timeout) * time.Second)
		
		issue, err := client.FetchIssue(owner, repo, num)
		if err != nil {
			return util.NewNetworkError("Failed to fetch issue/PR", err)
		}
		
		comments, err := client.FetchComments(owner, repo, num)
		if err != nil {
			return util.NewNetworkError("Failed to fetch comments", err)
		}
		util.Verbose("Fetched issue and %d comments", len(comments))

		// Step 4: Extract image URLs
		util.Info("Extracting image URLs from markdown...")
		var allURLs []string
		
		// From issue body
		issueURLs := markdown.ExtractImageURLs(issue.Body)
		allURLs = append(allURLs, issueURLs...)
		
		// From comments
		for _, comment := range comments {
			commentURLs := markdown.ExtractImageURLs(comment.Body)
			allURLs = append(allURLs, commentURLs...)
		}
		
		if len(allURLs) == 0 {
			util.Warn("No images found in issue/PR %s/%s#%s", owner, repo, num)
			return nil
		}
		util.Success("Found %d image URLs", len(allURLs))

		// Step 5: Download images
		util.Info("Downloading images...")
		maxSizeBytes := maxSize * 1024 * 1024 // Convert MB to bytes
		fetcher := download.NewFetcher(maxSizeBytes, time.Duration(timeout)*time.Second, 5)
		
		// Set up progress reporting
		if verbose {
			reporter := download.NewConsoleReporter(os.Stderr, true)
			fetcher.SetReporter(reporter)
		} else if !quiet {
			reporter := download.NewConsoleReporter(os.Stderr, false)
			fetcher.SetReporter(reporter)
		}
		
		ctx := context.Background()
		results := fetcher.FetchConcurrent(ctx, allURLs)
		
		// Count successful downloads
		successCount := 0
		var successfulResults []download.Result
		for _, result := range results {
			if result.Error == nil {
				successCount++
				successfulResults = append(successfulResults, result)
			} else {
				util.Verbose("Failed to download %s: %v", result.URL, result.Error)
			}
		}
		
		if successCount == 0 {
			return util.NewValidationError("No images could be downloaded", 
				"Check that the URLs are accessible and contain valid images")
		}
		util.Success("Downloaded %d/%d images successfully", successCount, len(allURLs))

		// Step 6: Store images
		var imageData []string
		if outDir != "" {
			// Disk storage mode
			util.Info("Saving images to disk...")
			if err := security.ValidateOutputPath(".", outDir); err != nil {
				return util.NewSecurityError(fmt.Sprintf("Invalid output directory: %v", err))
			}
			
			diskStorage, err := storage.NewDiskStorage(outDir, force)
			if err != nil {
				return util.NewFileSystemError("Failed to initialize disk storage", err)
			}
			
			for _, result := range successfulResults {
				filePath, err := diskStorage.Store(result.Data, result.ContentType, result.URL)
				if err != nil {
					util.Warn("Failed to save %s: %v", result.URL, err)
					continue
				}
				imageData = append(imageData, filePath)
				util.Verbose("Saved %s", filePath)
			}
			
			util.Success("Saved %d images to %s", len(imageData), outDir)
		} else {
			// Memory storage mode
			util.Info("Encoding images to base64...")
			memStorage := storage.NewMemoryStorage()
			
			for _, result := range successfulResults {
				encoded, err := memStorage.Store(result.Data, result.ContentType, result.URL)
				if err != nil {
					util.Warn("Failed to encode %s: %v", result.URL, err)
					continue
				}
				imageData = append(imageData, encoded)
			}
			
			// Output base64 strings
			for i, encoded := range imageData {
				fmt.Printf("Image %d (base64): %s\n", i+1, encoded)
			}
			util.Success("Encoded %d images to base64", len(imageData))
		}

		// Step 7: Claude integration (if requested)
		if sendPrompt != "" {
			util.Info("Sending to Claude...")
			
			// Validate Claude integration
			if err := claude.IsClaudeAvailable(); err != nil {
				return util.NewValidationError("Claude CLI not available", 
					"Install Claude CLI or remove --send flag")
			}
			
			if err := claude.ValidateClaudeInput(sendPrompt, imageData); err != nil {
				return util.NewValidationError(fmt.Sprintf("Invalid Claude input: %v", err), 
					"Check your prompt and ensure images were downloaded")
			}
			
			// Execute Claude
			sanitizedPrompt := claude.SanitizePrompt(sendPrompt)
			if err := claude.ExecuteClaude(sanitizedPrompt, imageData, continueCmd); err != nil {
				return util.NewAppError(util.ErrorTypeGeneric, "Claude execution failed", err)
			}
			
			util.Success("Claude analysis complete")
		}

		util.Success("Operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory for images (default: memory mode)")
	rootCmd.Flags().StringVar(&sendPrompt, "send", "", "Send images to Claude with this prompt")
	rootCmd.Flags().BoolVar(&continueCmd, "continue", false, "Continue previous Claude session")
	rootCmd.Flags().Int64Var(&maxSize, "max-size", 20, "Maximum image size in MB")
	rootCmd.Flags().IntVar(&timeout, "timeout", 15, "Download timeout in seconds")
	rootCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode (errors only)")
}

// setupLogging configures the logger based on command line flags
func setupLogging() {
	if quiet {
		util.SetDefaultLogLevel(util.LogLevelQuiet)
	} else if verbose {
		util.SetDefaultLogLevel(util.LogLevelVerbose)
	} else {
		util.SetDefaultLogLevel(util.LogLevelNormal)
	}
}

// checkPrerequisites validates that required tools are available
func checkPrerequisites() error {
	// Check if gh CLI is available
	if err := github.IsGHCliAvailable(); err != nil {
		return util.NewAuthError("GitHub CLI not available: " + err.Error())
	}
	
	// If Claude integration is requested, check Claude CLI availability
	if sendPrompt != "" {
		if err := claude.IsClaudeAvailable(); err != nil {
			return util.NewValidationError("Claude CLI not available", 
				"Install Claude CLI or remove --send flag")
		}
	}
	
	return nil
}

func Execute() error {
	// Set up error handling
	if err := rootCmd.Execute(); err != nil {
		// Get appropriate exit code
		exitCode := util.GetExitCode(err)
		
		// Format error message
		logger := util.GetDefaultLogger()
		if appErr, ok := err.(*util.AppError); ok {
			logger.ErrorPlain("%s", appErr.String())
		} else {
			logger.ErrorPlain("Error: %v", err)
		}
		
		os.Exit(exitCode)
	}
	return nil
}