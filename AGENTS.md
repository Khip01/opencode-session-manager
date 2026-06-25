# AGENTS.md

Project-specific guidance for AI agents working on
`opencode-session-manager`. Read this before exploring the repository;
the global rules in `~/.config/opencode/AGENTS.md` still apply
(commit format, no emoji, no em dash, ACC before destructive
actions).

## Overview

`opencode-sm` is a TUI for managing, recovering, and relocating
[OpenCode](https://opencode.ai) sessions. The bug it addresses:
OpenCode stores `session.directory` as an absolute path at creation
time and never updates it, so renaming or moving a project folder
makes all its chat sessions invisible in the UI even though the
data is still in the local SQLite database.

This tool reads the database, classifies sessions as orphans
(missing directory) or active (directory present), and lets the
user recover, remap, or migrate them via a Bubble Tea interface.

This is **v0.x alpha**. APIs may change. Backups are created before
every write.

## Tech stack

- Go 1.24 (toolchain auto-upgrade to 1.25+ via `GOTOOLCHAIN`)
- Bubble Tea v2 (`charm.land/bubbletea/v2`) for the TUI
- Bubbles v2 for components (list, viewport, filepicker, help, key)
- Lipgloss v2 for styling
- modernc.org/sqlite for SQLite (pure-Go, no CGO)
- testify for test assertions
- github.com/goreleaser/goreleaser for release packaging

CGO is disabled in release builds (`CGO_ENABLED=0`).

## Repository layout

```
opencode-session-manager/
├── cmd/opencode-sm/             # main package, CLI entry
├── internal/
│   ├── db/                      # SQL queries + types (Session, Project)
│   ├── relinker/                # Phase 1, manual remap, migrate, backup
│   └── tui/                     # Bubble Tea model + modals + views
├── scripts/
│   ├── install.sh               # bash installer (Linux + macOS)
│   ├── uninstall.sh             # bash uninstaller
│   ├── install.ps1              # PowerShell installer (Windows)
│   ├── uninstall.ps1            # PowerShell uninstaller
│   └── completions/             # bash/zsh/fish/pwsh completion files
├── .github/workflows/ci.yml     # CI matrix (ubuntu/macos/windows) + build
├── .goreleaser.yml              # multi-platform release packaging
├── .golangci.yml                # v2 schema, formatters + linters
├── go.mod / go.sum
├── Makefile                     # build/test/install/uninstall targets
├── README.md                    # user-facing docs
├── NOTICE                       # third-party attributions
└── LICENSE                      # MIT
```

`bin/` and `dist/` are gitignored (build outputs).

## Build and test

```sh
make build              # produces ./bin/opencode-sm
make test               # unit tests
make test-race          # tests with race detector
make test-coverage      # produces coverage.html
make tidy               # go mod tidy
make clean              # remove build artifacts
make install PREFIX=$HOME/.local/bin     # build + install (local)
make uninstall PREFIX=$HOME/.local/bin   # remove
make install-release    # download latest via scripts/install.sh
make uninstall-release  # run scripts/uninstall.sh
make help               # list all targets
```

Direct Go:

```sh
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build ./...
```

## Critical paths

These are safety-critical areas. Modifying them requires extra
care and explicit user ACC.

### `internal/relinker/backup.go`

`Backup()` creates `<dbpath>.opencode-sm-backup` before every
write. It must:
- Overwrite any existing backup (single safety copy)
- Use `io.Copy` to preserve bytes exactly
- `Sync()` the file before close so data is durable

If you change this, verify by running `make test-race` and a
manual cycle: install binary, run `opencode-sm`, do a relink,
check that `<db>.opencode-sm-backup` exists and contains the
pre-write state.

### `internal/relinker/migrate.go` and `internal/relinker/manual.go`

These run `UPDATE session SET directory = ?, project_id = ?`
inside transactions. The transaction must:
- Begin with `BeginTx(ctx, nil)`
- Use a prepared statement
- Commit at the end, Rollback on error
- Filter out no-op rows (already correct project_id) before writing

### `internal/tui/running_check.go`

Linux-only: scans `/proc/[0-9]+/{comm,cmdline}` to find running
`opencode` processes. Used to warn before destructive writes.
Returns empty slice (not nil) on non-Linux so callers always
receive a valid result.

Never panic in this file. Worst case is missing detection.

### SQLite schema assumptions

We touch these tables/columns:

- `session(id, project_id, parent_id, directory, title, agent, time_created, time_updated, time_archived)`
- `project(id, worktree, vcs, name)`
- `project_directory(project_id, directory)`

We do NOT touch: `session_message`, `part`, `session_share`,
`session_input`, `event`, `todo`, `permission`, `credential`,
`control_account`. If you need them, you're in the wrong layer.

The test fixture in `internal/db/fixture_test.go` and
`internal/relinker/*_test.go` is a simplified schema, not the
real opencode.db. Don't copy fixture assumptions into production
code without verifying against `internal/db/session.go` query
column lists.

## Adding a feature

Workflow for new functionality:

1. Decide which package owns the logic:
   - Pure SQL or types: `internal/db`
   - DB mutations or backups: `internal/relinker`
   - User-facing flow: `internal/tui`
2. Add a method to the existing service struct (Relinker, dataLoader)
   rather than a free function, so it composes with existing
   helpers (`open()`, `Backup()`).
3. Write the test first in the same package, using the in-memory
   fixture. Use `testify/assert` for normal assertions,
   `testify/require` for fatal preconditions.
4. Wire the TUI: add a keybinding in `keys.go`, a mode constant
   in `modal.go`, an Update handler in `update.go`, and a render
   branch in `app.go`.
5. Run `make test-race` and `make build` locally.
6. Show the diff and commit message. Wait for ACC.

## Fixing a bug

1. Write a failing test that reproduces the bug (use the in-memory
   fixture, not the real opencode.db).
2. Fix the production code.
3. Verify the test passes and no other tests regress.
4. Show the diff and commit message. Wait for ACC.

## UI conventions

- Modal state machine: `modeID` enum in `internal/tui/modal.go`.
  Add a new mode by adding a constant, a label, a dispatcher
  case in `update.go`, and a render branch in `app.go`.
- Keybindings live in `keys.go`. Each binding has `WithKeys` and
  `WithHelp`. Add to both `listShortHelp` and the relevant full
  help group.
- Strings displayed to the user must be ASCII safe. No em dash
  (use `-` or `,` instead). No emoji. Use plain words like `OK`,
  `WARN`, `ERR` instead of glyphs.
- Color palette is in `internal/tui/styles.go` as `opencodePalette()`.
  Extend there rather than hard-coding colors in views.

## Commit format

Conventional Commits. Use `Feat:`, `Fix:`, `Chore:` section
headers. End with `See CHANGELOG.md for full details.` See the
global `~/.config/opencode/AGENTS.md` for the full format.

Example:

```
feat: add cross-platform install and uninstall scripts

Feat:
- scripts/install.sh: bash installer for Linux and macOS
- scripts/uninstall.sh: bash uninstaller

Chore:
- Makefile: add install and uninstall targets
- README: rewrite Installation section

See CHANGELOG.md for full details.
```

## Common pitfalls

- **Em dash in code**: `internal/tui/*.go` strings are user-visible
  and must use `-` or `,` instead of the long dash character. Audit before commit.
- **Emoji in shell output**: scripts/install.sh etc. must use
  plain words (`OK`, `WARN`, `ERR`) for status prefixes.
- **Bubbles v2 API**: import path is `charm.land/bubbles/v2`, not
  `github.com/charmbracelet/bubbles/v2`. The latter only has v1.
- **Lipgloss v2 `Color`**: `Color` is a function (`func(s string) color.Color`),
  not a type. Use `color.Color` interface for fields.
- **modernc.org/sqlite**: file must exist before `db.Open()`
  or it errors with `ErrDBNotFound`. For tests, use
  `sql.Open("sqlite", ":memory:")` or pre-create the file.
- **`r.open()`**: the `Relinker.open()` method takes no context
  (lint complains if unused). Call `r.open()`, not `r.open(ctx)`.
- **`runtime.GOOS != "linux"`**: returning `nil` from a
  `[]T`-returning function breaks callers that assert
  `NotNil(t, slice)`. Return `make([]T, 0)` instead.
- **Cross-platform path tests**: don't hardcode `/tmp/...`.
  Use `filepath.Join(os.TempDir(), ...)` or `t.TempDir()`.

## References

- Issue [anomalyco/opencode#27822](https://github.com/anomalyco/opencode/issues/27822):
  upstream feature request for auto-relink. Assigned to
  `@kitlangton`. This tool exists because that issue is still open.
- [bbl21/opencode-session-recovery](https://github.com/bbl21/opencode-session-recovery):
  the Python predecessor. Our Phase 1 and Phase 2 algorithms are
  adapted from `scripts/recover_sessions.py`. Attribution in NOTICE.
- Bubble Tea v2 docs: https://charm.land/bubbletea/v2
- modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite
- Release process: `.goreleaser.yml` produces tarballs for
  linux/darwin/windows on amd64/arm64 (except windows/arm64).

## Git workflow reminders

- Always show proposed commit message + diff stat before commit.
- Wait for explicit ACC before `git commit`, `git push`,
  `git tag`, or `goreleaser release`.
- Tag format: `vX.Y.Z` for stable, `vX.Y.Z-alpha.N` or
  `vX.Y.Z-beta.N` for pre-release.
- Release notes are generated by goreleaser from Conventional
  Commits grouped by `feat`/`fix`/other.
