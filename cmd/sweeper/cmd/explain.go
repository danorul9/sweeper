package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/matcher"

	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain <path>",
	Short: "Show why a folder is considered leftover",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		targetPath := args[0]

		info, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("stat %s: %w", targetPath, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", targetPath)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		_ = cfg

		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return fmt.Errorf("build app index: %w", err)
		}

		m := matcher.New(idx)
		folderName := filepath.Base(targetPath)
		match := m.Match(folderName, targetPath, info.ModTime())

		if jsonOutput {
			data, err := json.MarshalIndent(match, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Path: %s\n", targetPath)
		fmt.Printf("Folder: %s\n", folderName)
		fmt.Printf("Verdict: %s\n", match.Verdict)
		fmt.Printf("Confidence: %.0f%%\n", match.Confidence*100)
		fmt.Println("Signals:")
		for _, s := range match.Signals {
			fmt.Printf("  [%s] %s (weight: %.2f)\n", s.Kind, s.Detail, s.Weight)
		}

		return nil
	},
}

func init() {
	explainCmd.Flags().Bool("json", false, "Output as JSON")
}
