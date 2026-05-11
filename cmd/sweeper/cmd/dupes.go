package cmd

import (
	"fmt"

	"github.com/danorul9/sweeper/internal/dupes"
	"github.com/spf13/cobra"
)

var dupesFlags struct {
	aggressive bool
}

var dupesCmd = &cobra.Command{
	Use:   "dupes",
	Short: "Find duplicate files in common directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := dupes.Find(dupesFlags.aggressive)
		if err != nil {
			return fmt.Errorf("dupes scan: %w", err)
		}
		for _, g := range groups {
			fmt.Printf("Group (SHA256: %s, size: %d, count: %d):\n", g.SHA256, g.Size, g.Count)
			for _, f := range g.Files {
				fmt.Printf("  %s\n", f.Path)
			}
		}
		return nil
	},
}

func init() {
	dupesCmd.Flags().BoolVar(&dupesFlags.aggressive, "aggressive", false, "Scan additional directories")
	rootCmd.AddCommand(dupesCmd)
}
