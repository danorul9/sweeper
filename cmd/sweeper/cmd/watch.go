package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for new orphan files",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Watch feature: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
