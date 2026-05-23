#!/usr/bin/env sh
#
# Wave install script.
#
# Detects the host OS + arch, downloads the matching tarball from the
# latest GitHub release, extracts the binary, and drops it into
# /usr/local/bin. Run as:
#
#   curl -sSfL https://luowensheng.github.io/wave/install.sh | sh
#
# or with a pinned version:
#
#   curl -sSfL https://luowensheng.github.io/wave/install.sh | sh -s -- v0.1.0
#
# Options:
#   INSTALL_DIR=path    target directory (default: /usr/local/bin)
#   WAVE_VERSION=vX.Y.Z pin a specific release tag

set -eu

REPO="luowensheng/wave"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${WAVE_VERSION:-}"

# Allow the first positional arg to override the version too:
#   curl … | sh -s -- v0.1.0
if [ "${1:-}" != "" ]; then
  VERSION="$1"
fi

main() {
  need curl
  need tar
  need uname
  need mkdir

  detect_os
  detect_arch
  resolve_version

  ASSET="wave_${VERSION#v}_${OS_LABEL}_${ARCH_LABEL}.${ARCHIVE_EXT}"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
  TMP="$(mktemp -d 2>/dev/null || mktemp -d -t 'wave-install')"
  trap 'rm -rf "$TMP"' EXIT

  echo "Downloading $URL ..."
  curl -fsSL "$URL" -o "$TMP/$ASSET"

  echo "Extracting ..."
  if [ "$ARCHIVE_EXT" = "zip" ]; then
    need unzip
    unzip -q "$TMP/$ASSET" -d "$TMP"
  else
    tar -xzf "$TMP/$ASSET" -C "$TMP"
  fi

  if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating $INSTALL_DIR ..."
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
      echo "Need sudo to create $INSTALL_DIR" >&2
      sudo mkdir -p "$INSTALL_DIR"
    fi
  fi

  echo "Installing to $INSTALL_DIR/wave ..."
  if ! mv "$TMP/wave" "$INSTALL_DIR/wave" 2>/dev/null; then
    echo "Need sudo to write $INSTALL_DIR/wave" >&2
    sudo mv "$TMP/wave" "$INSTALL_DIR/wave"
  fi
  chmod +x "$INSTALL_DIR/wave" 2>/dev/null || sudo chmod +x "$INSTALL_DIR/wave"

  echo
  echo "Wave installed."
  "$INSTALL_DIR/wave" version
  echo
  echo "Next steps:"
  echo "  wave help"
  echo "  https://luowensheng.github.io/wave/guide/quickstart"
}

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}

detect_os() {
  RAW="$(uname -s)"
  case "$RAW" in
    Linux)   OS_LABEL="Linux" ;;
    Darwin)  OS_LABEL="macOS" ;;
    MINGW*|MSYS*|CYGWIN*) OS_LABEL="Windows" ;;
    *) echo "unsupported OS: $RAW" >&2; exit 1 ;;
  esac
}

detect_arch() {
  RAW="$(uname -m)"
  case "$RAW" in
    x86_64|amd64) ARCH_LABEL="x86_64" ;;
    arm64|aarch64) ARCH_LABEL="arm64" ;;
    *) echo "unsupported arch: $RAW" >&2; exit 1 ;;
  esac
  if [ "$OS_LABEL" = "Windows" ]; then
    ARCHIVE_EXT="zip"
  else
    ARCHIVE_EXT="tar.gz"
  fi
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    return
  fi
  # latest stable release
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -m1 '"tag_name"' \
    | sed -E 's/.*"tag_name"[^"]*"([^"]+)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "could not determine latest release. Pin one explicitly: WAVE_VERSION=v0.1.0 sh install.sh" >&2
    exit 1
  fi
}

main "$@"
