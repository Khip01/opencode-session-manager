# Changelog

All notable changes to `opencode-session-manager` are documented
in this file. Versions follow [Semantic Versioning](https://semver.org/).

Format adapted from [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
