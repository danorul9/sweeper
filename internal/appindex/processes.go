package appindex

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type ProcessInfo struct {
	PID  int
	Name string
	Path string
}

var (
	runningProcs   map[string]bool
	runningProcsMu sync.Mutex
)

func GetRunningProcesses() (map[string]bool, error) {
	runningProcsMu.Lock()
	defer runningProcsMu.Unlock()

	if runningProcs != nil {
		return runningProcs, nil
	}

	runningProcs = make(map[string]bool)

	cmd := exec.Command("ps", "-eo", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return runningProcs, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			runningProcs[name] = true
			base := filepath.Base(name)
			if base != name {
				runningProcs[base] = true
			}
		}
	}

	return runningProcs, nil
}

func ClearProcessCache() {
	runningProcsMu.Lock()
	defer runningProcsMu.Unlock()
	runningProcs = nil
}

func FindProcessesForApp(appName string) []ProcessInfo {
	procs, err := GetRunningProcesses()
	if err != nil {
		return nil
	}

	var result []ProcessInfo
	lower := strings.ToLower(appName)
	for name := range procs {
		base := strings.ToLower(filepath.Base(name))
		if base == lower || strings.Contains(base, lower) {
			result = append(result, ProcessInfo{Name: name})
		}
	}
	return result
}

func IsProcessRunning(execName string) bool {
	procs, err := GetRunningProcesses()
	if err != nil {
		return false
	}
	if procs[execName] {
		return true
	}
	lower := strings.ToLower(execName)
	for name := range procs {
		if strings.EqualFold(filepath.Base(name), execName) {
			return true
		}
		if strings.ToLower(filepath.Base(name)) == lower {
			return true
		}
	}
	return false
}
