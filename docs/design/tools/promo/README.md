# Promo video render

Spec: `docs/design/promo.md`. One command per language, from anywhere in the repo:

```
docs/design/tools/promo/render.sh zh
docs/design/tools/promo/render.sh en
```

Needs Go and `ffmpeg` (Homebrew's is found on the PATH or at `/opt/homebrew/bin/ffmpeg`). It writes, to the
git-ignored `out/promo/`:

- `adhd-todo-summary-<lang>-4k.mp4` (3840 x 2160) and `-1080p.mp4`, H.264, 60 fps, yuv420p, AAC, about 28 s;
- `stills-<lang>/*.png`, 1920 x 1080 frames for review;
- `work-<lang>/`, the frames (about 1 GB; delete it when done).

About 5 minutes a language on an M-series Mac (frames and cards 1 min, the 4K and 1080p encode the rest).

## How it is made

- `app/views/promo_*_test.go` (build tag `promo`, so `go test ./...` and CI skip them; they are in
  `app/views` because they drive the unexported `Main`):
  - `promo_data_test.go`: the made-up Fake store (48 tasks over Monday to Friday, 7 people, 4 projects, stored
    paragraphs in both languages). Nothing from a real database.
  - `promo_frames_test.go`: `TestPromoWindow` drives the real views frame by frame. The app's clock
    (`App.Now`) and the renderer's own (mygo's `Tester`, reached by reflection since v0.5.1 has no setter) are
    one virtual clock that the test advances by 1/60 s, so the scene, the paragraph's typing, the ledger's
    count and every transition follow it. The pointer is a real pointer: the test moves it and clicks it at
    3.55 s (Summary) and 16.4 s (This week), then draws its 12 x 12 sprite onto the frame. Writes
    `w00000.png...` (2200 x 1440, rounded corners in the alpha) and `track.json`.
  - `promo_audio_test.go`: lays `theme.DayPlan.SoundCues` / `WeekPlan.SoundCues` at the seconds the scenes
    began, each wav resampled by `Cue.Speed()`, into `audio.wav`.
  - `promo_cards_test.go`: `TestPromoCards` writes the dimmed backdrop with the window's shadow, the title
    card's frames (the six figures hop one after another) and the end card, drawn with the same renderer.
- `render.sh`: runs the two tests, then one ffmpeg run: window over the backdrop, crossfades, fades, both
  sizes and the sound.

Run a test on its own with `PROMO_LANG=zh PROMO_OUT=/tmp/promo go test -tags promo ./app/views -run TestPromoWindow -count=1`.
