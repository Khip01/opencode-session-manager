package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func binaryFileName() string {
	if runtime.GOOS == "windows" {
		return "opencode-sm.exe"
	}
	return "opencode-sm"
}

func runUninstall(prefix string, purge, dryRun bool) int {
	fmt.Println("opencode-sm uninstaller")
	fmt.Println()

	if dryRun {
		fmt.Println("(DRY RUN, nothing will be removed)")
		fmt.Println()
	}

	result := doUninstall(prefix, purge, dryRun)
	printUninstallResult(result, dryRun)

	if len(result.errors) > 0 {
		return 1
	}
	return 0
}

type uninstallResult struct {
	removed      []string
	purgedDirs   []string
	purgedConfig string
	noConfig     bool
	errors       []error
}

func doUninstall(prefix string, purge, dryRun bool) uninstallResult {
	var r uninstallResult

	candidates := uninstallCandidates(prefix)
	for _, path := range candidates {
		if !fileExists(path) {
			continue
		}
		if err := removeBinary(path, dryRun); err != nil {
			r.errors = append(r.errors, err)
			continue
		}
		r.removed = append(r.removed, path)
	}

	// Clean up empty parent dirs only when we removed at least one
	// binary via standard location scan (not --prefix mode).
	if prefix == "" && len(r.removed) > 0 && !dryRun {
		for _, d := range emptyParentCandidates() {
			if isEmptyDir(d) {
				if err := os.Remove(d); err == nil {
					r.purgedDirs = append(r.purgedDirs, d)
				}
			}
		}
	}

	if purge {
		configDir := filepath.Join(homeDir(), ".config", "opencode-sm")
		if runtime.GOOS == "windows" {
			appData := os.Getenv("LOCALAPPDATA")
			if appData != "" {
				configDir = filepath.Join(appData, "opencode-sm")
			}
		}
		if dirExists(configDir) {
			if dryRun {
				r.purgedConfig = configDir
			} else if err := os.RemoveAll(configDir); err == nil {
				r.purgedConfig = configDir
			} else {
				r.errors = append(r.errors, err)
			}
		} else {
			r.noConfig = true
		}
	}

	return r
}

func uninstallCandidates(prefix string) []string {
	binary := binaryFileName()

	if prefix != "" {
		return []string{filepath.Join(prefix, binary)}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	candidates := []string{
		filepath.Join("/usr/local/bin", binary),
		filepath.Join(home, ".local", "bin", binary),
		filepath.Join(home, "go", "bin", binary),
		filepath.Join(home, "bin", binary),
	}

	if runtime.GOOS == "darwin" {
		candidates = append(candidates, filepath.Join("/opt/homebrew/bin", binary))
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("LOCALAPPDATA")
		if appData != "" {
			candidates = append(candidates,
				filepath.Join(appData, "Programs", "opencode-sm", binary))
		}
	}

	return candidates
}

func emptyParentCandidates() []string {
	home := homeDir()
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".local", "bin"),
	}
}

func removeBinary(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[dry-run] would remove: %s\n", path)
		return nil
	}
	dir := filepath.Dir(path)
	if !isWritableDir(dir) {
		return fmt.Errorf("cannot remove %s: directory not writable", path)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Printf("removed: %s\n", path)
	return nil
}

func printUninstallResult(r uninstallResult, dryRun bool) {
	if r.purgedConfig != "" {
		if dryRun {
			fmt.Printf("[dry-run] would purge: %s\n", r.purgedConfig)
		} else {
			fmt.Printf("purged: %s\n", r.purgedConfig)
		}
	} else if r.noConfig {
		configDir := filepath.Join(homeDir(), ".config", "opencode-sm")
		if runtime.GOOS == "windows" {
			appData := os.Getenv("LOCALAPPDATA")
			if appData != "" {
				configDir = filepath.Join(appData, "opencode-sm")
			}
		}
		fmt.Printf("no user config at %s\n", configDir)
	}

	for _, d := range r.purgedDirs {
		fmt.Printf("removed empty dir: %s\n", d)
	}

	if !dryRun && len(r.removed) == 0 {
		fmt.Println("")
		fmt.Println("opencode-sm not found in standard locations")
		fmt.Println("If installed elsewhere, remove manually")
	}

	if len(r.errors) > 0 {
		fmt.Println("")
		fmt.Println("errors:")
		for _, e := range r.errors {
			fmt.Printf("  %v\n", e)
		}
	}

	if !dryRun {
		fmt.Println("")
		fmt.Println("uninstall complete")
		fmt.Println("")
		fmt.Println("Note: this command does not touch:")
		fmt.Println("  - Backups (*.opencode-sm-backup) in the same dir as opencode.db")
		fmt.Println("  - opencode.db or any OpenCode data")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isEmptyDir(path string) bool {
	if !dirExists(path) {
		return false
	}
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}

func isWritableDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	f, err := os.CreateTemp(path, ".write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
