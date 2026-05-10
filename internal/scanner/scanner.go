package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/matcher"
)

type Scanner struct {
	cfg      *config.Config
	mode     config.ScanMode
	index    *appindex.AppIndex
	matcher  *matcher.Matcher
	maxDepth int
}

func New(cfg *config.Config, mode config.ScanMode) *Scanner {
	idx := &appindex.AppIndex{}
	matcher := matcher.New(idx)

	return &Scanner{
		cfg:      cfg,
		mode:     mode,
		index:    idx,
		matcher:  matcher,
		maxDepth: 10,
	}
}

func (s *Scanner) SetIndex(idx *appindex.AppIndex) {
	s.index = idx
	s.matcher = matcher.New(idx)
}

func (s *Scanner) SetMaxDepth(d int) {
	s.maxDepth = d
}

func (s *Scanner) Scan() (*ScanResult, error) {
	start := time.Now()

	var locations []Location
	switch s.mode {
	case config.ModeSafe:
		locations = SafeLocations()
	case config.ModeAggressive:
		locations = AggressiveLocations()
	case config.ModeReclaim:
		locations = ReclaimLocations()
	default:
		locations = SafeLocations()
	}

	leftoverCh := make(chan Leftover, 1000)
	g, ctx := errgroup.WithContext(context.Background())
	sem := semaphore.NewWeighted(int64(runtime.NumCPU() * 2))

	for _, loc := range locations {
		loc := loc
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf("acquire semaphore: %w", err)
		}
		g.Go(func() error {
			defer sem.Release(1)
			return s.scanLocation(ctx, loc, leftoverCh)
		})
	}

	go func() {
		g.Wait()
		close(leftoverCh)
	}()

	var items []Leftover
	var totalSize int64
	for lo := range leftoverCh {
		items = append(items, lo)
		totalSize += lo.Size
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := &ScanResult{
		Items:     items,
		TotalSize: totalSize,
		Duration:  time.Since(start).Round(time.Millisecond).String(),
	}

	if len(items) == 0 {
		result.Items = []Leftover{}
	}

	return result, nil
}

func (s *Scanner) scanLocation(ctx context.Context, loc Location, results chan<- Leftover) error {
	folders, err := ListFolders(loc.Path)
	if err != nil {
		return nil
	}

	for _, folderPath := range folders {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		folderName := filepath.Base(folderPath)

		if s.shouldIgnore(folderName) {
			continue
		}

		if matcher.IsSystemPath(folderName) {
			continue
		}

		if family := s.index.FamilyForFolder(folderName); family != nil {
			if s.index.IsFamilyInstalled(*family) {
				continue
			}
		}

		info, err := os.Stat(folderPath)
		if err != nil {
			continue
		}

		match := s.matcher.Match(folderName, folderPath, info.ModTime())
		if match.Verdict == VerdictInstalled {
			continue
		}

		size := DirSize(folderPath)

		if match.Confidence < 0.2 {
			continue
		}

		results <- Leftover{
			Path:     folderPath,
			Name:     folderName,
			Size:     size,
			Location: string(loc.Type),
			ModTime:  info.ModTime(),
			Match:    match,
		}
	}
	return nil
}

func (s *Scanner) shouldIgnore(name string) bool {
	for _, ignore := range s.cfg.Ignore {
		if matched, _ := filepath.Match(ignore, name); matched {
			return true
		}
		if name == ignore {
			return true
		}
	}
	return false
}
