package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	outDir      string
	sendPrompt  string
	continueCmd bool
	maxSize     int64
	timeout     int
	force       bool
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
		target := args[0]
		
		fmt.Printf("Processing target: %s\n", target)
		
		if outDir != "" {
			fmt.Printf("Output directory: %s\n", outDir)
		} else {
			fmt.Println("Using memory mode (base64 output)")
		}
		
		if sendPrompt != "" {
			fmt.Printf("Claude prompt: %s\n", sendPrompt)
		}
		
		// TODO: Implement the main pipeline
		fmt.Println("Implementation coming in Phase 2...")
		
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
}

func Execute() error {
	return rootCmd.Execute()
}