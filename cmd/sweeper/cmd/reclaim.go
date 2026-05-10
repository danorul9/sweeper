package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/scanner"

	"github.com/spf13/cobra"
)

var reclaimCmd = &cobra.Command{
	Use:   "reclaim",
	Short: "Safe-to-delete caches and logs only",
	Long: `Aggressive only on known-safe categories: Caches, Logs, TemporaryItems,
Saved Application State. Never touches Application Support or Containers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return fmt.Errorf("build app index: %w", err)
		}

		s := scanner.New(cfg, config.ModeReclaim)
		s.SetIndex(idx)

		fmt.Fprintf(os.Stderr, "Scanning for safe-to-reclaim caches and logs...\n")
		result, err := s.Scan()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("json marshal: %w", err)
			}
			os.Stdout.Write(data)
			os.Stdout.Write([]byte("\n"))
			return nil
		}

		fmt.Printf("Reclaiming %s of safe-to-delete caches and logs...\n", formatSize(result.TotalSize))
		for _, item := range result.Items {
			fmt.Printf("  %-40s  %s\n", item.Name, formatSize(item.Size))
		}

		if dryRun {
			fmt.Fprintf(os.Stderr, "\nDry-run mode: no files were deleted.\n")
		} else {
			fmt.Println("\nDone.", formatSize(result.TotalSize), "reclaimed.")
		}
		return nil
	},
}

func init() {
	reclaimCmd.Flags().Bool("json", false, "Output as JSON")
	reclaimCmd.Flags().Bool("dry-run", false, "Preview without reclaiming")
}
