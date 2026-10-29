#!/usr/bin/env bash
# Builds a signed release APK of the phone app into dist/withdo-<version>.apk.
# publish-release.sh also uploads it as adhd-todo-<version>.apk: Android 0.9–0.11 look for exactly that name.
#   scripts/android-release.sh --init   once: make the release keystore and keep its password in
#                                       the keychain (back the keystore up: updates need it)
#   scripts/android-release.sh          build the core, the APK, and check its signature
# The keystore lives outside the repo; the password never touches a file.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

store="${ADHD_RELEASE_STORE_FILE:-$HOME/.config/adhd-todo/android-release.jks}"
alias_name="adhd-todo"
service="adhd-todo-android-release"
# Gradle finds the SDK through ANDROID_HOME or android/local.properties; a fresh worktree has no
# local.properties, so export the path the script uses itself.
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
sdk="$ANDROID_HOME"
export JAVA_HOME="${JAVA_HOME:-$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home}"

if [ "${1:-}" = "--init" ]; then
  if [ -f "$store" ]; then
    echo "the keystore already exists: $store" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$store")"
  password="$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9' | head -c 32)"
  security add-generic-password -U -s "$service" -a "$alias_name" -w "$password"
  "$JAVA_HOME/bin/keytool" -genkeypair -keystore "$store" -storetype PKCS12 -alias "$alias_name" \
    -keyalg RSA -keysize 4096 -validity 10000 -dname "CN=ADHD Todo, O=sri-ai.xyz" \
    -storepass "$password" -keypass "$password" >/dev/null 2>&1
  chmod 600 "$store"
  echo "made $store (password in the keychain as $service). Back it up: updates need this key."
  exit 0
fi

if [ ! -f "$store" ]; then
  echo "no release keystore at $store; run scripts/android-release.sh --init first" >&2
  exit 1
fi
password="$(security find-generic-password -s "$service" -a "$alias_name" -w)"

scripts/android-core.sh
version="$(sed -n 's/.*versionName = "\(.*\)".*/\1/p' android/app/build.gradle.kts)"
(
  cd android
  ADHD_RELEASE_STORE_FILE="$store" ADHD_RELEASE_STORE_PASSWORD="$password" \
    ADHD_RELEASE_KEY_ALIAS="$alias_name" ADHD_RELEASE_KEY_PASSWORD="$password" \
    ./gradlew --no-daemon -q assembleRelease
)
apk="android/app/build/outputs/apk/release/app-release.apk"
apksigner="$(ls -d "$sdk"/build-tools/* | sort -V | tail -1)/apksigner"
"$apksigner" verify --min-sdk-version 26 "$apk"
mkdir -p dist
out="dist/withdo-$version.apk"
cp "$apk" "$out"
echo "built $out ($(du -h "$out" | cut -f1)), signed by:"
"$apksigner" verify --print-certs "$apk" | grep -E "certificate DN|SHA-256"
