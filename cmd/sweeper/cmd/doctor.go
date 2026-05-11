package cmd

import (
	"fmt"

	"github.com/danorul9/sweeper/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run system health checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		issues, err := doctor.Run()
		if err != nil {
			return fmt.Errorf("doctor: %w", err)
		}
		for _, iss := range issues {
			fmt.Printf("[%s] [%s] %s: %s\n", iss.Severity, iss.Category, iss.Path, iss.Description)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
