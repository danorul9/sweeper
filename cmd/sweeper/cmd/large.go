package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

type LargeFile struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

var largeDirs = []string{"Downloads", "Desktop", "Documents", "Movies"}

var largeCmd = &cobra.Command{
	Use:   "large",
	Short: "Find files over 100MB in user directories",
	Long: `Scans ~/Downloads, ~/Desktop, ~/Documents, and ~/Movies
for files larger than the minimum size threshold (default 100MB).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		threshold, _ := cmd.Flags().GetInt64("min-size")

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("home dir: %w", err)
		}

		var largeFiles []LargeFile

		for _, d := range largeDirs {
			dir := filepath.Join(home, d)
			info, err := os.Stat(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				fmt.Fprintf(os.Stderr, "warning: %s: %v\n", dir, err)
				continue
			}
			if !info.IsDir() {
				continue
			}

			fmt.Fprintf(os.Stderr, "Scanning %s ...\n", dir)
			filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				fi, err := d.Info()
				if err != nil {
					return nil
				}
				if fi.Size() >= threshold {
					largeFiles = append(largeFiles, LargeFile{
						Path:    path,
						Size:    fi.Size(),
						ModTime: fi.ModTime().Format("2006-01-02 15:04"),
					})
				}
				return nil
			})
		}

		sort.Slice(largeFiles, func(i, j int) bool {
			return largeFiles[i].Size > largeFiles[j].Size
		})

		if jsonOutput {
			data, err := json.MarshalIndent(largeFiles, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if len(largeFiles) == 0 {
			fmt.Printf("No files over %s found.\n", formatSize(threshold))
			return nil
		}

		var total int64
		fmt.Printf("\nFiles over %s:\n\n", formatSize(threshold))
		for _, f := range largeFiles {
			total += f.Size
			fmt.Printf("  %-70s  %s\n", f.Path, formatSize(f.Size))
		}
		fmt.Printf("\nTotal: %d files, %s\n", len(largeFiles), formatSize(total))

		_ = dryRun
		return nil
	},
}

func init() {
	largeCmd.Flags().Bool("json", false, "Output as JSON")
	largeCmd.Flags().Bool("dry-run", false, "Preview without deleting")
	largeCmd.Flags().Int64("min-size", 100*1024*1024, "Minimum file size in bytes")
}
