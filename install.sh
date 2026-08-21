#!/bin/sh
set -eu

REPO="Dread-Code/lazypost"

usage() {
  cat <<'EOF'
Usage: install.sh [VERSION]

Installs the lazypost binary from GitHub Releases into ~/.local/bin
(or $PREFIX/bin when PREFIX is set, e.g. PREFIX=/usr/local).

VERSION defaults to the latest release.
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

VERSION=${1:-}
PREFIX=${PREFIX:-"$HOME/.local"}
BINDIR="$PREFIX/bin"

case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux ;;
  *)      echo "install.sh: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *)             echo "install.sh: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

fetch_text() {
  url=$1
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$url"
  else
    echo "install.sh: need curl or wget" >&2
    exit 1
  fi
}

download_file() {
  destination=$1
  url=$2
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$destination" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$destination" "$url"
  else
    echo "install.sh: need curl or wget" >&2
    exit 1
  fi
}

if [ -z "$VERSION" ]; then
  echo "install.sh: resolving latest release..."
  VERSION=$(fetch_text "https://api.github.com/repos/$REPO/releases/latest" |
    grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -n 1)
  [ -n "$VERSION" ] || { echo "install.sh: could not resolve the latest release" >&2; exit 1; }
fi

ASSET="lazypost-$OS-$ARCH"
BASE="https://github.com/$REPO/releases/download/$VERSION"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' 0 HUP INT TERM

echo "install.sh: downloading $ASSET from $VERSION..."
download_file "$TMP/$ASSET" "$BASE/$ASSET"
download_file "$TMP/checksums.txt" "$BASE/checksums.txt"

checksum=$(grep -F "$ASSET" "$TMP/checksums.txt" || true)
[ -n "$checksum" ] || { echo "install.sh: checksum entry missing for $ASSET" >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s\n' "$checksum" | (cd "$TMP" && sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  printf '%s\n' "$checksum" | (cd "$TMP" && shasum -a 256 -c -)
else
  echo "install.sh: need sha256sum or shasum" >&2
  exit 1
fi

mkdir -p "$BINDIR"
install -m 0755 "$TMP/$ASSET" "$BINDIR/lazypost"

echo "installed: $BINDIR/lazypost"
echo "version:   $($BINDIR/lazypost -version)"
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) echo "add to your PATH: export PATH=\"$BINDIR:\$PATH\"" ;;
esac
