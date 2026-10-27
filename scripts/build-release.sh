#!/usr/bin/env bash
# Builds the release files of both platforms each release ships, macOS and Windows x64, and gathers
# the ones to upload into app/build/release with their SHA256SUMS.txt:
#   macOS:   the DMG of the universal app (which is the withdo command too), the standalone withdo
#            tarball, and the signed update archive, deltas and manifest
#            (withdo-<v>-darwin-universal.tar.gz, update-darwin-universal.json);
#   Windows: the NSIS installer of the app, which carries the console withdo.exe next to it (and
#            adhd-core.exe, the name old agent configs run), and the signed update archive and
#            manifest (withdo-<v>-windows-amd64.tar.gz, update-windows-amd64.json).
# It uploads nothing; scripts/publish-release.sh does.
#   scripts/build-release.sh
# The update key is the secret half of mygo's key pair (mygo keygen). It stays out of the repository;
# set MYGO_UPDATE_KEY to use another file than the default.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# Updates are on (docs/plans/m10-auto-update.md): mygo signs every platform's update archive with the
# private key, so the build needs it and stops without it rather than make an app that cannot update.
KEY="${MYGO_UPDATE_KEY:-$HOME/Library/Application Support/mygo/update-keys/mygo-update.key}"
[ -r "$KEY" ] || { echo "no update key at $KEY (set MYGO_UPDATE_KEY; the key pair is made once by: cd app && go tool mygo keygen)" >&2; exit 1; }
export MYGO_UPDATER_PRIVATE_KEY="$(cat "$KEY")"

version="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' app/mygo.json)"
mac="$PWD/app/build/darwin-universal"
win="$PWD/app/build/windows-amd64"
release="$PWD/app/build/release"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# The app is withdo too (app/cli.go), so the macOS bundle carries one program. On Windows the
# app is a GUI program that cannot be an MCP stdio server, so the console withdo.exe the agents
# run is built into app/resources/windows-amd64/, which mygo installs next to the app. A copy is
# named adhd-core.exe, the program the agent configs written before the rename run. makensis,
# which makes the Windows installer, aborts with std::bad_alloc unless the locale is UTF-8.
mkdir -p app/resources/windows-amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o app/resources/windows-amd64/withdo.exe ./cmd/withdo
cp app/resources/windows-amd64/withdo.exe app/resources/windows-amd64/adhd-core.exe
# "ADHD Todo.exe" is the launcher that stands in for the app's old executable name: the old version's
# updater swaps the new files into its folder and starts "ADHD Todo.exe" again, as do the shortcuts it
# made. The launcher starts Withdo.exe next to it (cmd/legacy-launcher).
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o "app/resources/windows-amd64/ADHD Todo.exe" ./cmd/legacy-launcher
(cd app && LC_ALL=en_US.UTF-8 go tool mygo build -platform darwin/universal,windows/amd64)

# The standalone withdo for macOS, for machines without the app: built on its own, unsigned. One
# universal file for both architectures.
for arch in arm64 amd64; do
  GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$work/withdo-$arch" ./cmd/withdo
done
mkdir -p "$work/mac"
lipo -create -output "$work/mac/withdo" "$work/withdo-arm64" "$work/withdo-amd64"

# Everything to upload, under the names the releases give it (GitHub turns spaces into dots).
rm -rf "$release"
mkdir -p "$release"
cp "$mac/Withdo $version.dmg" "$release/Withdo-$version-macos-universal.dmg"
tar -czf "$release/withdo-cli-$version-macos-universal.tar.gz" -C "$work/mac" withdo
cp "$win/Withdo Setup $version amd64.exe" "$release/Withdo-Setup-$version-windows-x64.exe"
for f in "$mac"/withdo-* "$mac"/update-*.json "$win"/withdo-* "$win"/update-*.json; do
  if [ -e "$f" ]; then cp "$f" "$release/"; fi
done
# Without these an installed app finds no update, so a build that lacks one is no release.
for f in "withdo-$version-darwin-universal.tar.gz" update-darwin-universal.json \
         "withdo-$version-windows-amd64.tar.gz" update-windows-amd64.json; do
  [ -e "$release/$f" ] || { echo "the build made no $f" >&2; exit 1; }
done
(cd "$release" && shasum -a 256 * > SHA256SUMS.txt)
echo "built $version into $release"
