package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type RunningProcess struct {
	PID     int
	Comm    string
	Cmdline string
}

var (
	ErrUnsupportedOS    = errors.New("running-process detection not implemented for this OS")
	ErrNoProcFilesystem = errors.New("/proc filesystem not available")
)

func DetectRunningInstances() ([]RunningProcess, error) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoProcFilesystem
		}
		return nil, err
	}

	var matches []RunningProcess
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		proc, ok := readProcess(pid)
		if !ok {
			continue
		}
		if isOpencodeProcess(proc) {
			matches = append(matches, proc)
		}
	}
	return matches, nil
}

func readProcess(pid int) (RunningProcess, bool) {
	proc := RunningProcess{PID: pid}

	if comm, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm")); err == nil {
		proc.Comm = strings.TrimSpace(string(comm))
	}
	if cmdline, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline")); err == nil {
		proc.Cmdline = strings.TrimSpace(strings.ReplaceAll(string(cmdline), "\x00", " "))
	}

	return proc, proc.Comm != "" || proc.Cmdline != ""
}

func isOpencodeProcess(p RunningProcess) bool {
	comm := strings.ToLower(p.Comm)
	cmdline := strings.ToLower(p.Cmdline)
	if strings.Contains(comm, "opencode") {
		return true
	}
	if strings.Contains(cmdline, "opencode") {
		return true
	}
	return false
}

func formatRunningList(procs []RunningProcess) string {
	if len(procs) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(procs))
	for _, p := range procs {
		name := p.Comm
		if name == "" {
			name = "opencode"
		}
		parts = append(parts, "PID "+itoa(p.PID)+" ("+name+")")
	}
	return strings.Join(parts, ", ")
}
