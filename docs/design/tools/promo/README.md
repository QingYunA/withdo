# Promo video render

Spec: `docs/design/promo.md`. One command per language, from anywhere in the repo:

```
docs/design/tools/promo/render.sh zh
docs/design/tools/promo/render.sh en
```

Needs Go and `ffmpeg` (Homebrew's is found on the PATH or at `/opt/homebrew/bin/ffmpeg`). It writes, to the
git-ignored `out/promo/`:

- `adhd-todo-summary-<lang>-4k.mp4` (3840 x 2160) and `-1080p.mp4`, H.264, 60 fps, yuv420p, AAC, about 32 s;
- `stills-<lang>/*.png`, 1920 x 1080 frames for review.

About 2 minutes a language on an M-series Mac (about 45 s of frames, the rest the 4K and 1080p encode).
The frames go to `out/promo/work-<lang>/` (about 1 GB of 4K PNGs), which the script deletes when it is done.

## How it is made

- `app/views/promo_*_test.go` (build tag `promo`, so `go test ./...` and CI skip them; they are in
  `app/views` because they drive the unexported `Main`):
  - `promo_data_test.go`: the made-up Fake store (48 tasks over Monday to Friday, 7 people, 4 projects, stored
    paragraphs in both languages). Nothing from a real database.
  - `promo_frames_test.go`: `TestPromoWindow` drives the real views frame by frame. The app's clock
    (`App.Now`) and the renderer's own (mygo's `Tester`, reached by reflection since v0.5.1 has no setter) are
    one virtual clock that the test advances by 1/60 s, so the scene, the paragraph's typing, the ledger's
    count and every transition follow it. The pointer is a real pointer: the test moves it and clicks it
    (Summary, then This week), then draws its 12 x 12 sprite. The camera eases in on the scene band while a
    scene plays; the window is then rendered again at the larger scale (4 px a point), never enlarged.
    Writes finished 3840 x 2160 frames `w00000.png...`, `track.json` (the clicks, the camera's path, the
    sound cues), `timing.env` and `stills.txt` for the script, and `audio.wav`.
  - `promo_audio_test.go`: lays `theme.DayPlan.SoundCues` / `WeekPlan.SoundCues` at the seconds the scenes
    began, each wav resampled by `Cue.Speed()`.
  - `promo_cards_test.go`: `TestPromoCards` writes the title card's frames (the six agents' figures hop one
    after another) and the end card, drawn with the same renderer.
- `render.sh`: runs the two tests, then one ffmpeg run: crossfades, fades, both sizes and the sound.

The length of the video follows the app's clocks: the window segment ends when the week's ledger has
counted up (`timing.env`).

Run a test on its own with `PROMO_LANG=zh PROMO_OUT=/tmp/promo go test -tags promo ./app/views -run TestPromoWindow -count=1`.
