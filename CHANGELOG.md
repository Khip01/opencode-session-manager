# Changelog

All notable changes to `opencode-session-manager` are documented
in this file. Versions follow [Semantic Versioning](https://semver.org/).

Format adapted from [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Changed
- **Layout restructured to a 2x2 grid.** The previous layout had a
  single left/right split with metadata and chat stacked inside one
  tall right panel, plus a global footer at the bottom for help
  hints. The new layout uses four bordered panels: top-left is the
  session list (80% of left column height), bottom-left is the
  keybinding hints (20%), top-right is session metadata (30% of
  right column height), bottom-right is the scrollable chat
  preview (70%). The global footer is gone; the hints panel
  replaces it so shortcuts are always visible without competing
  with chat for vertical space.
- **Chat preview is now scrollable.** The bottom-right panel wraps
  a Bubble Tea `viewport.Model`, so long chat histories can be
  scrolled with mouse wheel or arrow keys. Previously long lines
  overflowed horizontally and were clipped by the panel border.

### Fixed
- Terminal never received mouse events at all, so the chat preview
  scroll-by-hover-routing logic never had any input to route. The
  previous build relied on Bubble Tea v2's default behaviour of
  asking the terminal to send mouse reports, which v2 only does when
  the View's MouseMode is set explicitly (v1 returned
  `tea.EnableMouseCellMotion` from Init; v2 has no such Cmd and the
  View must set MouseMode itself). Fix: set
  `v.MouseMode = tea.MouseModeCellMotion` on the base View returned
  by the TUI so the runtime emits `CSI ?1006h` and the terminal
  starts sending click, release, and wheel events.
- Chat preview PageUp / PageDown / Home / End keys were defined
  but did not scroll the chat viewport. Two compounding bugs:
  (a) the keybinding strings used the literal words `pagedown` /
  `pageup` but Bubble Tea v2 renders the KeyPgDown / KeyPgUp codes
  as `pgdown` / `pgup`, so `key.Matches` returned false and the
  cases never fired; (b) `handleListKey` had a value receiver, so
  even when the cases fired the viewport mutations happened on a
  throwaway copy of the model. Fix: rename bindings to `pgdown` /
  `pgup` and switch the receiver to `*model`. Test coverage added
  in `internal/tui/chat_scroll_test.go` asserts YOffset actually
  changes after each handled key.
- Help bar at the bottom of the TUI was invisible because the help
  text used default styles (dark-key #626262, dark-desc #4A4A4A)
  on a dark background (#161b22 panel). Overridden in `newModel`
  to use accent (blue) for key names and foreground (off-white)
  for descriptions, with muted separator. The keybinding hints are
  now clearly readable on any terminal with a dark palette.
- `-db-path` (and any other string flag) given without a value now
  prints a clean focused error message instead of Go's noisy
  `flag needs an argument: -db-path` plus a full usage dump. Pre-
  scan of `os.Args` in `detectMissingValue` catches the common
  `-flag` mistake before `flag.Parse` runs.
- Chat preview now skips messages that have no text parts (e.g.
  tool calls, step boundaries, reasoning-only). Previously these
  rendered as bare role labels with no content, which looked like
  empty rows in the panel.
- Chat preview text now wraps to fit the detail panel width via
  `lipgloss.Width`. Previously long lines would overflow the panel
  horizontally and the viewport would clip them on the right.
- Body height reserved for the footer increased from `m.height - 6`
  to `m.height - 8` so the help bar is never clipped on terminals
  whose height happens to land exactly on the threshold.
- Chat preview limit is now 10 most recent messages (was 6) and
  3 lines per message (was 6), so users see more of the recent
  conversation without losing the scannable summary feel.
- **Value receiver bug (critical fix).** The entire chat-scroll
  fix chain described above was technically correct at the unit
  test level but **never worked in the interactive TUI** because
  `View(m model)` and several helper methods (`updateMouseFromMsg`,
  `forwardToComponents`, `applyLoaded`) used value receivers. Any
  mutation to the chat viewport inside View was silently discarded
  because View only operates on a value-receiver copy. Additionally,
  `updateMouseFromMsg(msg tea.MouseMsg)` stored mouse coordinates
  on a throwaway copy and `forwardToComponents` returned only a
  Cmd, discarding model state changes. Fix: change `View` and all
  Update-path helpers to `*model` pointer receivers; introduce
  `syncViewportContent` (called from `handleWindowSize` and
  `refreshDetail`) to mutate the viewport in the Update path where
  the pointer lives; remove all `SetWidth`/`SetHeight`/`SetContent`
  calls from `renderRightColumn` (View function) so the viewport
  is only written from Update, not read from a copy.
  Identified with help from Gemini 3.1 Pro analysis.

## [0.1.0-alpha.4] - 2026-06-26

### Added
- Chat preview in the detail panel. When a session is selected, the
  bottom of the right panel shows the most recent messages (up to 6)
  with role labels (USER / ASSISTANT) and text content. Backed by a
  new `internal/db/messages.go` package with `Message`, `MessagePart`
  types and a `ListMessages` query that joins `message` and `part`
  tables and parses the JSON `data` columns with a tiny
  `jsonStringField` helper (no extra dependency).

### Fixed
- Header layout had version string and DB path jammed together with
  no gap because `lipgloss.JoinHorizontal` just concatenates. Now
  the right side (`DB:` plus watching indicator) is placed at the
  far right of the terminal width via `lipgloss.PlaceHorizontal`,
  leaving a visible gap between the version badge and the path.
- Status / help bar at the bottom of the TUI was using muted text on
  panel background and was too easy to miss. Now uses foreground
  text with `Bold(true)` so the keybinding hints are clearly visible.

## [0.1.0-alpha.4] - 2026-06-26

### Fixed
- Release install URLs in published body now have correct `v`
  prefix. Goreleaser's `{{ .Version }}` template renders the tag
  without the leading `v` (e.g. `0.1.0-alpha.3`), but
  `raw.githubusercontent.com` and `go install` URLs require it.
  `.goreleaser.yml` now uses `{{ .Tag }}` for raw URLs so they
  render as `v0.1.0-alpha.3` correctly. Also added a
  `normalize_version` helper in both install scripts so users
  who pass `--version 0.1.0-alpha.3` (without `v`) get the
  correct download URL after normalization.
- Existing `v0.1.0-alpha.3` release body on GitHub edited
  retroactively via `gh release edit` so users copying install
  commands from that release page get working URLs.

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
- Live list filter (`/`).
- Watch mode for auto-refresh when opencode.db changes (`w`).
- Running-instance detection with optional SIGTERM.
- Auto-backup before every write (`<db>.opencode-sm-backup`).
- Install and uninstall scripts (bash + PowerShell) with shell
  completions for bash, zsh, fish, and PowerShell.
- GitHub Actions CI matrix across Ubuntu, macOS, and Windows.
- Multi-platform release artifacts via goreleaser.
