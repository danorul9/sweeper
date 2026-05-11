package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Restore files from last trash snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		snapDir := filepath.Join(home, ".config", "sweeper", "snapshots")
		entries, err := os.ReadDir(snapDir)
		if err != nil {
			return fmt.Errorf("no snapshots found: %w", err)
		}
		if len(entries) == 0 {
			fmt.Println("No snapshots available.")
			return nil
		}
		latest := entries[len(entries)-1]
		data, err := os.ReadFile(filepath.Join(snapDir, latest.Name()))
		if err != nil {
			return err
		}
		fmt.Printf("Latest snapshot: %s\n%s\n", latest.Name(), string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
}
