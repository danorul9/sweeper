package cmd

import (
	"fmt"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/scanner"
	"github.com/spf13/cobra"
)

var scanFlags struct {
	safe bool
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for orphaned app leftovers",
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return fmt.Errorf("build app index: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		mode := config.ModeAggressive
		if scanFlags.safe {
			mode = config.ModeSafe
		}

		s := scanner.New(cfg, mode)
		s.SetIndex(idx)

		result, err := s.Scan()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		for _, item := range result.Items {
			fmt.Printf("%s (confidence: %.0f%%, size: %d)\n", item.Path, item.Match.Confidence*100, item.Size)
		}
		fmt.Printf("\nTotal items: %d, Total size: %d bytes, Duration: %s\n", len(result.Items), result.TotalSize, result.Duration)
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanFlags.safe, "safe", false, "Use safe mode (cached paths only)")
	rootCmd.AddCommand(scanCmd)
}
