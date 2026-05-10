package cmd

import (
	"os"

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
			return scanCmd.RunE(cmd, args)
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
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(undoCmd)
	rootCmd.AddCommand(reclaimCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(largeCmd)
	rootCmd.AddCommand(dupesCmd)
}
