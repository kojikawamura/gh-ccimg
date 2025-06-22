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
	debug       bool
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
		util.Debug("Input target: %s", target)
		owner, repo, num, err := github.ParseTarget(target)
		if err != nil {
			util.Debug("Parse error: %v", err)
			return util.NewValidationError(fmt.Sprintf("Invalid target format: %s", target), 
				"Use format: OWNER/REPO#NUM or https://github.com/OWNER/REPO/issues/NUM")
		}
		util.Verbose("Parsed: %s/%s#%s", owner, repo, num)
		util.Debug("Parsed components - Owner: %s, Repo: %s, Number: %s", owner, repo, num)

		// Step 2: Check prerequisites
		util.Debug("Checking prerequisites...")
		if err := checkPrerequisites(); err != nil {
			util.Debug("Prerequisites check failed: %v", err)
			return err
		}
		util.Debug("Prerequisites check passed")

		// Step 3: Fetch GitHub data
		util.Info("Fetching GitHub data...")
		util.Debug("Creating GitHub client with timeout: %ds", timeout)
		client := github.NewClient(time.Duration(timeout) * time.Second)
		
		util.Debug("Fetching issue/PR data from GitHub API...")
		issue, err := client.FetchIssue(owner, repo, num)
		if err != nil {
			util.Debug("Failed to fetch issue: %v", err)
			return util.NewNetworkError("Failed to fetch issue/PR data", err)
		}
		util.Debug("Issue fetched successfully, body length: %d characters", len(issue.Body))
		
		util.Debug("Fetching comments from GitHub API...")
		comments, err := client.FetchComments(owner, repo, num)
		if err != nil {
			util.Debug("Failed to fetch comments: %v", err)
			return util.NewNetworkError("Failed to fetch comments", err)
		}
		util.Verbose("Fetched issue and %d comments", len(comments))
		util.Debug("Comments fetched successfully, count: %d", len(comments))

		// Step 4: Extract image URLs
		util.Info("Extracting image URLs from markdown...")
		util.Debug("Starting image URL extraction from markdown content")
		var allURLs []string
		
		// From issue body
		util.Debug("Extracting URLs from issue body...")
		issueURLs := markdown.ExtractImageURLs(issue.Body)
		util.Debug("Found %d URLs in issue body", len(issueURLs))
		for i, url := range issueURLs {
			util.Debug("Issue URL %d: %s", i+1, url)
		}
		allURLs = append(allURLs, issueURLs...)
		
		// From comments
		util.Debug("Extracting URLs from %d comments...", len(comments))
		for i, comment := range comments {
			commentURLs := markdown.ExtractImageURLs(comment.Body)
			util.Debug("Found %d URLs in comment %d", len(commentURLs), i+1)
			for j, url := range commentURLs {
				util.Debug("Comment %d URL %d: %s", i+1, j+1, url)
			}
			allURLs = append(allURLs, commentURLs...)
		}
		
		if len(allURLs) == 0 {
			util.Debug("No image URLs found in any markdown content")
			util.Warn("No images found in issue/PR %s/%s#%s", owner, repo, num)
			return nil
		}
		util.Success("Found %d image URLs", len(allURLs))
		util.Debug("Total unique URLs to download: %d", len(allURLs))

		// Step 5: Download images
		util.Info("Downloading images...")
		maxSizeBytes := maxSize * 1024 * 1024 // Convert MB to bytes
		util.Debug("Download configuration - Max size: %d MB (%d bytes), Timeout: %ds, Concurrency: 5", maxSize, maxSizeBytes, timeout)
		fetcher := download.NewFetcher(maxSizeBytes, time.Duration(timeout)*time.Second, 5)
		
		// Set up progress reporting
		if verbose || debug {
			reporter := download.NewConsoleReporter(os.Stderr, true)
			fetcher.SetReporter(reporter)
		} else if !quiet {
			reporter := download.NewConsoleReporter(os.Stderr, false)
			fetcher.SetReporter(reporter)
		}
		
		util.Debug("Starting concurrent download of %d URLs...", len(allURLs))
		ctx := context.Background()
		results := fetcher.FetchConcurrent(ctx, allURLs)
		
		// Count successful downloads and log failures
		successCount := 0
		var successfulResults []download.Result
		var failureReasons []string
		for _, result := range results {
			if result.Error == nil {
				successCount++
				successfulResults = append(successfulResults, result)
				util.Debug("Successfully downloaded %s (%d bytes, %s)", result.URL, result.Size, result.ContentType)
			} else {
				util.Verbose("Failed to download %s: %v", result.URL, result.Error)
				util.Debug("Download failure for %s: %v", result.URL, result.Error)
				failureReasons = append(failureReasons, fmt.Sprintf("%s: %v", result.URL, result.Error))
			}
		}
		
		if successCount == 0 {
			util.Debug("All downloads failed. Failure summary: %v", failureReasons)
			suggestion := "Check that the URLs are accessible and contain valid images. Use --debug for detailed error information"
			if len(failureReasons) > 0 {
				suggestion += fmt.Sprintf(". Common issues: network connectivity, rate limiting, invalid URLs, or files too large (current limit: %dMB)", maxSize)
			}
			return util.NewValidationError("No images could be downloaded", suggestion)
		}
		util.Success("Downloaded %d/%d images successfully", successCount, len(allURLs))
		util.Debug("Download completed. Success: %d, Failures: %d", successCount, len(allURLs)-successCount)

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
			
			// Security warning for sensitive data
			warnSensitiveData(successfulResults, owner, repo, num)
			
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
			util.Debug("Executing Claude with prompt length: %d characters, image count: %d", len(sanitizedPrompt), len(imageData))
			if err := claude.ExecuteClaude(sanitizedPrompt, imageData, continueCmd); err != nil {
				util.Debug("Claude execution failed: %v", err)
				return util.NewClaudeError("Claude execution failed", err)
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
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Debug mode (detailed troubleshooting info)")
}

// setupLogging configures the logger based on command line flags
func setupLogging() {
	if quiet {
		util.SetDefaultLogLevel(util.LogLevelQuiet)
	} else if debug {
		util.SetDefaultLogLevel(util.LogLevelDebug)
		util.Debug("Debug mode enabled - detailed troubleshooting information will be shown")
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

// warnSensitiveData displays security warnings about potentially sensitive data
func warnSensitiveData(results []download.Result, owner, repo, num string) {
	util.Warn("🔒 SECURITY WARNING: You are about to send image data to Claude")
	util.Warn("   • Repository: %s/%s#%s", owner, repo, num)
	util.Warn("   • Image count: %d", len(results))
	util.Warn("   • These images may contain sensitive information:")
	util.Warn("     - API keys, tokens, or passwords")
	util.Warn("     - Internal system details or configurations")
	util.Warn("     - Personal or confidential information")
	util.Warn("     - Proprietary code or business logic")
	util.Warn("   • Data will be sent to Anthropic's Claude service")
	util.Warn("   • Review all images before proceeding")
	util.Warn("")
}