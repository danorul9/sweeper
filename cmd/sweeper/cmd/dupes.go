package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/cobra"
)

type DupeFile struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

type DupeGroup struct {
	SHA256    string     `json:"sha256"`
	Size      int64      `json:"size"`
	Count     int        `json:"count"`
	TotalSize int64      `json:"total_size"`
	Files     []DupeFile `json:"files"`
}

var dupeDirs = []string{"Downloads", "Desktop", "Documents"}
var dupeDirsAggressive = []string{"Downloads", "Desktop", "Documents", "Pictures", "Movies", "Music"}

var dupesCmd = &cobra.Command{
	Use:   "dupes",
	Short: "Find duplicate files by checksum",
	Long: `Scan directories and find duplicate files using xxhash for fast first pass
and SHA-256 for confirmation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		aggressive, _ := cmd.Flags().GetBool("aggressive")

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("home dir: %w", err)
		}

		dirs := dupeDirs
		if aggressive {
			dirs = dupeDirsAggressive
		}

		// Phase 1: xxhash fast pass
		fmt.Fprintf(os.Stderr, "Phase 1: Fast checksum (xxhash)...\n")
		hashGroups := make(map[uint64][]string)

		for _, d := range dirs {
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

			fmt.Fprintf(os.Stderr, "  Scanning %s ...\n", dir)
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
				// skip empty files
				if fi.Size() == 0 {
					return nil
				}
				// skip files under 1KB for aggressive; always skip under 1 byte
				if fi.Size() < 1 {
					return nil
				}

				h := xxhash.New()
				f, err := os.Open(path)
				if err != nil {
					return nil
				}
				_, err = io.Copy(h, f)
				f.Close()
				if err != nil {
					return nil
				}
				sum := h.Sum64()
				hashGroups[sum] = append(hashGroups[sum], path)
				return nil
			})
		}

		// Phase 2: SHA-256 confirmation for collisions
		fmt.Fprintf(os.Stderr, "Phase 2: Confirming duplicates (SHA-256)...\n")
		var groups []DupeGroup

		for _, paths := range hashGroups {
			if len(paths) < 2 {
				continue
			}

			// group by SHA-256
			shaGroups := make(map[string][]string)
			for _, p := range paths {
				f, err := os.Open(p)
				if err != nil {
					continue
				}
				h := sha256.New()
				_, err = io.Copy(h, f)
				f.Close()
				if err != nil {
					continue
				}
				sum := hex.EncodeToString(h.Sum(nil))
				shaGroups[sum] = append(shaGroups[sum], p)
			}

			for sha, dupePaths := range shaGroups {
				if len(dupePaths) < 2 {
					continue
				}

				var files []DupeFile
				var totalSize int64
				for _, p := range dupePaths {
					fi, err := os.Stat(p)
					if err != nil {
						continue
					}
					files = append(files, DupeFile{
						Path:    p,
						Size:    fi.Size(),
						ModTime: fi.ModTime().Format("2006-01-02 15:04"),
					})
					totalSize += fi.Size()
				}

				if len(files) >= 2 {
					groups = append(groups, DupeGroup{
						SHA256:    sha,
						Size:      files[0].Size,
						Count:     len(files),
						TotalSize: totalSize,
						Files:     files,
					})
				}
			}
		}

		// sort by total waste descending
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].TotalSize > groups[j].TotalSize
		})

		if jsonOutput {
			data, err := json.MarshalIndent(groups, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if len(groups) == 0 {
			fmt.Println("No duplicate files found.")
			return nil
		}

		var totalWaste int64
		for _, g := range groups {
			waste := g.TotalSize - g.Size
			totalWaste += waste
			fmt.Printf("\n  %s  (%d copies, %s wasted)\n", g.SHA256[:12], g.Count, formatSize(waste))
			for _, f := range g.Files {
				fmt.Printf("    %s  %s\n", formatSize(f.Size), f.Path)
			}
		}
		fmt.Printf("\nTotal: %d duplicate groups, %s reclaimable\n", len(groups), formatSize(totalWaste))

		return nil
	},
}

func init() {
	dupesCmd.Flags().Bool("json", false, "Output as JSON")
	dupesCmd.Flags().Bool("aggressive", false, "Scan more directories (Pictures, Movies, Music)")
}
