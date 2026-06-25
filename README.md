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
- Migrate sessions between projects (updates directory + project_id atomically)
- Detect running OpenCode instances before mutating the database
- Auto-backup the database before every write
- Watch mode for live refresh when opencode.db changes

## TUI Preview

```
┌─ opencode-sm 73f1baf ● watching ───────────── DB: ~/.local/share/opencode/opencode.db ─┐
│  [Orphans (4)] [Active (30)]                                                                 │
├──────────────────────────────────────┬──────────────────────────────────────────────────────┤
│  Sessions                             │  Preview                                              │
│  ▸ [!] ses_18160c697f  New session -   │  New session - 2026-05-31T15:20:24.680Z               │
│    [!] ses_18180ef2a7  Analisis Lag    │                                                       │
│    [!] ses_181d994e2f  Analisis Lag    │  ID:           ses_18160c697ffe1ZbKpabSTmp6Gd         │
│    [!] ses_181603040f  Explore codeba   │  Directory:    /mnt/external/Khip/campus/...         │
│    ────────────────────────────────  │  Status:       ORPHAN — directory does not exist    │
│    [ ] ses_102d0b637f  Opencode sess   │                                                       │
│    [ ] ses_10cee4094f  Audit Linux-s   │  Agent:        build                                  │
│    [ ] ses_10d377caaf  Opencode Disc   │  Project ID:   19442b1bcfca0cf7b504c1bad92782d792...   │
│    ...                                │  Subagent:     no                                     │
│                                      │  Archived:     no                                     │
│                                      │                                                       │
│                                      │  Created:      2026-05-31 22:20:24 UTC                │
│                                      │  Updated:      2026-06-24 18:42:11 UTC                │
│                                      │                                                       │
│                                      │  Tip: use Phase 1 (project_id match) or manual     │
│                                      │       remap to restore this session.                │
├──────────────────────────────────────┴──────────────────────────────────────────────────────┤
│ 4 orphans, 30 active                                                                       │
│ ↑/k up · ↓/j down · / filter · w watch · r relink · m manual · x migrate · q quit         │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

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

### Requirements

- Go 1.22+ (tested on 1.26)
- No CGO required (pure-Go SQLite via `modernc.org/sqlite`)

## Usage

```sh
opencode-sm                          # Launch TUI with default DB path
opencode-sm --db-path /path/to/db    # Use a specific opencode.db
opencode-sm --watch                  # Auto-refresh when opencode.db changes
opencode-sm --version                # Print version
opencode-sm --help                   # Print help
```

The TUI launches in alt-screen mode. Press `?` for the full keybinding cheatsheet
or `q` to quit.

## Keybindings

### List view

| Key | Action |
|---|---|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `tab` / `→` / `l` | Next tab (Orphans / Active) |
| `shift+tab` / `←` / `h` | Previous tab |
| `/` | Filter list (live fuzzy) |
| `r` | Relink selected session (Phase 1 or Manual) |
| `m` | Manual path remap |
| `x` | Cross-project migrate |
| `w` | Toggle watch mode |
| `?` | Toggle full help |
| `q` / `ctrl+c` | Quit |

### Modal flow

| Key | Action |
|---|---|
| `↑` / `↓` | Choose option / navigate |
| `enter` | Confirm |
| `y` | Yes |
| `n` / `esc` | Cancel |
| `backspace` | Up directory (in filepicker) |

## Relink Flow

The core feature. From any session in the Orphans tab:

1. Press `r` → choose Phase 1 (auto, project_id) or Manual path remap
2. **Phase 1** looks up the session's `project_id` in the current `project` table
   and uses that project's `worktree` as the new directory.
3. **Manual** opens a directory picker; pick any directory on disk.
4. If OpenCode is currently running, a warning appears with the running PIDs and
   offers to send SIGTERM before continuing (optional).
5. A confirmation modal shows old path → new path + a backup notice.
6. On confirm, a backup is written to `<db>.opencode-sm-backup`, then the update
   is applied in a single transaction.

The list refreshes automatically after every apply.

## Migrate Flow

For moving N most recent sessions from one project to another:

1. Press `x` on any session
2. Pick the target project directory (must already exist as a `project.worktree`)
3. Preview the N sessions that will be migrated (default N=5)
4. Confirm to apply — both `session.directory` and `session.project_id` are updated

Sessions already in the target project are skipped automatically.

## Safety

- Every destructive operation creates a backup at `<db>.opencode-sm-backup`
  before writing. The previous backup is overwritten (single safety copy).
- All writes use SQL transactions for atomicity.
- Only `session.directory` (and `session.project_id` for migrate) are modified.
  Conversation content in `session_message` and `part` is never touched.
- Running-instance detection is advisory — you can proceed without killing,
  because SQLite handles concurrent access via file locking.
- The relinker API is exported from `internal/relinker` and can be invoked
  programmatically without the TUI if you need a scriptable recovery path.

## Status

This is **v0.x software** — feature-complete enough for daily use, but APIs
and database formats may change before 1.0. Pin to a tag in production.

| Milestone | Status |
|---|---|
| 1. Foundation scaffold + CLI | done |
| 2. SQLite layer + Phase 1 relinker | done |
| 3. TUI MVP (list, detail, tabs) | done |
| 4. Interactive relink modal stack | done |
| 5. Cross-project migrate + filter | done |
| 6. Polish (watch, CI, lint, README) | done |
| 7. Release binary + goreleaser | pending |

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
