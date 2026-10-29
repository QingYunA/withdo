#!/usr/bin/env bash
# Uploads the files of scripts/build-release.sh (and the Android APK) to a DRAFT release v<version> of
# this repository, QingYunA/withdo, with the notes of the "## <version>" section of CHANGELOG.md.
# Installed apps (mygo's updater, the Android updater) read its latest release.
#   scripts/publish-release.sh 0.9.0              make or refill the draft
#   scripts/publish-release.sh 0.9.0 --publish    tag v0.9.0 on origin/main and make it the latest release
# Run build-release.sh and android-release.sh first. Nothing is built here.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

REPO="QingYunA/withdo"
version="${1:-}"
[ -n "$version" ] || { echo "usage: scripts/publish-release.sh <version> [--publish]" >&2; exit 2; }
publish=false
case "${2:-}" in
  "") ;;
  --publish) publish=true ;;
  *) echo "unknown argument ${2}" >&2; exit 2 ;;
esac

built="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' app/mygo.json)"
[ "$built" = "$version" ] || { echo "app/mygo.json is at $built, not $version" >&2; exit 1; }

release="$PWD/app/build/release"
apk="withdo-$version.apk"
# Android 0.9–0.11 update only from an asset named adhd-todo-<version>.apk: the same file goes up under
# that name too, until no such phone is left.
old_apk="adhd-todo-$version.apk"
for f in "Withdo-$version-macos-universal.dmg" "withdo-cli-$version-macos-universal.tar.gz" \
         "Withdo-Setup-$version-windows-x64.exe" "withdo-$version-darwin-universal.tar.gz" \
         "withdo-$version-windows-amd64.tar.gz" update-darwin-universal.json update-windows-amd64.json \
         SHA256SUMS.txt; do
  [ -e "$release/$f" ] || { echo "missing $release/$f: run scripts/build-release.sh" >&2; exit 1; }
done

# The APK joins after the desktop build (which empties the folder), with its line in SHA256SUMS.txt:
# the phone checks the download against that line.
if [ ! -e "$release/$apk" ]; then
  [ -e "dist/$apk" ] || { echo "missing dist/$apk: run scripts/android-release.sh" >&2; exit 1; }
  cp "dist/$apk" "$release/$apk"
fi
[ -e "$release/$old_apk" ] || cp "$release/$apk" "$release/$old_apk"
for f in "$apk" "$old_apk"; do
  if ! grep -q " $f\$" "$release/SHA256SUMS.txt"; then
    (cd "$release" && shasum -a 256 "$f" >> SHA256SUMS.txt)
  fi
done

# The notes: the changelog section, then the line about agents that keep running the old MCP server.
notes="$(mktemp)"
trap 'rm -f "$notes"' EXIT
awk -v v="$version" '
  $0 == "## " v { on = 1; next }
  /^## / { on = 0 }
  on { print }
' CHANGELOG.md | sed -e '/./,$!d' > "$notes"
[ -s "$notes" ] || { echo "CHANGELOG.md has no \"## $version\" section" >&2; exit 1; }
printf '\n**Restart your coding agents after updating** (Claude Code and the others): an agent keeps running the old MCP server until it is restarted, so it does not see the new tools before.\n' >> "$notes"

tag="v$version"
if gh release view "$tag" --repo "$REPO" >/dev/null 2>&1; then
  draft="$(gh release view "$tag" --repo "$REPO" --json isDraft --jq .isDraft)"
  [ "$draft" = true ] || { echo "$tag is already published on $REPO" >&2; exit 1; }
  echo "refilling the draft $tag"
  gh release upload "$tag" --repo "$REPO" --clobber "$release"/*
  gh release edit "$tag" --repo "$REPO" --title "Withdo $tag" --notes-file "$notes"
else
  gh release create "$tag" --repo "$REPO" --draft --title "Withdo $tag" --notes-file "$notes" "$release"/*
fi

if [ "$publish" = true ]; then
  # The tag goes on origin/main first, so publishing releases that commit.
  git fetch origin --quiet
  if ! git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then git tag "$tag" origin/main; fi
  git push origin "$tag"
  # Publishing makes it the latest release, which is what installed apps ask.
  gh release edit "$tag" --repo "$REPO" --draft=false --latest
  echo "published $tag on $REPO"
else
  echo "draft $tag is on $REPO; check it, then: scripts/publish-release.sh $version --publish"
fi
