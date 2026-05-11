package cmd

import (
	"fmt"

	"github.com/danorul9/sweeper/internal/large"
	"github.com/spf13/cobra"
)

var largeFlags struct {
	threshold int64
}

var largeCmd = &cobra.Command{
	Use:   "large",
	Short: "Find large files in common directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := large.Scan(largeFlags.threshold)
		if err != nil {
			return fmt.Errorf("large scan: %w", err)
		}
		for _, f := range files {
			fmt.Printf("%s (%d bytes)\n", f.Path, f.Size)
		}
		return nil
	},
}

func init() {
	largeCmd.Flags().Int64Var(&largeFlags.threshold, "threshold", 100_000_000, "Size threshold in bytes")
	rootCmd.AddCommand(largeCmd)
}
