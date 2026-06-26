# opencode-session-manager

> TUI for managing, recovering, and relocating OpenCode sessions across renamed or moved project directories.

`opencode-sm` is an interactive terminal companion for [OpenCode](https://opencode.ai).
It addresses a long-standing pain point: **renaming or moving a project folder makes
all its chat sessions disappear from the OpenCode UI**, even though the conversation
history is still safe in the local SQLite database.

## Version

**0.1.0-alpha.4** — pre-release. The feature set is functional but not yet
production-stable. Expect breaking schema changes until 1.0.0.

## Why

OpenCode stores each session's working directory as an absolute path at creation
time and never updates it. After a folder rename, the UI filters out sessions whose
`directory` no longer resolves on disk, even though `session.project_id` is still
valid (it is derived from the git root commit hash, which is stable across renames).

`opencode-sm` bridges that gap and adds session management features OpenCode does
not yet ship.

## Features

### Session Browser
- Browse all sessions in one place, split into Orphans and Active tabs
- Detail preview for any selected session: status, path, agent, timestamps
- Scrollable chat preview with markdown rendering (bold, italic, code,
  lists, headings, blockquotes) and scrollbar indicator
- Fixed "Chat Preview" header (doesn't scroll away)
- Toggle between orphans/active via tab, shift+tab, left/right arrows
- Live filter via list component
- OpenCode-inspired dark theme (Lipgloss)

### Relink Flow
- Press `r` on any orphan session to open the relink modal
- **Phase 1 (auto, project_id)** — automatically matches orphan sessions
  to their new path when the git root commit hash is still valid
- **Manual path remap** — browse the filesystem via Bubble filepicker
  to pick the new directory manually
- Confirmation modal showing old/new path, strategy, and backup info
- Running OpenCode instance detection (Linux) with warning and optional
  SIGTERM kill before destructive writes

### Cross-Project Migration
- Migrate N most recent sessions from one project to another
- Updates both `session.directory` and `session.project_id`

### Core Engine
- SQLite layer: list/find/get/update sessions, projects, and messages
- Phase 1 matching algorithm (adapted from bbl21's Python implementation)
- Manual path remap with preview
- Auto-backup (`<dbpath>.opencode-sm-backup`) before any write operation
- Transactional writes and error-safe rollback

### Install & Uninstall
- Cross-platform install scripts (bash + PowerShell)
- Self-uninstall via `opencode-sm uninstall` subcommand
- Shell completions for bash, zsh, fish, powershell
- GitHub Actions CI matrix across Ubuntu, macOS, Windows

## Installation

### From source (requires Go 1.26+)

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
opencode-sm                             # Launch TUI with default DB path
opencode-sm --db-path /path/to/db       # Use a specific opencode.db
opencode-sm --version                   # Print version and exit
opencode-sm --help                      # Print help
```

```sh
opencode-sm uninstall --dry-run    # Preview files to remove
opencode-sm uninstall --prefix ~/.local/bin  # Custom prefix
opencode-sm uninstall --purge      # Remove config and data too
```

### Keybindings

| Key | Mode | Action |
|---|---|---|
| `j`/`k`, `↑`/`↓` | List | Navigate sessions |
| `tab`/`→`, `shift+tab`/`←` | List | Switch tab |
| `r` | List | Open relink modal |
| `m` | List | Quick manual remap |
| `x` | List | Migrate to project |
| `q` | Any | Quit |
| `esc` | Modal | Cancel / go back |
| `y`, `enter` | Modal | Confirm |
| `n` | Modal | Reject |

## Architecture

```
cmd/opencode-sm/main.go          — CLI entry, flags, version
   internal/
     db/                            — SQLite queries + helpers
       types.go                     — Session, Project, Message structs
       session.go                   — ListSessions, GetSession, UpdateSessionDirectory
       project.go                   — ListProjects, BuildWorktreeIndex
       messages.go                  — ListMessages, MessagePart types
     relinker/                      — Core matching algorithm
       relinker.go                  — Phase 1 (project_id match), ApplyAll
       manual.go                    — Manual path remap, preview
       backup.go                    — Backup before write
     tui/                           — Bubble Tea TUI
       app.go                       — Root model, Init, View, Run
       data.go                      — Session loader (orphan/active split)
       keys.go                      — Keybindings
       update.go                    — Update + modal dispatch
       modal.go                     — Modal state machine
       relink_modal.go              — Relink flow (Phase 1 / Manual)
       apply.go                     — Apply pending relink
       filepicker_modal.go          — Directory picker
       confirm_modal.go             — Yes/no + result dialogs
       running_check.go             — Linux process detection
       process.go                   — SIGTERM kill
       list_view.go                 — Bubbles list wrapper
       detail_view.go               — Metadata + chat preview rendering
       markdown.go                  — Lightweight markdown renderer
       scrollbar.go                 — Scrollbar for chat preview
       styles.go                    — Lipgloss theme (OpenCode palette)
       util.go                      — Helpers
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
