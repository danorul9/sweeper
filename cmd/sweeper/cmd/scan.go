package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/core"
	"github.com/danorul9/sweeper/internal/scanner"
	"github.com/danorul9/sweeper/internal/telemetry"
	"github.com/danorul9/sweeper/internal/ui"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for leftover app files",
	Long: `Scan ~/Library paths for folders left behind by uninstalled applications.
Interactive TUI by default. Use --json for machine output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		aggressive, _ := cmd.Flags().GetBool("aggressive")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		shareTelemetry, _ := cmd.Flags().GetBool("share-telemetry")

		mode := config.ModeSafe
		if aggressive {
			mode = config.ModeAggressive
		}

		fmt.Fprintf(os.Stderr, "Building app index...\n")
		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return fmt.Errorf("build app index: %w", err)
		}
		fmt.Fprintf(os.Stderr, "App index built: %d apps, %d bundle IDs\n", len(idx.Names), len(idx.BundleIDs))

		s := scanner.New(cfg, mode)
		s.SetIndex(idx)

		fmt.Fprintf(os.Stderr, "Scanning library paths (%s mode)...\n", mode)
		result, err := s.Scan()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if shareTelemetry {
			recordTelemetry(result)
		}

		if jsonOutput {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal result: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if dryRun {
			fmt.Printf("\nScan complete in %s\n", result.Duration)
			if len(result.Items) == 0 {
				fmt.Println("No leftovers found.")
				return nil
			}
			for _, item := range result.Items {
				confidence := 0.0
				verdict := "unknown"
				if item.Match != nil {
					confidence = item.Match.Confidence
					verdict = item.Match.Verdict.String()
				}
				fmt.Printf("  %-40s  %s\n", item.Name, formatVerdict(verdict, confidence))
				if item.Match != nil {
					for _, s := range item.Match.Signals {
						fmt.Printf("    %s\n", s.Detail)
					}
				}
			}
			fmt.Printf("\nTotal: %d items, %s\n", len(result.Items), formatSize(result.TotalSize))
			return nil
		}

		if !isInteractive() {
			return printTerminal(result)
		}
		ui.RunTUI(result)
		return nil
	},
}

func init() {
	scanCmd.Flags().Bool("json", false, "Output as JSON")
	scanCmd.Flags().Bool("aggressive", false, "Scan containers, prefs, app support")
	scanCmd.Flags().Bool("dry-run", false, "Preview without deleting")
	scanCmd.Flags().Bool("share-telemetry", false, "Submit unknown folder + bundle ID pairs to improve fingerprint DB")
}

func recordTelemetry(result *core.ScanResult) {
	for _, item := range result.Items {
		if item.Match == nil {
			continue
		}
		if item.Match.Verdict != core.VerdictUncertain && item.Match.Verdict != core.VerdictLeftover {
			continue
		}
		var signals []string
		for _, s := range item.Match.Signals {
			signals = append(signals, s.Kind)
		}
		sub := telemetry.Submission{
			Folder:   item.Name,
			Location: item.Location,
			Size:     item.Size,
			Signals:  signals,
		}
		if err := telemetry.Record(sub); err != nil {
			fmt.Fprintf(os.Stderr, "warning: telemetry: %v\n", err)
		}
	}
	count, _ := telemetry.Count()
	fmt.Fprintf(os.Stderr, "Telemetry: %d observations recorded (total: %d)\n", len(result.Items), count)
}

func isInteractive() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func printTerminal(result *core.ScanResult) error {
	fmt.Printf("\nScan complete in %s\n", result.Duration)
	if len(result.Items) == 0 {
		fmt.Println("No leftovers found.")
		return nil
	}
	for _, item := range result.Items {
		confidence := 0.0
		verdict := "unknown"
		if item.Match != nil {
			confidence = item.Match.Confidence
			verdict = item.Match.Verdict.String()
		}
		fmt.Printf("  %-40s  %s\n", item.Name, formatVerdict(verdict, confidence))
		if item.Match != nil {
			for _, s := range item.Match.Signals {
				fmt.Printf("    %s\n", s.Detail)
			}
		}
	}
	fmt.Printf("\nTotal: %d items, %s\n", len(result.Items), formatSize(result.TotalSize))
	return nil
}

func formatVerdict(verdict string, confidence float64) string {
	switch verdict {
	case "LEFTOVER":
		return fmt.Sprintf("SAFE (%.0f%%)", confidence*100)
	case "INSTALLED":
		return fmt.Sprintf("KEEP (%.0f%%)", confidence*100)
	case "UNCERTAIN":
		return fmt.Sprintf("UNSURE (%.0f%%)", confidence*100)
	default:
		return fmt.Sprintf("UNKNOWN (%.0f%%)", confidence*100)
	}
}

func formatSize(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
