#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?usage: release.sh vX.Y.Z}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

rm -rf "$ROOT/dist"
mkdir -p "$ROOT/dist"

for target in "darwin arm64" "darwin amd64" "linux arm64" "linux amd64"; do
	set -- $target
	echo "building lazypost-$1-$2"
	GOOS="$1" GOARCH="$2" CGO_ENABLED=0 \
		go build -trimpath -ldflags "-X main.version=$VERSION" \
		-o "$ROOT/dist/lazypost-$1-$2" .
done

for asset in lazypost-darwin-arm64 lazypost-darwin-amd64 lazypost-linux-arm64 lazypost-linux-amd64; do
	[ -x "$ROOT/dist/$asset" ] || { echo "missing release asset: $asset" >&2; exit 1; }
done

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$ROOT/dist" && sha256sum lazypost-* > checksums.txt)
else
	(cd "$ROOT/dist" && shasum -a 256 lazypost-* > checksums.txt)
fi
[ "$(wc -l < "$ROOT/dist/checksums.txt" | tr -d ' ')" -eq 4 ] || {
	echo "expected four release checksums" >&2
	exit 1
}
echo "done: $(cd "$ROOT/dist" && ls)"
