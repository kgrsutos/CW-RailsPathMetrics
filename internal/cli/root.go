package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cwrstats",
	Short: "AWS CloudWatch Rails path metrics analyzer",
	Long: `cwrstats is a CLI tool that analyzes AWS CloudWatch logs for Rails applications.
It aggregates request metrics by path including request count, average, minimum, and maximum processing time.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
