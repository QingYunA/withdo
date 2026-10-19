#!/usr/bin/env bash
# Builds the macOS release files into app/build: the universal app (which is adhd-core too), the
# signed update archive and manifest when updates are on, the DMG, and the standalone adhd-core tarball with its
# SHA256SUMS.txt. It uploads nothing.
#   scripts/build-mac.sh
# The update key is the secret half of mygo's key pair (mygo keygen). It stays out of the repository;
# set MYGO_UPDATE_KEY to use another file than the default.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# Updates are off while the repository is private (docs/plans/m10-auto-update.md): with no "updates"
# in mygo.json the app is built without them, and no key is needed.
KEY="${MYGO_UPDATE_KEY:-$HOME/Library/Application Support/mygo/update-keys/mygo-update.key}"
if grep -q '"updates"' app/mygo.json; then
  [ -r "$KEY" ] || { echo "no update key at $KEY (set MYGO_UPDATE_KEY, or run: cd app && go tool mygo keygen)" >&2; exit 1; }
  export MYGO_UPDATER_PRIVATE_KEY="$(cat "$KEY")"
fi

version="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' app/mygo.json)"
out="$PWD/app/build/darwin-universal"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# The app is adhd-core too (app/cli.go), so the bundle carries one program.
(cd app && go tool mygo build -platform darwin/universal)

# The standalone adhd-core, for machines without the app: built on its own for both architectures,
# unsigned, as a bare file.
for arch in arm64 amd64; do
  GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$work/adhd-core-$arch" ./cmd/adhd-core
done
mkdir -p "$work/tar"
lipo -create -output "$work/tar/adhd-core" "$work/adhd-core-arm64" "$work/adhd-core-amd64"
tar -czf "$out/adhd-core-$version-macos-universal.tar.gz" -C "$work/tar" adhd-core

# The DMG under the name the release gives it (GitHub turns the spaces of the built name into dots).
mv "$out/ADHD Todo $version.dmg" "$out/ADHD-Todo-$version-macos-universal.dmg"
(cd "$out" && shasum -a 256 "ADHD-Todo-$version-macos-universal.dmg" "adhd-core-$version-macos-universal.tar.gz" > SHA256SUMS.txt)
echo "built $version into $out"
