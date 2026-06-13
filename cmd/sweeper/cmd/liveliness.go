package cmd

import (
	"fmt"
	"strings"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/liveliness"
	"github.com/spf13/cobra"
)

var livelinessCmd = &cobra.Command{
	Use:   "liveliness [path]",
	Short: "Evidence-based orphan detection for ~/.* directories",
	Long: `Score hidden home directories by measurable signals (mod time, open handles,
child file age, size, references to installed apps) instead of app fingerprints.

With no path, scans all hidden directories in ~/. With a path, scores a single directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			idx, err := appindex.BuildOrLoadCached()
			if err != nil {
				return fmt.Errorf("build app index: %w", err)
			}
			path := args[0]
			item := liveliness.ScorePath(path, idx)
			fmt.Printf("Path:    %s\n", item.Path)
			fmt.Printf("Name:    %s\n", item.Name)
			fmt.Printf("Size:    %d bytes\n", item.Size)
			fmt.Printf("Score:   %.1f\n", item.Score)
			fmt.Printf("Verdict: %s\n", item.Verdict)
			if len(item.Evidences) > 0 {
				fmt.Println("\nEvidence:")
				for _, e := range item.Evidences {
					fmt.Printf("  %s (%.1f): %s\n", e.Name, e.Weight, e.Detail)
				}
			}
			return nil
		}

		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return fmt.Errorf("build app index: %w", err)
		}
		items, err := liveliness.Run(idx)
		if err != nil {
			return err
		}
		fmt.Printf("%-6s  %6s  %10s  %s\n", "STATE", "SCORE", "SIZE", "NAME")
		fmt.Println(strings.Repeat("─", 52))
		for _, item := range items {
			scoreStr := fmt.Sprintf("%.0f%%", item.Score*100)
			sizeStr := humanSize(item.Size)
			state := "COLD"
			if item.Verdict == "dead" {
				state = "DEAD"
			}
			fmt.Printf("%-6s  %6s  %10s  %s\n", state, scoreStr, sizeStr, item.Name)
		}
		return nil
	},
}

func humanSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(livelinessCmd)
}
