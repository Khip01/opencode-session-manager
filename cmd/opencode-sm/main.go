package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/Khip01/opencode-session-manager/internal/tui"
)

const defaultVersion = "0.1.0-dev"

var version = defaultVersion

func main() {
	var (
		dbPath  = flag.String("db-path", "", "Path to opencode.db (default: ~/.local/share/opencode/opencode.db)")
		showVer = flag.Bool("version", false, "Print version and exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "opencode-sm %s\n", version)
		fmt.Fprintf(os.Stderr, "TUI for managing, recovering, and relocating OpenCode sessions.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("opencode-sm %s\n", version)
		return
	}

	resolved, err := db.ResolvePath(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving database path: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(tui.Options{
		DBPath:  resolved,
		Version: version,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
