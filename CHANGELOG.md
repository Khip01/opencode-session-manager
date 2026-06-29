# Changelog

All notable changes to `opencode-session-manager` are documented
in this file. Versions follow [Semantic Versioning](https://semver.org/).

Format adapted from [Keep a Changelog](https://keepachangelog.com/).

## [0.1.0-alpha.4] - 2026-06-26

### Added
- **Markdown renderer** (`internal/tui/markdown.go`): bold, italic,
  inline code, fenced code blocks, links, headings, lists, blockquotes.
- **Scrollbar** on the chat preview panel (blue thumb, gray track).
- **Fixed "Chat Preview" header** — stays at the top while chat content
  scrolls beneath.
- **Layout responsive to terminal height** — fixed 7-row hints panel,
  remaining space distributed.
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
- **Layout rebuilt from scratch** matching `test-tui-multiple-window-go`.
  Panels now use `lipgloss.Place` + explicit `Width`/`Height` so they
  never extend past the terminal bottom. Left/right columns always have
  identical total height.
- **Panel backgrounds simplified** — all panels use terminal default
  background (no explicit `Background`). Only user chat messages retain
  a subtle `#1e1e1e` background.
- **Chat messages**: unescaped `\n`, `\"`, `\\`. No longer truncated
  (unlimited lines). User messages have left-only double blue border
  with subtle background, full panel width.
- **Hints panel** uses `ShortHelpView` (compact) instead of `FullHelpView`.
  `?`/`/` help key removed (full keymap always visible).
- **Watch mode removed** — `--watch` flag, `watch.go`, and keybinding
  deleted (was non-functional).
- **Detail panel padding** adjusted: `PaddingTop(1)+PaddingBottom(1)+
  PaddingLeft(2)` so the scrollbar touches the right border.

### Fixed
- **Alt-screen break on modal open**: modal views now set
  `v.AltScreen = true` via `tuiView()` helper; `r`/`m`/`x` keys no
  longer break out of fixed viewport mode.
- **Chat scroll**: `SetContent`/`GotoBottom` moved from View path
  (value receiver) to Update path (pointer receiver).
- **Responsive resize**: `m.list.SetWidth/SetHeight` moved from
  `renderBody` (value receiver) to `handleWindowSize` (pointer).
- **Panel borders invisible**: exact height calculation with proper
  border/padding accounting so borders are fully visible.
- **Text indentation removed**: no leading spaces in paragraphs or
  assistant messages.
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
- **Bottom-right chat panel rendering at height 0** (the "mendelep"
  panel bug). `renderBody` passed the zero-valued `m.rightBotH`
  model field to `makePanelChat` instead of the locally computed
  `rightBotH` variable. The field is never assigned anywhere in
  the codebase, so the chat panel was always rendered at height 0
  and the right column appeared shorter than the left column.
  Fix: use the local `rightBotH` variable. Added regression test
  `TestRenderBody_DoesNotReadRightBotHField` in
  `internal/tui/render_body_test.go`.
- **Panel helpers ignored long-line wrapping** when content was
  taller than its container. `makePanelPad` and `makePanelChat`
  applied `MaxHeight` to the raw content before wrapping it to the
  panel's inner width, so any line wider than the panel was wrapped
  to multiple rows AFTER the height clip and the resulting block no
  longer fit inside `Height(h)`. The body ended up overflowing the
  terminal by 18 rows on a 120x40 terminal. Fix: wrap content to
  the inner content width first, then clip to the inner content
  height. Added regression tests `TestRenderBody_FitsAvailableHeight`,
  `TestMakePanelPad_HonorsHeight`, and `TestMakePanelChat_HonorsHeight`
  in `internal/tui/render_body_test.go`.

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
