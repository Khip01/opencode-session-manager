// Package main is the entry point for the opencode-sm CLI binary.
//
// Two primary modes:
//   - TUI mode (default): launch the Bubble Tea session manager
//   - Uninstall mode (--uninstall): remove the binary from standard
//     install locations. Supports --prefix, --purge, --dry-run flags.
//
// Both modes share --version and --help.
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
	// Subcommand dispatch: allow `opencode-sm uninstall` in addition
	// to `opencode-sm --uninstall`. Flag package only handles dashes,
	// so we rewrite the argv before parsing if the first non-flag arg
	// is a recognized subcommand.
	for _, a := range os.Args[1:] {
		switch a {
		case "uninstall":
			os.Args = append([]string{os.Args[0], "--uninstall"}, os.Args[2:]...)
		}
	}

	var (
		dbPath    = flag.String("db-path", "", "Path to opencode.db (default: ~/.local/share/opencode/opencode.db)")
		watch     = flag.Bool("watch", false, "Auto-refresh session list when opencode.db changes")
		showVer   = flag.Bool("version", false, "Print version and exit")
		showHelp  = flag.Bool("help", false, "Show help")
		uninstall = flag.Bool("uninstall", false, "Remove the opencode-sm binary from standard install locations")
		prefix    = flag.String("prefix", "", "With --uninstall: remove the binary from DIR instead of scanning standard locations")
		purge     = flag.Bool("purge", false, "With --uninstall: also remove ~/.config/opencode-sm/")
		dryRun    = flag.Bool("dry-run", false, "With --uninstall: show what would be removed without removing")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "opencode-sm %s\n", version)
		fmt.Fprintf(os.Stderr, "TUI for managing, recovering, and relocating OpenCode sessions.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Run modes:\n")
		fmt.Fprintf(os.Stderr, "  (default)        Launch the TUI session manager\n")
		fmt.Fprintf(os.Stderr, "  --uninstall      Remove the binary from standard install locations\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showHelp {
		flag.Usage()
		return
	}

	if *showVer {
		fmt.Printf("opencode-sm %s\n", version)
		return
	}

	if *uninstall {
		os.Exit(runUninstall(*prefix, *purge, *dryRun))
	}

	resolved, err := db.ResolvePath(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving database path: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(tui.Options{
		DBPath:  resolved,
		Version: version,
		Watch:   *watch,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
