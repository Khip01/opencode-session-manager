#!/usr/bin/env bash
#
# opencode-sm uninstaller (bash), for Linux and macOS
#
# Usage:
#   ./uninstall.sh                 # remove from standard locations
#   ./uninstall.sh --prefix DIR    # remove from specific directory
#   ./uninstall.sh --dry-run       # show what would be removed
#   ./uninstall.sh --purge         # also remove ~/.config/opencode-sm if present
#   ./uninstall.sh --help          # show help

set -euo pipefail

readonly BINARY="opencode-sm"

PREFIX=""
DRY_RUN=false
PURGE=false

show_help() {
    cat <<EOF
opencode-sm uninstaller (bash)

USAGE:
  $0 [options]

OPTIONS:
  --prefix DIR          Remove from DIR instead of scanning standard locations
  --purge               Also remove user config (if any)
  --dry-run             Show what would be removed without removing
  -h, --help            Show this help

REMOVED FILES:
  - The opencode-sm binary from the install location
  - (with --purge) ~/.config/opencode-sm/ if present

NOTE:
  This script does NOT remove:
  - Shell history (check ~/.bash_history, etc.)
  - Backups created by opencode-sm itself
  - opencode.db or any OpenCode data

EOF
}

log() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; }
die() { err "$@"; exit 1; }

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --prefix)  PREFIX="$2"; shift 2 ;;
            --purge)   PURGE=true; shift ;;
            --dry-run) DRY_RUN=true; shift ;;
            -h|--help) show_help; exit 0 ;;
            *) die "unknown argument: $1 (use --help)" ;;
        esac
    done
}

remove_file() {
    local path="$1"
    if [[ -e "$path" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log "[dry-run] would remove: $path"
        else
            if [[ ! -w "$(dirname "$path")" ]]; then
                err "cannot remove $path (permission denied)"
                return 1
            fi
            rm -f "$path"
            log "removed: $path"
            return 0
        fi
    fi
    return 0
}

remove_dir_if_empty() {
    local dir="$1"
    [[ -d "$dir" ]] || return 0
    if [[ -z "$(ls -A "$dir" 2>/dev/null)" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log "[dry-run] would remove empty dir: $dir"
        else
            rmdir "$dir" 2>/dev/null || true
            log "removed empty dir: $dir"
        fi
    fi
}

scan_locations() {
    local candidates=(
        "/usr/local/bin/$BINARY"
        "$HOME/.local/bin/$BINARY"
        "$HOME/go/bin/$BINARY"
        "$HOME/bin/$BINARY"
        "/opt/homebrew/bin/$BINARY"
    )

    local removed=0
    for path in "${candidates[@]}"; do
        if [[ -e "$path" ]]; then
            if remove_file "$path"; then
                removed=$((removed + 1))
            fi
        fi
    done

    # Only remove empty parent dirs if --prefix wasn't set
    if [[ -z "$PREFIX" && "$removed" -gt 0 ]]; then
        remove_dir_if_empty "$HOME/.local/bin"
    fi

    if [[ "$removed" -eq 0 && "$DRY_RUN" != true ]]; then
        log ""
        log "opencode-sm not found in standard locations"
        log "If installed elsewhere, remove manually"
    fi
}

remove_from_prefix() {
    local target="$PREFIX/$BINARY"
    if [[ ! -e "$target" ]]; then
        err "no opencode-sm found at $target"
        return 1
    fi
    remove_file "$target"
}

purge_config() {
    local config_dir="$HOME/.config/opencode-sm"
    if [[ -d "$config_dir" ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log "[dry-run] would purge: $config_dir"
        else
            rm -rf "$config_dir"
            log "purged: $config_dir"
        fi
    else
        log "no user config at $config_dir"
    fi
}

main() {
    parse_args "$@"

    log "opencode-sm uninstaller"
    log ""

    if [[ "$DRY_RUN" == true ]]; then
        log "(DRY RUN, nothing will be removed)"
        log ""
    fi

    if [[ -n "$PREFIX" ]]; then
        remove_from_prefix
    else
        scan_locations
    fi

    if [[ "$PURGE" == true ]]; then
        purge_config
    fi

    if [[ "$DRY_RUN" != true ]]; then
        log ""
        log "uninstall complete"
        log ""
        log "Note: this script does not touch:"
        log "  - Backups (*.opencode-sm-backup) in the same dir as opencode.db"
        log "  - opencode.db or any OpenCode data"
    fi
}

main "$@"
