#!/usr/bin/env bash
# Builds the release files of both platforms each release ships, macOS and Windows x64, and gathers
# the ones to upload into app/build/release with their SHA256SUMS.txt:
#   macOS:   the DMG of the universal app (which is adhd-core too), the standalone adhd-core tarball,
#            and the signed update archive and manifest when updates are on;
#   Windows: the NSIS installer of the app, which carries the console adhd-core.exe next to it.
# It uploads nothing.
#   scripts/build-release.sh
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
mac="$PWD/app/build/darwin-universal"
win="$PWD/app/build/windows-amd64"
release="$PWD/app/build/release"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# The app is adhd-core too (app/cli.go), so the macOS bundle carries one program. On Windows the
# app is a GUI program that cannot be an MCP stdio server, so the console adhd-core.exe the agents
# run is built into app/resources/windows-amd64/, which mygo installs next to the app. makensis,
# which makes the Windows installer, aborts with std::bad_alloc unless the locale is UTF-8.
mkdir -p app/resources/windows-amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o app/resources/windows-amd64/adhd-core.exe ./cmd/adhd-core
(cd app && LC_ALL=en_US.UTF-8 go tool mygo build -platform darwin/universal,windows/amd64)

# The standalone adhd-core for macOS, for machines without the app: built on its own, unsigned. One
# universal file for both architectures.
for arch in arm64 amd64; do
  GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$work/adhd-core-$arch" ./cmd/adhd-core
done
mkdir -p "$work/mac"
lipo -create -output "$work/mac/adhd-core" "$work/adhd-core-arm64" "$work/adhd-core-amd64"

# Everything to upload, under the names the releases give it (GitHub turns spaces into dots).
rm -rf "$release"
mkdir -p "$release"
cp "$mac/ADHD Todo $version.dmg" "$release/ADHD-Todo-$version-macos-universal.dmg"
tar -czf "$release/adhd-core-$version-macos-universal.tar.gz" -C "$work/mac" adhd-core
cp "$win/ADHD Todo Setup $version amd64.exe" "$release/ADHD-Todo-Setup-$version-windows-x64.exe"
for f in "$mac"/adhd-todo-* "$mac"/update-*.json "$win"/adhd-todo-* "$win"/update-*.json; do
  if [ -e "$f" ]; then cp "$f" "$release/"; fi
done
(cd "$release" && shasum -a 256 * > SHA256SUMS.txt)
echo "built $version into $release"
