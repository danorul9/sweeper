package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reclaimCmd = &cobra.Command{
	Use:   "reclaim",
	Short: "Show reclaimable space summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Reclaimable space: run 'sweeper scan' then review results")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reclaimCmd)
}
