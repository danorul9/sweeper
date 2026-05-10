package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/danorul9/sweeper/internal/actions"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Historical cleanup analytics",
	Long: `Show aggregate statistics from all past sweeper runs:
scans, deletions, space reclaimed, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		stats, err := actions.LoadStats()
		if err != nil {
			return fmt.Errorf("load stats: %w", err)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				return fmt.Errorf("json marshal: %w", err)
			}
			os.Stdout.Write(data)
			os.Stdout.Write([]byte("\n"))
			return nil
		}

		if stats.TotalScans == 0 && stats.TotalDeletes == 0 {
			fmt.Println("No historical data yet. Run sweeper scan to get started.")
			return nil
		}

		fmt.Println("  Sweeper Stats")
		fmt.Println()
		fmt.Printf("  First scan:    %s\n", stats.FirstScan.Format("Jan 2, 2006 15:04"))
		fmt.Printf("  Last scan:     %s\n", stats.LastScan.Format("Jan 2, 2006 15:04"))
		fmt.Println()
		fmt.Printf("  Total scans:   %d\n", stats.TotalScans)
		fmt.Printf("  Total deletes: %d\n", stats.TotalDeletes)
		fmt.Printf("  Successful:    %d\n", stats.TotalSuccess)
		fmt.Printf("  Failed:        %d\n", stats.TotalFail)
		fmt.Printf("  Space freed:   %s\n", formatSize(stats.TotalSize))
		fmt.Println()
		fmt.Println("  Recent activity:")

		if len(stats.RecentEntries) == 0 {
			fmt.Println("    (none)")
		}
		for _, e := range stats.RecentEntries {
			status := "OK"
			if !e.Success {
				status = "FAIL"
			}
			fmt.Printf("    [%s] %s | %s items=%d size=%s\n",
				e.Timestamp.Format("2006-01-02 15:04"),
				status,
				e.Action,
				e.Items,
				formatSize(e.Size),
			)
		}

		return nil
	},
}