package actions

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/config"
)

type OperationLog struct {
	Timestamp time.Time
	Action    string
	Paths     []string
	TotalSize int64
	Success   bool
	Error     string
}

type LogEntry struct {
	Timestamp time.Time
	Success   bool
	Action    string
	Items     int
	Size      int64
	ErrorMsg  string
}

type LogStats struct {
	TotalScans    int
	TotalDeletes  int
	TotalSuccess  int
	TotalFail     int
	TotalSize     int64
	FirstScan     time.Time
	LastScan      time.Time
	RecentEntries []LogEntry
}

var logLineRE = regexp.MustCompile(`^\[([^\]]+)\] (OK|FAIL) \| action=(\S+) items=(\d+) size=(\d+)(?:: (.*))?$`)

func LogOperation(log OperationLog) error {
	supportDir, err := config.AppSupportDir()
	if err != nil {
		return err
	}

	logsDir := filepath.Join(supportDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	filename := time.Now().Format("2006-01-02") + ".log"
	path := filepath.Join(logsDir, filename)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	status := "OK"
	status2 := ""
	if !log.Success {
		status = "FAIL"
		status2 = ": " + log.Error
	}

	line := fmt.Sprintf("[%s] %s | action=%s items=%d size=%d%s\n",
		log.Timestamp.Format(time.RFC3339),
		status,
		log.Action,
		len(log.Paths),
		log.TotalSize,
		status2,
	)

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}

func LoadStats() (*LogStats, error) {
	supportDir, err := config.AppSupportDir()
	if err != nil {
		return nil, err
	}
	logsDir := filepath.Join(supportDir, "logs")

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &LogStats{}, nil
		}
		return nil, fmt.Errorf("read logs dir: %w", err)
	}

	stats := &LogStats{}
	var allEntries []LogEntry

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		fileEntries, err := parseLogFile(filepath.Join(logsDir, e.Name()))
		if err != nil {
			continue
		}
		allEntries = append(allEntries, fileEntries...)
	}

	if len(allEntries) == 0 {
		return stats, nil
	}

	stats.FirstScan = allEntries[0].Timestamp
	stats.LastScan = allEntries[len(allEntries)-1].Timestamp

	for _, entry := range allEntries {
		switch entry.Action {
		case "scan":
			stats.TotalScans++
		case "delete":
			stats.TotalDeletes++
		}
		if entry.Success {
			stats.TotalSuccess++
			stats.TotalSize += entry.Size
		} else {
			stats.TotalFail++
		}
	}

	n := len(allEntries)
	if n > 10 {
		n = 10
	}
	stats.RecentEntries = allEntries[len(allEntries)-n:]

	return stats, nil
}

func parseLogFile(path string) ([]LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := logLineRE.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339, matches[1])
		if err != nil {
			continue
		}

		items, _ := strconv.Atoi(matches[4])
		size, _ := strconv.ParseInt(matches[5], 10, 64)

		entries = append(entries, LogEntry{
			Timestamp: ts,
			Success:   matches[2] == "OK",
			Action:    matches[3],
			Items:     items,
			Size:      size,
			ErrorMsg:  matches[6],
		})
	}
	return entries, scanner.Err()
}
