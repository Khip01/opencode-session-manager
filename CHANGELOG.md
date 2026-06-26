# Changelog

All notable changes to `opencode-session-manager` are documented
in this file. Versions follow [Semantic Versioning](https://semver.org/).

Format adapted from [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
