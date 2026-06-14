package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/danorul9/sweeper/internal/ui"
	"github.com/spf13/cobra"
)
var (
	rootCmd = &cobra.Command{
		Use:   "sweeper",
		Short: "macOS app leftover detector & cleaner",
		Long: `Sweeper detects and cleans files left behind by uninstalled applications
on macOS. It scans library paths, cross-references leftovers against
installed apps using bundle IDs, fingerprints, and confidence scoring.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ui.RunHub()
			return nil
		},
	}
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersion(ver, commit, date string) {
	appVersion = ver
	appCommit = commit
	appDate = date
	rootCmd.Version = ver
	rootCmd.SetVersionTemplate("sweeper {{.Version}} (commit: " + appCommit + ", built: " + appDate + ")\n")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(explainCmd)
}

// maybeJSON checks the --json flag. If set, marshals data to stdout
// and returns (true, nil). If the flag is not set, returns (false, nil).
func maybeJSON(cmd *cobra.Command, data any) (bool, error) {
	if jsonOut, _ := cmd.Flags().GetBool("json"); !jsonOut {
		return false, nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return true, fmt.Errorf("marshal output: %w", err)
	}
	return true, nil
}
