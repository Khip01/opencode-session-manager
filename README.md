# opencode-session-manager

> TUI for managing, recovering, and relocating OpenCode sessions across renamed or moved project directories.

`opencode-sm` is an interactive terminal companion for [OpenCode](https://opencode.ai).
It addresses a long-standing pain point: **renaming or moving a project folder makes
all its chat sessions disappear from the OpenCode UI**, even though the conversation
history is still safe in the local SQLite database.

## Why

OpenCode stores each session's working directory as an absolute path at creation
time and never updates it. After a folder rename, the UI filters out sessions whose
`directory` no longer resolves on disk, even though `session.project_id` is still
valid (it is derived from the git root commit hash, which is stable across renames).

`opencode-sm` bridges that gap and adds session management features OpenCode does
not yet ship:

- Browse all sessions (orphans + active) in one place
- Recover conversations orphaned by folder rename via project_id match
- Manually relink sessions to any directory when automatic matching fails
- Migrate sessions between projects
- Detect running OpenCode instances before mutating the database
- Auto-backup the database before every write

## Status

**Early development.** Milestone 1 (foundation) is complete. Future milestones
will add the full feature set.

## Installation

### From source

```sh
go install github.com/Khip01/opencode-session-manager/cmd/opencode-sm@latest
```

### Build locally

```sh
make build
./bin/opencode-sm
```

## Usage

```sh
opencode-sm                       # Launch TUI with default DB path
opencode-sm --db-path /path/to/db # Use a specific opencode.db
opencode-sm --version             # Print version
opencode-sm --help                # Print help
```

## Credits & Inspiration

The Phase 1 (project_id matching) and Phase 2 (path heuristic) algorithms are
adapted from [bbl21/opencode-session-recovery](https://github.com/bbl21/opencode-session-recovery)
(MIT License). See `NOTICE` for full attribution.

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Bubbles](https://github.com/charmbracelet/bubbles), and
[Lipgloss](https://github.com/charmbracelet/lipgloss) from the Charm ecosystem.

The SQLite driver is [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite),
a pure-Go port of SQLite (no CGO required).

## License

MIT. See `LICENSE` for the full text and `NOTICE` for third-party attributions.
