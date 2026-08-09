#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?usage: release.sh vX.Y.Z}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

rm -rf "$ROOT/dist"
mkdir -p "$ROOT/dist"

for target in "darwin arm64" "darwin amd64" "linux amd64"; do
	set -- $target
	echo "building lazypost-$1-$2"
	GOOS="$1" GOARCH="$2" CGO_ENABLED=0 \
		go build -trimpath -ldflags "-X main.version=$VERSION" \
		-o "$ROOT/dist/lazypost-$1-$2" .
done

(cd "$ROOT/dist" && shasum -a 256 lazypost-* > checksums.txt)
echo "done: $(cd "$ROOT/dist" && ls)"