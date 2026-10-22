# M12 voice input

Decision 4 of "Input first (v31)" in `docs/design/ui-spec.md`. The first spike used the system
recognizer (`SFSpeechRecognizer`); it refuses while the person's Dictation is off, so voice now goes
through our own service. No text to speech.

## Architecture

```
app (internal/speech recorder)  --WAV-->  adhd-sync POST /v1/transcribe  --base64-->  MiMo mimo-v2.5-asr
        16 kHz mono PCM16, <= 60 s          bearer = the sync session            key lives only on the server
                                        <--{"text"}--                         <--text in choices[0]--
```

- **Recorder** (`internal/speech`): `Recorder` with `Available`, `Authorization`,
  `RequestAuthorization`, `Start(onLevel)`, `Stop() (wav, error)`, `Cancel`. macOS: an
  `AVAudioEngine` input-node tap through `github.com/ebitengine/purego/objc` (no cgo, because
  `mygo build` forces `CGO_ENABLED=0` and the release cross-builds Windows). The tap takes the
  hardware's format; `pcm.go` mixes the channels to mono, resamples to 16 kHz by streaming linear
  interpolation (independent of block size), tracks the loudness of each block and stops adding
  samples at 60 s. `EncodeWAV` writes the 44 byte header. Only the microphone permission
  (`AVCaptureDevice`) is needed. Other platforms: `ErrUnavailable`, so the UI hides the button.
  The object and block lifetimes follow one rule: `Start` releases the engine and the tap on every
  failure path; `Stop`/`Cancel` take the running state out under the mutex, then end the audio.
- **Usage description**: `app/mygo.json` has `macos.infoPlist.NSMicrophoneUsageDescription`; without
  it macOS ends the process at the prompt. mygo's `InfoPlist.strings` only localizes the app's name,
  so the description is English. Once the app is signed with the hardened runtime (m10), the
  entitlements file needs `com.apple.security.device.audio-input`, or the microphone yields silence.
- **Server** (`internal/syncserver/transcribe.go`): `POST /v1/transcribe`, `Content-Type: audio/wav`,
  bearer-authenticated like the other `/v1` routes (it needs only the token's subject, not a
  workspace). It checks the RIFF header (PCM16, at most 61 s, at most 7 MiB), counts the request
  against the account's UTC day (default 200, in memory like the sign-in limiter, refunded when the
  recognizer fails), calls MiMo with the key and returns `{"text"}`. Errors: 401 `unauthenticated`,
  503 `voice_disabled` (no key), 415, 400 `bad_audio`, 413, 429 `voice_quota` with `Retry-After`,
  502 `transcribe_failed`. Neither the key nor MiMo's body is returned or logged. Configuration is
  `MIMO_API_KEY`, `MIMO_BASE_URL`, `VOICE_DAILY_LIMIT` (`deploy/sync/README.md`, "Voice input").
- **Client** (`internal/syncclient/transcribe.go`): `Client.Transcribe(ctx, wav)` with the session's
  token source and no retry; sentinels `ErrVoiceNotSignedIn`, `ErrVoiceOff`, `ErrVoiceQuota`,
  `ErrVoiceNetwork`, `ErrVoiceNoSpeech`, `ErrVoiceFailed`. `data.Real.Transcribe` builds the client
  from the saved sync config when none is live.
- **UI** (`app/views/voice.go`, ui-spec "Voice input (m12)"): `App.Recorder` and `App.Transcriber`
  are injected like `Assistant`; tests use fakes. The text goes into the page composer and through
  `submit`, the normal send path.

## Not done

- **Hold `⌥ Space`**: mygo 0.5.1's `GlobalShortcut.Register(acc, fn)` has no key-up.
- **Menu bar popover**: does not exist on macOS yet (decision 6); it reuses `voiceButton` and
  `voiceLine` when built.
- **Windows recording**: needs a WASAPI or `waveIn` recorder behind the same `Recorder`; the button
  appears by itself once `Available` is true.
- **Android / iOS** record with their own APIs and call the same endpoint.
- **Language** is `auto` (Chinese and English); no setting.

## Testing the recorder on a Mac

```
internal/speech/cmd/record/run.sh -for 5s -o /tmp/record.wav
```

Builds a throwaway `Record.app` with the usage description, ad-hoc signs it and runs it through
LaunchServices. Allow the microphone prompt, speak, expect `recorded ... peak level 0.xx`; play
`/tmp/record.wav` to hear it. A bare `go run` is ended by macOS unless the terminal itself has
microphone rights.

## Risks

- purego blocks are less exercised than cgo: a wrong signature crashes the process. The tap block's
  arguments (`AVAudioPCMBuffer`, `AVAudioTime`) and the `floatChannelData` read are the part only a
  real microphone proves; run `cmd/record` before each release.
- The recognizer's key is one shared secret: a leaked server env means a quota burn. The per-account
  daily limit bounds one account, not the key; watch the MiMo dashboard.
- The quota is in memory: a restart resets the day's counts.
