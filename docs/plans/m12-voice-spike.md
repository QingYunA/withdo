# M12 voice input spike

Decision 4 of "Input first (v31)": voice is the system's speech to text, streaming, no text to speech.
mygo has no speech API (0.5.1). This spike finds how to reach the system recognizers from Go. Nothing
is wired into the UI yet.

## Result in one paragraph

macOS works without cgo: `internal/speech` drives `SFSpeechRecognizer` + `AVAudioEngine` through
`github.com/ebitengine/purego/objc` (the same mechanism mygo uses, and the reason the release build
is `CGO_ENABLED=0`). The permission prompts, the engine start and the recognition task run end to
end on a real Mac; what could not be seen here is text coming back, because Dictation is switched
off on the spike machine and the system answers `Siri and Dictation are disabled` (see Testing).
Windows and the other platforms get a stub. Android and iOS use their own APIs from the native app.

## macOS

- **API**: `SFSpeechAudioBufferRecognitionRequest` (partial results on, on-device when
  `supportsOnDeviceRecognition`), fed by an `installTapOnBus` tap of `AVAudioEngine.inputNode`.
  Results come in a block (`bestTranscription.formattedString`, `isFinal`).
- **Why not cgo**: `mygo build` forces `CGO_ENABLED=0` (cmd/mygo/proc.go) and builds the universal
  binary from two plain Go builds. An Objective-C file would turn on cgo, need the Xcode toolchain
  for both arches and break the cross-compile from the same script. purego needs none of it.
- **Why not mygo's own `internal/darwin`**: it is `internal`, so the app may not import it. It also
  has no speech code; it only shows the pattern (purego `objc.Send`).
- **Blocks**: `objc.NewBlock(func(objc.Block, args...))` makes the result handler, the tap and the
  authorization callbacks. They run on system threads (purego supports that with cgo off). The
  recognizer's `queue` is set to a fresh `NSOperationQueue`; the default is the main queue, which a
  command line tool never drains and the UI should not carry.
- **Permissions**: two TCC prompts, speech recognition and microphone, shown once per app.
  `Info.plist` must carry `NSMicrophoneUsageDescription` and `NSSpeechRecognitionUsageDescription`,
  or macOS aborts the process when the prompt is due (seen: `signal: abort trap` for a bare binary).
  mygo supports this: `macos.infoPlist` in `app/mygo.json` adds keys to the bundle's Info.plist
  (docs/configuration.md). Add, when wiring in:

  ```json
  "macos": { "infoPlist": {
    "NSMicrophoneUsageDescription": "Say a task instead of typing it.",
    "NSSpeechRecognitionUsageDescription": "Turn what you say into text for the input."
  } }
  ```

  These strings need translations: mygo writes `InfoPlist.strings` per language from `i18n`; check
  it picks them up (it does for the display name) or the prompts stay English.
- **Entitlements**: none while the app is not signed with the hardened runtime (today
  `macos.signingIdentity` is unset). Once it is signed and notarized (m10), add an `entitlements`
  plist with `com.apple.security.device.audio-input` or the microphone yields silence.
- **Build impact**: `go.mod` makes `github.com/ebitengine/purego` a direct dependency (it was
  already in the build through mygo). Darwin-only file; `GOOS=windows GOARCH=amd64 go build ./...`
  and the Linux build are unaffected.
- **Recognition modes**: on-device is requested when supported (privacy, no quota, offline), so it
  needs the language's dictation model, which needs Dictation on. Without it the server path also
  fails (`Siri and Dictation are disabled`). So the app must treat `LastError` as normal and say
  "turn on Dictation in System Settings > Keyboard" once.
- **Language**: `SFSpeechRecognizer new` uses the system locale. Chinese and English mixed in one
  sentence is weak in either; a setting for the recognition language is worth having (a locale per
  `initWithLocale:`), not done in the spike.
- **iOS later**: the same API, but through gomobile/Swift, not purego.

## Windows

- **API**: `Windows.Media.SpeechRecognition.SpeechRecognizer` with a continuous session
  (`ContinuousRecognitionSession`, `HypothesisGenerated` for partials, `ResultGenerated` for settled
  phrases) and `SpeechRecognitionTopicConstraint(Dictation)`. Needs "Online speech recognition" on
  (Settings > Privacy > Speech), a microphone, and `microphone` capability consent; for an
  unpackaged desktop app the OS asks the first time the microphone opens.
- **From Go**: there is no stock binding. Options, cheapest first:
  1. `github.com/deploymenttheory/go-bindings-winrt` has a `speechrecognition` package (generated
     WinRT vtables); young and unreviewed, but pure Go. Try it first in the Windows follow-up.
  2. `github.com/saltosystems/winrt-go` has the generator and the `media` package but no speech
     package; generate one.
  3. A hand-written vtable layer over `golang.org/x/sys/windows` (`RoGetActivationFactory`, event
     handlers as `syscall.NewCallback`). About 400 lines; the event delegates are the painful part.
  4. A helper process (PowerShell 5.1 loads WinRT types) streaming text on stdout. Works without
     new dependencies but PowerShell startup is about 1 s and awaiting `IAsyncOperation` needs glue.
  5. SAPI (`SAPI.SpInProcRecognizer` through go-ole): old, needs a trained desktop recognizer, poor
     quality. Not recommended.
