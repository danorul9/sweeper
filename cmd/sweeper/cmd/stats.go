package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show sweep statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Stats feature: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
