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
	"strings"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/Khip01/opencode-session-manager/internal/tui"
)

const defaultVersion = "0.1.0-alpha.4"

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

	// Go's flag package prints a noisy "flag needs an argument" error
	// when a string flag is given without a value, then dumps the full
	// usage. Pre-scan os.Args for a string flag that lacks a value
	// BEFORE calling Parse(), so we can show a clean focused message.
	if missing := detectMissingValue(os.Args[1:]); missing != "" {
		fmt.Fprintf(os.Stderr,
			"Error: -%s requires a value\n\n"+
				"Example:\n"+
				"  %s -%s /path/to/opencode.db\n\n"+
				"Run %s --help for full usage.\n",
			missing, os.Args[0], missing, os.Args[0])
		os.Exit(2)
	}
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n\n", err)
		flag.Usage()
		os.Exit(2)
	}

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
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// isMissingValueError reports whether err is the Go flag package's
// "flag needs an argument" error (or its "invalid argument" error
// when the value fails the parser, e.g. for bool/int).
func isMissingValueError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "flag needs an argument") ||
		strings.Contains(msg, "invalid value")
}

// missingFlagFromErr extracts the flag name from the Go flag
// package error message. Examples:
//
//	"flag needs an argument: -db-path" -> "db-path"
//	"invalid value \"\" for flag -watch"  -> "watch"
//
// Returns the flag name without the leading dash(es), or empty if
// it cannot be extracted.
func missingFlagFromErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, prefix := range []string{"flag needs an argument: -", "invalid value \""} {
		if i := strings.Index(msg, prefix); i >= 0 {
			rest := msg[i+len(prefix):]
			end := strings.IndexAny(rest, " \"")
			if end < 0 {
				end = len(rest)
			}
			return rest[:end]
		}
	}
	return ""
}

// detectMissingValue scans raw args for a string-valued flag that
// is not followed by a non-flag argument. We have to do this
// ourselves because Go's flag.Parse prints its own noisy error to
// stderr and then calls the usage function before we get a chance
// to intercept. Pre-scanning lets us catch the most common user
// mistake (typing `-db-path` with no value) before flag.Parse
// sees it. Returns the flag name without leading dashes, or "" if
// no missing value is detected.
func detectMissingValue(args []string) string {
	// Flags that take a string value. Must match the flag.Bool
	// declarations in main().
	stringFlags := map[string]bool{
		"db-path": true,
		"prefix":  true,
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		var name string
		switch {
		case strings.HasPrefix(a, "--"):
			name = strings.TrimPrefix(a, "--")
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--"):
			name = strings.TrimPrefix(a, "-")
		default:
			continue
		}
		// --flag=value form is always complete
		if eq := strings.Index(name, "="); eq >= 0 {
			continue
		}
		if !stringFlags[name] {
			continue
		}
		// Need a value after this flag
		next := ""
		if i+1 < len(args) {
			next = args[i+1]
		}
		if next == "" || strings.HasPrefix(next, "-") {
			return name
		}
	}
	return ""
}
