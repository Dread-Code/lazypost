#!/usr/bin/env bash
set -euo pipefail

REPO="Dread-Code/lazypost"

usage() {
  cat <<'EOF'
Usage: install.sh [VERSION]

Installs the lazypost binary from GitHub Releases into ~/.local/bin
(or $PREFIX/bin when PREFIX is set, e.g. PREFIX=/usr/local).

VERSION defaults to the latest release (e.g. v0.1.0).
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

VERSION="${1:-}"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)      echo "install.sh: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             echo "install.sh: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
  SHA_CMD=(sha256sum -c -)
else
  SHA_CMD=(shasum -a 256 -c -)
fi

fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$@"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$@"
  else
    echo "install.sh: need curl or wget" >&2
    exit 1
  fi
}

if [ -z "$VERSION" ]; then
  echo "install.sh: resolving latest release..."
  VERSION="$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -n1)"
  [ -n "$VERSION" ] || { echo "install.sh: could not resolve the latest release" >&2; exit 1; }
fi

ASSET="lazypost-$OS-$ARCH"
BASE="https://github.com/$REPO/releases/download/$VERSION"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "install.sh: downloading $ASSET from $VERSION..."
fetch -o "$TMP/$ASSET" "$BASE/$ASSET"
fetch -o "$TMP/checksums.txt" "$BASE/checksums.txt"

(cd "$TMP" && grep -F "$ASSET" checksums.txt | "${SHA_CMD[@]}")

mkdir -p "$BINDIR"
install -m 0755 "$TMP/$ASSET" "$BINDIR/lazypost"

echo "installed: $BINDIR/lazypost"
echo "version:   $("$BINDIR/lazypost" -version)"
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) echo "add to your PATH: export PATH=\"$BINDIR:\$PATH\"" ;;
esac
