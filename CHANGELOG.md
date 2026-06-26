# Changelog

All notable changes to `opencode-session-manager` are documented
in this file. Versions follow [Semantic Versioning](https://semver.org/).

Format adapted from [Keep a Changelog](https://keepachangelog.com/).

## [0.1.0-alpha.4] - 2026-06-26

### Added
- **Interactive relink flow.** Press `r` on any orphan session to
  open a modal with two strategies: Phase 1 (auto project_id match
  derived from git root commit hash) and Manual path remap (Bubbles
  filepicker to browse the filesystem). Includes confirmation dialog
  with backup notice, running-instance detection (Linux /proc), and
  optional SIGTERM kill before destructive writes.
- **Chat preview** in the detail panel. When a session is selected,
  the bottom of the right panel shows the most recent messages (up
  to 6) with role labels (USER / ASSISTANT) and text content. Backed
  by `internal/db/messages.go` with `Message`, `MessagePart` types
  and a `ListMessages` query that joins `message`+`part` tables.
- **Scrollable chat viewport.** The chat preview wraps a Bubble Tea
  `viewport.Model` so long histories can be scrolled with mouse
  wheel or arrow keys. PageUp/PageDown/Home/End keys supported.

### Changed
- **Layout restructured to a 2x2 grid.** Top-left: session list
  (80% of left column). Bottom-left: keybinding hints (20%).
  Top-right: session metadata (30% of right column). Bottom-right:
  scrollable chat preview (70%). Global footer replaced by the
  hints panel so shortcuts stay visible without competing with
  chat for vertical space.
- Status / help bar now uses foreground bold text instead of muted
  on panel background, improving visibility on dark terminals.
- `-db-path` (and any flag) without a value now prints a focused
  error instead of Go's default noisy flag+usage dump.
- Chat preview limit: 10 most recent messages (was 6), 3 lines per
  message (was 6) for better scanability.

### Fixed
- **Value receiver bug (critical).** `View(m model)` and several
  Update helpers (`updateMouseFromMsg`, `forwardToComponents`,
  `applyLoaded`) used value receivers, so all chat-viewport
  mutations were silently discarded. Fix: change to `*model`
  pointer receivers everywhere in the Update path; introduce
  `syncViewportContent` called from `handleWindowSize` and
  `refreshDetail` to write viewport only from Update, not from
  View. (Identified by Gemini 3.1 Pro analysis.)
- Mouse events never emitted because Bubble Tea v2's View must
  set `MouseMode` explicitly (v1's `tea.EnableMouseCellMotion`
  from Init no longer works). Fix: set
  `v.MouseMode = tea.MouseModeCellMotion` on the base View.
- Chat-scroll keybindings used wrong string values: `pagedown`/
  `pageup` instead of Bubble Tea v2's actual `pgdown`/`pgup`,
  so `key.Matches` never fired. Renamed bindings; added test
  coverage in `chat_scroll_test.go`.
- Help bar text used default Bubble styles (dark-key #626262,
  dark-desc #4A4A4A) on dark panel background (#161b22) and was
  invisible. Overridden to accent blue for keys, foreground
  off-white for descriptions.
- Chat preview skipped messages with no text parts (tool calls,
  reasoning steps). Text now wraps to fit panel width via
  `lipgloss.Width` instead of overflowing horizontally.
- Header: version label and DB path now placed at opposite ends
  via `lipgloss.PlaceHorizontal` instead of jammed together.
- Body height reserved for footer increased by `+2` so the help
  bar is never clipped on tight terminal sizes.
- Release install URLs in published body now use `{{ .Tag }}`
  (with `v` prefix) instead of `{{ .Version }}` (without it).
  Added `normalize_version` helper in install scripts.
- Install scripts now prefer user-local writable PATH dirs
  (`~/.local/bin`, `~/go/bin`, `~/bin`) before falling back to
  system dirs, fixing `Permission denied` for non-root users.

## [0.1.0-alpha.3] - 2026-06-26

### Fixed
- Install scripts (`install.sh` and `install.ps1`) now check
  **writable** directories before selecting an install location.
  Previously they picked the first PATH directory, which on Linux
  and macOS is `/usr/local/bin`. That directory requires `sudo`,
  so non-root users hit `Permission denied` during `cp`. The fix
  prefers user-local writable dirs (`~/.local/bin`, `~/go/bin`,
  `~/bin`) and only falls back to system dirs if they are writable.
  Users who want a system-wide install can run the installer with
  `sudo` or use `--prefix DIR` explicitly.

## [0.1.0-alpha.2] - 2026-06-25

### Added
- `opencode-sm uninstall` subcommand for self-uninstall. Accepts
  `--prefix DIR`, `--purge`, and `--dry-run` flags. Available on
  Linux, macOS, and Windows.
- `scripts/uninstall.sh` and `scripts/uninstall.ps1` now delegate
  to `opencode-sm uninstall` when the binary is on PATH, so behavior
  stays consistent with the subcommand.
- Both `opencode-sm uninstall` (subcommand) and `opencode-sm
  --uninstall` (flag) forms are accepted.
- Project-level `AGENTS.md` documenting tech stack, repository
  layout, critical paths, and common pitfalls for future agents.

### Fixed
- Install scripts (`install.sh` and `install.ps1`) now use the
  `/releases` plural API endpoint instead of `/releases/latest`.
  The latter returns 404 when only pre-release tags exist, which
  broke install on repos that have not yet cut a stable release.

### Known issues
- The installer still fails for non-root users on Linux/macOS
  because it tries to write to `/usr/local/bin` (permission
  denied). Workaround: use `--prefix ~/.local/bin` or run with
  `sudo`. Fixed in `0.1.0-alpha.3`.

## [0.1.0-alpha.1] - 2026-06-25

### Added
- Initial pre-release.
- TUI for browsing, recovering, and relocating OpenCode sessions.
- Phase 1 (project_id) auto-recovery for renamed git repos.
- Manual path remap with directory picker.
- Cross-project migrate (N most recent sessions).
- Watch mode for auto-refresh when opencode.db changes (`w`).
- Running-instance detection with optional SIGTERM.
- Auto-backup before every write (`<db>.opencode-sm-backup`).
- Install and uninstall scripts (bash + PowerShell) with shell
  completions for bash, zsh, fish, and PowerShell.
- GitHub Actions CI matrix across Ubuntu, macOS, and Windows.
- Multi-platform release artifacts via goreleaser.
