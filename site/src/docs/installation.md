# $ install lazypost

One command (checksum-verified; installs to `~/.local/bin`, override with `PREFIX`):

```bash
$ curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh
# resolves the latest release, verifies checksums, installs to ~/.local/bin
$ curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh -s -- v0.4.0
# pin a version; install location override: PREFIX=/usr/local sh install.sh
$ go build -o lazypost .
# or build from source — Go 1.25+
```

Pre-built binaries for macOS and Linux (arm64 + amd64) are attached to every GitHub release.
Re-running the install command updates to the latest release. Check what you are running with
`lazypost -version`.
