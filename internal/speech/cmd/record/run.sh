#!/usr/bin/env bash
# Builds cmd/record into a throwaway Record.app (a bare binary has no Info.plist, so macOS ends it
# the moment it asks for the microphone) and runs it through LaunchServices, so the permission
# prompt names "Record". Output is relayed from a log file.
#   internal/speech/cmd/record/run.sh [-for 5s] [-o /tmp/record.wav]
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
app="${TMPDIR:-/tmp}/Record.app"
log="${TMPDIR:-/tmp}/record.log"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
CGO_ENABLED=0 go build -o "$app/Contents/MacOS/record" ./internal/speech/cmd/record
cat >"$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleIdentifier</key><string>dev.adhdtodo.record</string>
  <key>CFBundleName</key><string>Record</string>
  <key>CFBundleExecutable</key><string>record</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSUIElement</key><true/>
  <key>NSMicrophoneUsageDescription</key><string>Check that voice input hears you.</string>
</dict></plist>
PLIST
codesign --force --sign - "$app"
: >"$log"
open -W --stdout "$log" --stderr "$log" "$app" --args "$@" &
tail -f "$log" &
tailpid=$!
wait %1 || true
sleep 0.5
kill "$tailpid" 2>/dev/null || true
