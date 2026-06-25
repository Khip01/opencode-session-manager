#!/usr/bin/env bash
#
# opencode-sm installer (bash), for Linux and macOS
#
# Usage:
#   ./install.sh                         # install latest
#   ./install.sh --version v0.1.0-alpha.1 # install specific version
#   ./install.sh --from-local ./dist/foo.tar.gz  # install from local tarball
#   ./install.sh --prefix ~/.local/bin   # install to specific directory
#   ./install.sh --dry-run               # show what would happen
#   ./install.sh --help                  # show help
#
# Environment variables:
#   OPENCODE_SM_INSTALL_DIR   override install directory
#   OPENCODE_SM_VERSION       override version (default: latest)

set -euo pipefail

readonly REPO="Khip01/opencode-session-manager"
readonly BINARY="opencode-sm"
readonly GITHUB_API="https://api.github.com"

VERSION="${OPENCODE_SM_VERSION:-}"
INSTALL_DIR="${OPENCODE_SM_INSTALL_DIR:-}"
LOCAL_FILE=""
DRY_RUN=false
NO_PATH_CHECK=false

show_help() {
    cat <<EOF
opencode-sm installer (bash)

USAGE:
  $0 [options]

OPTIONS:
  --version VERSION     Install specific version (e.g. v0.1.0-alpha.1)
  --from-local PATH     Install from local tarball instead of downloading
  --prefix DIR          Install to DIR (default: auto-detect)
  --dry-run             Show what would happen without installing
  --no-path-check       Skip PATH warning if install dir not in PATH
  -h, --help            Show this help

ENVIRONMENT:
  OPENCODE_SM_VERSION    Same as --version
  OPENCODE_SM_INSTALL_DIR Same as --prefix

EXAMPLES:
  $0                                       # Install latest from GitHub
  $0 --version v0.1.0-alpha.1              # Install specific version
  $0 --from-local ./opencode-sm.tar.gz     # Install from local file
  $0 --prefix ~/.local/bin                 # Install to ~/.local/bin
  $0 --dry-run                             # Preview without installing

EOF
}

log() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; }
die() { err "$@"; exit 1; }

detect_os_arch() {
    local raw_os raw_arch
    raw_os=$(uname -s)
    raw_arch=$(uname -m)

    case "$raw_os" in
        Linux)  OS="linux" ;;
        Darwin) OS="darwin" ;;
        *) die "unsupported OS: $raw_os (use install.ps1 for Windows)" ;;
    esac

    case "$raw_arch" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)  ARCH="arm64" ;;
        *) die "unsupported architecture: $raw_arch" ;;
    esac

    if [[ "$OS" == "linux" && "$ARCH" == "arm64" ]]; then
        : # supported
    fi
}

capitalize() {
    printf '%s' "$1" | awk '{ printf toupper(substr($0,1,1)) substr($0,2) }'
}

resolve_latest_version() {
    local json
    if ! json=$(curl -fsSL "$GITHUB_API/repos/$REPO/releases?per_page=20"); then
        die "could not fetch releases from GitHub"
    fi

    # /releases (plural) returns ALL non-draft releases including
    # pre-releases, sorted by created_at desc. We pick the first
    # tag_name. Note: /releases/latest ignores pre-releases which
    # is why we cannot use it for alpha/beta tags.
    local tag
    tag=$(printf '%s' "$json" \
        | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -1)

    [[ -n "$tag" ]] || die "could not determine latest version from GitHub (no releases found)"
    printf '%s\n' "$tag"
}

build_download_url() {
    local v="$1"
    local v_stripped="${v#v}"
    local os_cap
    os_cap=$(capitalize "$OS")
    printf 'https://github.com/%s/releases/download/%s/opencode-session-manager_%s_%s_%s.tar.gz\n' \
        "$REPO" "$v" "$v_stripped" "$os_cap" "$ARCH"
}

build_checksum_url() {
    local v="$1"
    local v_stripped="${v#v}"
    printf 'https://github.com/%s/releases/download/%s/opencode-session-manager_%s_checksums.txt\n' \
        "$REPO" "$v" "$v_stripped"
}

determine_install_dir() {
    if [[ -n "$INSTALL_DIR" ]]; then
        mkdir -p "$INSTALL_DIR"
        printf '%s\n' "$INSTALL_DIR"
        return
    fi

    local candidates=(
        "/usr/local/bin"
        "$HOME/.local/bin"
        "$HOME/go/bin"
        "$HOME/bin"
    )

    for dir in "${candidates[@]}"; do
        if [[ ":$PATH:" == *":$dir:"* ]]; then
            mkdir -p "$dir"
            printf '%s\n' "$dir"
            return
        fi
    done

    local fallback="$HOME/.local/bin"
    mkdir -p "$fallback"
    printf '%s\n' "$fallback"
}

