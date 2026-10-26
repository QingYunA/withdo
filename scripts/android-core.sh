#!/usr/bin/env bash
# Builds the Go core for Android: android/core/core.aar, for phones (arm64) and the emulator
# (amd64/x86_64). The .aar is not committed; the Kotlin app is.
# Needs gomobile (already initialized), the Android SDK and NDK, and a JDK 17.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
# Without -trimpath every worktree path gets its own build cache entries and each new worktree recompiles everything.
export GOFLAGS="${GOFLAGS:--trimpath}"

export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
export ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-$HOME/Library/Android/sdk/ndk/27.2.12479018}"
if [ -z "${JAVA_HOME:-}" ]; then
  if command -v brew >/dev/null 2>&1; then
    JAVA_HOME="$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home"
    export JAVA_HOME
  fi
fi
export PATH="$(go env GOPATH)/bin:$PATH"

if ! command -v gomobile >/dev/null 2>&1; then
  echo "gomobile is not installed: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init" >&2
  exit 1
fi
for dir in "$ANDROID_HOME" "$ANDROID_NDK_HOME"; do
  if [ ! -d "$dir" ]; then
    echo "not found: $dir (set ANDROID_HOME and ANDROID_NDK_HOME)" >&2
    exit 1
  fi
done
if [ -z "${JAVA_HOME:-}" ] || [ ! -x "$JAVA_HOME/bin/javac" ]; then
  echo "a JDK is required: set JAVA_HOME (e.g. $(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home)" >&2
  exit 1
fi

mkdir -p android/core
rm -f android/core/core.aar
gomobile bind -ldflags "-s -w" -target=android/arm64,android/amd64 -androidapi 26 \
  -javapkg xyz.sriai.adhdtodo -o android/core/core.aar ./mobile/core

members="$(unzip -Z1 android/core/core.aar)"
for member in classes.jar jni/arm64-v8a/libgojni.so jni/x86_64/libgojni.so; do
  if ! grep -Fxq "$member" <<<"$members"; then
    echo "android/core/core.aar is missing $member" >&2
    exit 1
  fi
done
echo "built android/core/core.aar (classes.jar, arm64-v8a and x86_64 libgojni.so)"
