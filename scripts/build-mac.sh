#!/usr/bin/env bash
# Builds the macOS release files into app/build: the universal app with adhd-core inside it, the
# signed update archive and manifest, the DMG, and the standalone adhd-core tarball with its
# SHA256SUMS.txt. It uploads nothing.
#   scripts/build-mac.sh
# The update key is the secret half of mygo's key pair (mygo keygen). It stays out of the repository;
# set MYGO_UPDATE_KEY to use another file than the default.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

KEY="${MYGO_UPDATE_KEY:-$HOME/Library/Application Support/mygo/update-keys/mygo-update.key}"
[ -r "$KEY" ] || { echo "no update key at $KEY (set MYGO_UPDATE_KEY, or run: cd app && go tool mygo keygen)" >&2; exit 1; }

version="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' app/mygo.json)"
out="$PWD/app/build/darwin-universal"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# One adhd-core for both architectures. It is a resource of the app, which mygo build signs with it.
for arch in arm64 amd64; do
  GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$work/adhd-core-$arch" ./cmd/adhd-core
done
mkdir -p app/resources/darwin
lipo -create -output app/resources/darwin/adhd-core "$work/adhd-core-arm64" "$work/adhd-core-amd64"

(cd app && MYGO_UPDATER_PRIVATE_KEY="$(cat "$KEY")" go tool mygo build -platform darwin/universal)

# The standalone adhd-core, for machines without the app: the same binary, unsigned, as a bare file.
mkdir -p "$work/tar"
cp app/resources/darwin/adhd-core "$work/tar/adhd-core"
tar -czf "$out/adhd-core-$version-macos-universal.tar.gz" -C "$work/tar" adhd-core

# The DMG under the name the release gives it (GitHub turns the spaces of the built name into dots).
mv "$out/ADHD Todo $version.dmg" "$out/ADHD-Todo-$version-macos-universal.dmg"
(cd "$out" && shasum -a 256 "ADHD-Todo-$version-macos-universal.dmg" "adhd-core-$version-macos-universal.tar.gz" > SHA256SUMS.txt)
echo "built $version into $out"
