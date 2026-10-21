#!/usr/bin/env bash
# Re-render the Android design screenshots into docs/design/screens/android.
#   scripts/android-screens.sh [test-class-regex]
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

if [ -z "${JAVA_HOME:-}" ] && command -v brew >/dev/null 2>&1; then
  JAVA_HOME="$(brew --prefix openjdk@17)"
  export JAVA_HOME
fi
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
export GRADLE_USER_HOME="${GRADLE_USER_HOME:-/tmp/adhd-gradle}"
export ANDROID_USER_HOME="${ANDROID_USER_HOME:-/tmp/adhd-android-home}"

if [ ! -f android/core/core.aar ]; then
  echo "android/core/core.aar is missing: run scripts/android-core.sh first" >&2
  exit 1
fi
if [ -z "${JAVA_HOME:-}" ] || [ ! -x "$JAVA_HOME/bin/java" ]; then
  echo "a JDK is required: set JAVA_HOME (e.g. $(brew --prefix openjdk@17 2>/dev/null || echo /path/to/jdk17))" >&2
  exit 1
fi

TEST="${1:-xyz.sriai.adhdtodo.AndroidScreensTest}"
(cd android && ./gradlew --offline --no-daemon -Pkotlin.compiler.execution.strategy=in-process \
  -Proborazzi.test.record=true testDebugUnitTest --tests "$TEST")

OUT="$PWD/docs/design/screens/android"
mkdir -p "$OUT"
cp android/app/build/screenshots/*.png "$OUT"/
echo "wrote $(ls "$OUT"/*.png | wc -l | tr -d ' ') screenshots to docs/design/screens/android"