sha256_of() {
    sha256sum "$1" | awk '{print $1}'
}

verify_checksum() {
    local archive="$1" checksum_file="$2" archive_name="$3"
    local expected actual

    [[ -s "$checksum_file" ]] || { log "checksums file empty, skipping verification"; return 0; }

    expected=$(grep -E "[[:space:]]${archive_name}$" "$checksum_file" | awk '{print $1}' | head -1)

    if [[ -z "$expected" ]]; then
        log "no checksum entry for $archive_name, skipping verification"
        return 0
    fi

    actual=$(sha256_of "$archive")
    if [[ "$expected" != "$actual" ]]; then
        err "checksum mismatch for $archive_name"
        err "  expected: $expected"
        err "  actual:   $actual"
        return 1
    fi
    log "checksum verified (sha256)"
}

install_binary() {
    local archive="$1"
    local target="$INSTALL_DIR/$BINARY"

    log "installing to $target"

    if [[ "$DRY_RUN" == true ]]; then
        log "[dry-run] would copy $archive -> $target"
        log "[dry-run] would chmod 755 $target"
        return 0
    fi

    if [[ -e "$target" && ! -w "$target" ]]; then
        die "cannot write to $target (permission denied); try --prefix for user dir"
    fi

    cp "$archive" "$target"
    chmod 755 "$target"
    log "installed: $target"
}

warn_path() {
    if [[ "$NO_PATH_CHECK" == true ]]; then
        return
    fi
    if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
        return
    fi

    log ""
    log "WARNING: $INSTALL_DIR is not in your PATH"
    log ""
    log "Add this line to your shell rc file (~/.bashrc, ~/.zshrc, etc.):"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
    log ""
    log "Or run opencode-sm with full path:"
    log "  $INSTALL_DIR/$BINARY --version"
}

print_success() {
    log ""
    log "opencode-sm $VERSION installed to $INSTALL_DIR/$BINARY"
    log ""
    if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
        log "Run:"
        log "  $BINARY --version"
    else
        warn_path
    fi
}

TMPDIR_GLOBAL=""

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)      VERSION="$2"; shift 2 ;;
            --from-local)   LOCAL_FILE="$2"; shift 2 ;;
            --prefix)       INSTALL_DIR="$2"; shift 2 ;;
            --dry-run)      DRY_RUN=true; shift ;;
            --no-path-check) NO_PATH_CHECK=true; shift ;;
            -h|--help)      show_help; exit 0 ;;
            *) die "unknown argument: $1 (use --help)" ;;
        esac
    done
}

main() {
    parse_args "$@"
    detect_os_arch

    if [[ -z "$VERSION" && -z "$LOCAL_FILE" ]]; then
        VERSION=$(resolve_latest_version)
    fi
    [[ -n "$VERSION" ]] || die "version is required"

    TMPDIR_GLOBAL=$(mktemp -d)
    trap 'cleanup' EXIT

    local archive_name
    archive_name="opencode-session-manager_${VERSION#v}_$(capitalize "$OS")_$ARCH.tar.gz"
    local archive="$TMPDIR_GLOBAL/$archive_name"
    local checksum_file="$TMPDIR_GLOBAL/checksums.txt"

    if [[ -n "$LOCAL_FILE" ]]; then
        log "using local file: $LOCAL_FILE"
        cp "$LOCAL_FILE" "$archive"
        VERSION="${VERSION:-local}"
    else
        local url
        url=$(build_download_url "$VERSION")
        log "downloading $url"
        if ! curl -fsSL -o "$archive" "$url"; then
            die "download failed (network issue or version not found)"
        fi

        local cs_url
        cs_url=$(build_checksum_url "$VERSION")
        if ! curl -fsSL -o "$checksum_file" "$cs_url"; then
            log "checksums file not available, skipping verification"
            : > "$checksum_file"
        fi
        verify_checksum "$archive" "$checksum_file" "$archive_name" || die "checksum verification failed"
    fi

    log "extracting archive"
    tar -xzf "$archive" -C "$TMPDIR_GLOBAL"

    local extracted
    extracted=$(find "$TMPDIR_GLOBAL" -name "$BINARY" -type f | head -1)
    [[ -n "$extracted" ]] || die "binary not found in archive"

    INSTALL_DIR=$(determine_install_dir)

    install_binary "$extracted"
    print_success
}

cleanup() {
    if [[ -n "$TMPDIR_GLOBAL" && -d "$TMPDIR_GLOBAL" ]]; then
        rm -rf "$TMPDIR_GLOBAL"
    fi
}

main "$@"
