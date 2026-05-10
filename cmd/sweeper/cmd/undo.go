package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/danorul9/sweeper/internal/actions"

	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Restore files from the last cleanup snapshot",
	Long: `Read the latest snapshot from ~/Library/Application Support/Sweeper/snapshots/
and restore files from Trash.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		snapshot, err := actions.LoadLatestSnapshot()
		if err != nil {
			return fmt.Errorf("load snapshot: %w", err)
		}
		if snapshot == nil {
			fmt.Println("No cleanup snapshots found.")
			return nil
		}

		if jsonOutput {
			data, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Restoring snapshot from %s (%d items)...\n",
			snapshot.Timestamp.Format("Jan 2, 2006 15:04:05"),
			len(snapshot.Items))

		if dryRun {
			fmt.Println("Dry-run: no files restored.")
			return nil
		}

		if err := snapshot.Restore(); err != nil {
			return fmt.Errorf("restore: %w", err)
		}

		fmt.Println("Restore complete.")
		return nil
	},
}

func init() {
	undoCmd.Flags().Bool("json", false, "Output as JSON")
	undoCmd.Flags().Bool("dry-run", false, "Preview without restoring")
}
