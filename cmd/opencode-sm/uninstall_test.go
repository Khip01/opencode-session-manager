package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	if fileExists(filepath.Join(dir, "missing")) {
		t.Error("fileExists returned true for non-existent file")
	}

	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !fileExists(existing) {
		t.Error("fileExists returned false for existing file")
	}

	if fileExists(dir) {
		t.Error("fileExists returned true for directory")
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()

	if dirExists(filepath.Join(dir, "missing")) {
		t.Error("dirExists returned true for non-existent dir")
	}

	if !dirExists(dir) {
		t.Error("dirExists returned false for existing dir")
	}
}

func TestIsEmptyDir(t *testing.T) {
	dir := t.TempDir()

	if !isEmptyDir(dir) {
		t.Error("isEmptyDir returned false for empty dir")
	}

	child := filepath.Join(dir, "child.txt")
	if err := os.WriteFile(child, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if isEmptyDir(dir) {
		t.Error("isEmptyDir returned true for non-empty dir")
	}

	if isEmptyDir(filepath.Join(dir, "missing")) {
		t.Error("isEmptyDir returned true for non-existent dir")
	}
}

func TestIsWritableDir(t *testing.T) {
	dir := t.TempDir()

	if !isWritableDir(dir) {
		t.Error("isWritableDir returned false for writable temp dir")
	}

	if isWritableDir(filepath.Join(dir, "missing")) {
		t.Error("isWritableDir returned true for non-existent dir")
	}
}

func TestBinaryFileName(t *testing.T) {
	got := binaryFileName()
	want := "opencode-sm"
	if runtime.GOOS == "windows" {
		want = "opencode-sm.exe"
	}
	if got != want {
		t.Errorf("binaryFileName() = %q, want %q", got, want)
	}
}

func TestUninstallCandidates_WithPrefix(t *testing.T) {
	c := uninstallCandidates("/custom/path")
	if len(c) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(c))
	}
	want := filepath.Join("/custom/path", binaryFileName())
	if c[0] != want {
		t.Errorf("candidate[0] = %q, want %q", c[0], want)
	}
}

func TestUninstallCandidates_NoPrefix(t *testing.T) {
	c := uninstallCandidates("")
	if len(c) < 3 {
		t.Errorf("expected at least 3 candidates, got %d", len(c))
	}
	foundHome := false
	for _, p := range c {
		if filepath.Dir(p) == filepath.Join(homeDir(), ".local", "bin") {
			foundHome = true
		}
	}
	if !foundHome {
		t.Error("expected ~/.local/bin in candidates")
	}
}

func TestEmptyParentCandidates(t *testing.T) {
	c := emptyParentCandidates()
	if len(c) == 0 {
		t.Error("expected at least one parent candidate")
	}
	found := false
	for _, p := range c {
		if p == filepath.Join(homeDir(), ".local", "bin") {
			found = true
		}
	}
	if !found {
		t.Error("expected ~/.local/bin in parent candidates")
	}
}

func TestRemoveBinary_DryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-binary")
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := removeBinary(path, true); err != nil {
		t.Errorf("dry-run removeBinary returned error: %v", err)
	}
	if !fileExists(path) {
		t.Error("dry-run removed the file")
	}
}

func TestRemoveBinary_Real(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-binary")
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := removeBinary(path, false); err != nil {
		t.Errorf("removeBinary returned error: %v", err)
	}
	if fileExists(path) {
		t.Error("real remove did not remove the file")
	}
}

func TestRemoveBinary_NotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission test")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}

	// Test the isWritableDir function directly with a read-only dir.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if isWritableDir(dir) {
		t.Error("isWritableDir should return false for read-only dir")
	}
}

func TestDoUninstall_PrefixMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, binaryFileName())
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := doUninstall(dir, false, false)

	found := false
	for _, rem := range r.removed {
		if rem == path {
			found = true
		}
	}
	if !found {
		t.Error("prefix mode did not remove the binary")
	}
	if fileExists(path) {
		t.Error("file still exists after uninstall")
	}
}

func TestDoUninstall_PrefixMode_NotFound(t *testing.T) {
	dir := t.TempDir()

	r := doUninstall(dir, false, false)

	if len(r.removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(r.removed))
	}
}

func TestDoUninstall_Purge_NoConfig(t *testing.T) {
	r := doUninstall("", true, false)
	if r.purgedConfig != "" {
		t.Errorf("expected empty purgedConfig, got %q", r.purgedConfig)
	}
	if !r.noConfig {
		t.Error("expected noConfig=true when config dir missing")
	}
}

func TestDoUninstall_Purge_WithConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "opencode-sm")
	if runtime.GOOS == "windows" {
		appData := t.TempDir()
		t.Setenv("LOCALAPPDATA", appData)
		configDir = filepath.Join(appData, "opencode-sm")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := doUninstall("", true, false)

	if r.purgedConfig != configDir {
		t.Errorf("expected purgedConfig=%q, got %q", configDir, r.purgedConfig)
	}
	if dirExists(configDir) {
		t.Error("config dir should be removed")
	}
}

func TestDoUninstall_PurgeDryRun(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "opencode-sm")
	if runtime.GOOS == "windows" {
		appData := t.TempDir()
		t.Setenv("LOCALAPPDATA", appData)
		configDir = filepath.Join(appData, "opencode-sm")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := doUninstall("", true, true)

	if r.purgedConfig != configDir {
		t.Errorf("expected purgedConfig=%q in dry-run, got %q", configDir, r.purgedConfig)
	}
	if !dirExists(configDir) {
		t.Error("dry-run should not remove config dir")
	}
}

func TestDoUninstall_CleansEmptyParentDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	localBin := filepath.Join(tmpHome, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	binary := filepath.Join(localBin, binaryFileName())
	if err := os.WriteFile(binary, []byte("x"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := doUninstall("", false, false)

	foundDir := false
	for _, d := range r.purgedDirs {
		if d == localBin {
			foundDir = true
		}
	}
	if !foundDir {
		t.Errorf("expected %s in purgedDirs, got %v", localBin, r.purgedDirs)
	}
}

func TestHomeDir(t *testing.T) {
	home := homeDir()
	if home == "" {
		t.Skip("no home directory available")
	}
}
