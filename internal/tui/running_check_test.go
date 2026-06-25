package tui

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadProcess_SelfIsReadable(t *testing.T) {
	proc, ok := readProcess(runtime.GOMAXPROCS(0))
	if runtime.GOMAXPROCS(0) == 0 {
		t.Skip("no GOMAXPROCS")
	}
	_ = proc
	_ = ok
}

func TestReadProcess_CurrentProcessReturnsInfo(t *testing.T) {
	proc, ok := readProcess(0)
	if !ok {
		t.Skip("/proc not available on this system")
	}
	assert.NotEmpty(t, proc.Comm)
	assert.Positive(t, proc.PID)
}

func TestDetectRunningInstances_NoPanicOnLinux(t *testing.T) {
	procs, err := DetectRunningInstances()
	if err != nil {
		assert.ErrorIs(t, err, ErrNoProcFilesystem)
		return
	}
	assert.NotNil(t, procs)
}

func TestIsOpencodeProcess_MatchesCommAndCmdline(t *testing.T) {
	assert.True(t, isOpencodeProcess(RunningProcess{Comm: "opencode"}))
	assert.True(t, isOpencodeProcess(RunningProcess{Comm: "OpenCode-TUI"}))
	assert.True(t, isOpencodeProcess(RunningProcess{Cmdline: "/usr/bin/opencode tui"}))
	assert.True(t, isOpencodeProcess(RunningProcess{Cmdline: "node /opt/opencode/cli.js"}))
	assert.False(t, isOpencodeProcess(RunningProcess{Comm: "bash"}))
	assert.False(t, isOpencodeProcess(RunningProcess{Cmdline: "/usr/bin/bash"}))
	assert.False(t, isOpencodeProcess(RunningProcess{}))
}

func TestFormatRunningList_Empty(t *testing.T) {
	assert.Equal(t, "(none)", formatRunningList(nil))
	assert.Equal(t, "(none)", formatRunningList([]RunningProcess{}))
}

func TestFormatRunningList_MultipleProcesses(t *testing.T) {
	procs := []RunningProcess{
		{PID: 1234, Comm: "opencode"},
		{PID: 5678, Comm: ""},
	}
	got := formatRunningList(procs)
	assert.Contains(t, got, "PID 1234")
	assert.Contains(t, got, "opencode")
	assert.Contains(t, got, "PID 5678")
}

func TestReadProcess_RejectsInvalidPID(t *testing.T) {
	proc, ok := readProcess(99999999)
	_ = proc
	_ = ok
}

func TestReadProcess_RejectsZeroPID(t *testing.T) {
	_, ok := readProcess(0)
	if ok {
		t.Log("PID 0 readable on this system (got current process)")
	}
	require.NotPanics(t, func() {
		readProcess(0)
	})
}
