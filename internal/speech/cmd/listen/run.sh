#!/usr/bin/env bash
# Builds cmd/listen into a throwaway Listen.app (a bare binary has no Info.plist, so macOS ends it
# the moment it asks for the microphone) and runs it through LaunchServices, so the permission
# prompts name "Listen". Output is relayed from a log file.
#   internal/speech/cmd/listen/run.sh [-for 8s]
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
app="${TMPDIR:-/tmp}/Listen.app"
log="${TMPDIR:-/tmp}/listen.log"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
CGO_ENABLED=0 go build -o "$app/Contents/MacOS/listen" ./internal/speech/cmd/listen
cat >"$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleIdentifier</key><string>dev.adhdtodo.listen</string>
  <key>CFBundleName</key><string>Listen</string>
  <key>CFBundleExecutable</key><string>listen</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSUIElement</key><true/>
  <key>NSMicrophoneUsageDescription</key><string>Spike: listen for a task.</string>
  <key>NSSpeechRecognitionUsageDescription</key><string>Spike: turn speech into a task.</string>
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