- **Not prototyped**: I cannot run Windows here, and an untested vtable layer would be worth little.
  The stub (`speech_other.go`) compiles for Windows; `GOOS=windows GOARCH=amd64 go build ./...`
  passes. Recommended: option 1, in its own PR with a Windows tester, behind the same `Recognizer`.
- **Build impact**: pure Go with `CGO_ENABLED=0` like the rest; the Windows release script stays as
  is.

## Android (no code in this spike)

`android.speech.SpeechRecognizer` in the Kotlin app: `createSpeechRecognizer(context)` (or
`createOnDeviceSpeechRecognizer` on API 33+), `RecognizerIntent.ACTION_RECOGNIZE_SPEECH` with
`EXTRA_PARTIAL_RESULTS = true`, `RecognitionListener.onPartialResults` / `onResults`. Needs
`RECORD_AUDIO` (runtime permission) and, from Android 11, a `<queries>` entry for
`android.speech.RecognitionService`. It must be created and used on the main thread. The Kotlin side
owns it and hands text to the Go core (mobile/) like any other input; the Go `internal/speech`
package is not used there. iOS is the same story with `SFSpeechRecognizer` in Swift.

## Interface

`internal/speech` (`speech.go`): `Recognizer` with `Available()`, `Authorization()`,
`RequestAuthorization(done)`, `Start(onPartial, onFinal)`, `Stop()`, `Cancel()`, `LastError()`.
`onPartial` carries the whole text so far (the UI replaces the input's voice part with it; words
the recognizer may still change are not distinguished: SFSpeech gives per-segment confidence, and
`text.tertiary` for unsettled words would use `segments[].confidence` or the last N segments, a
detail for the UI PR). Callbacks arrive on system threads: hop to the UI thread (mygo's main-thread
helper) before touching state.

## Testing it yourself on a Mac

```
internal/speech/cmd/listen/run.sh -for 8s
```

It builds `Listen.app` in `$TMPDIR` with the two usage descriptions, ad-hoc signs it, runs it through
LaunchServices (so the prompts say "Listen") and relays its output. Allow the speech and the
microphone prompts, speak during the countdown, expect `partial:` lines and one `final:`.
`go run ./internal/speech/cmd/listen` also works from a terminal that has microphone and speech
rights, but a bare binary is aborted by macOS when it is not launched from such a terminal.

If it prints `error: Siri and Dictation are disabled`, turn on System Settings > Keyboard >
Dictation and run again. That is what the spike machine said, so the last step (words coming back)
is the one thing not proven here.

## Risks

- Dictation off, or the model not downloaded: no recognition at all. UI needs a clear one-line
  message and a settings deep link (`x-apple.systempreferences:com.apple.Keyboard-Settings.extension`).
- A recognition request lasts about a minute on the server path (on-device has no such limit): the
  UI should send or restart at 55 s.
- purego blocks are less exercised than cgo: a bad signature crashes the process instead of
  returning an error. Mitigation: keep the speech code in one file, covered by the listen command,
  and run it in a CI-less manual check before each release.
- Releasing blocks: the code releases its blocks in `teardown`; the runtime keeps a block alive
  while the recognizer holds a copy. Leak-check with Instruments once wired in.
- Hold-to-talk with `⌥ Space` needs the global hotkey (mygo `internal/darwin/hotkey.go`, the
  public hotkey API of mygo) to report key up as well as key down; verify before promising it.
- Windows has the biggest unknown (binding quality); keep it behind the same interface so the UI
  does not wait for it.

## Steps to wire it into the composer

1. `app/mygo.json`: add `macos.infoPlist` (above) and the translations of the two strings.
2. `app/`: a `voice.go` service owning one `speech.Recognizer`, with `Begin()`, `End(send bool)`,
   state `idle | listening | denied | unavailable`, publishing partial text through the same channel
   the other state changes use; callbacks hop to the UI thread.
3. First use: if `Authorization()` is `AuthNotDetermined`, call `RequestAuthorization` when the
   person first presses the microphone, not at launch; if `AuthDenied`, show one line with a link to
   System Settings > Privacy & Security.
4. Composer: the pixel microphone button toggles listening (click) and a press and hold of `⌥ Space`
   listens while held. Partial text replaces the voice part of the input (keep what was typed
   before); the microphone becomes the filled accent square with the pixel waveform.
5. Release (or second click) calls `Stop`; when `onFinal` arrives, send the sentence through the
   normal input path (applied at once, with Undo). Esc calls `Cancel` and restores the typed text.
6. Menu bar flyout (decision 6) reuses the same service.
7. Windows: build the recognizer behind the same interface, then switch the stub off per GOOS.
8. Tests: a fake `Recognizer` for the composer logic (partial, final, error, cancel); the real one
   stays a manual check (`run.sh`).
