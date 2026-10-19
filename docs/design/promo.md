# Promo video: the day's shipping

Decided 2026-10-18 (discussion of v21–v23). A short video of Summary for the README, the release page
and social posts: the day's shipping with a crowd of agents, then the week. Rendered offline frame by
frame from the app's own views, so it is smooth, reproducible and uses only made-up data.

## Format

- 16:9. Two sizes from one render: 3840 x 2160 and 1920 x 1080, H.264 MP4, 60 fps, about 32 s.
- Two languages: Chinese (`zh-Hans`, the window in Chinese) and English. Same frames otherwise.
- Sound: only the Summary scene's own sounds (`app/theme/sounds/*.wav`, at the times
  `theme.DayPlan.SoundCues` / `WeekPlan.SoundCues` give). No music, no voice.
- The video files are not committed. They go to `out/promo/` (ignored) and are attached to a
  release or the README by hand.

## The picture

- The window, 1176 x 720 points rendered by the real renderer at 8/3 px a point (3136 x 1920, 82 % of the
  frame's width and 89 % of its height; a 3 pt cell of the scene is exactly 8 px), with its rounded
  corners and a soft shadow, centred on a backdrop that fills the frame: the dusk scene
  (`scenes/dusk.png`) scaled up with nearest neighbour and dimmed, so the window is the subject.
- A camera. While the scene plays (today's throws; the week's fall) it eases in 400 ms (smoothstep)
  to 1.5x on the scene band, which then fills the frame's width, and eases out again when the scene
  is over. The pointer's clicks, the paragraph and the ledger are seen with the whole window. Zoomed
  in, the window is rendered again at 4 px a point (a cell is 12 px), not enlarged, so text and
  pixels stay sharp. The path is in `track.json`.
- A pixel mouse pointer (a 12 x 12 arrow drawn at 3 pt a cell, white with a dark outline) moves
  between clicks on an eased path and dips a cell when it clicks. It is the user's pointer, so a
  pointer is right here (unlike a figure for Cursor).
- Title and end cards: the same backdrop, not dimmed, with native text (SF Pro / PingFang) and pixel
  art from `app/theme/sprites/`.

## Storyboard

| Time | What |
| --- | --- |
| 0–2.5 s | Title card: zh `你和 Agent，今天做了什么`, en `What you and your agents got done today`; the six agents' figures stand in a row under it and hop once, one after another. |
| 2.5–3.2 s | Cut to the window on Mine. The pointer moves to `Summary` in the sidebar and clicks (3.2 s). |
| 3.45–11.8 s | Today, the scene plays (dusk sky): 7 people, 16 goods, the lid, lands, stars, everyone hops. The camera is in from 3.85 s to 11.4 s. |
| 11.4–15.3 s | The camera is out. The assistant's paragraph types in, the ledger rows light up and count. |
| 14.6–15.5 s | The pointer moves to `This week` and clicks (15.5 s). |
| 15.75–21.3 s | The week (night): 48 goods fall into 7 stacks, Friday the highest, the busiest one hops. The camera is in from 16.15 s to 20.9 s. |
| 21.3–27.4 s | The camera is out. The week's ledger rows light up and count to 48, and the picture holds a moment. |
| 27.3–32.3 s | End card: the app icon (`app/resources/icon.png`), `adhd-todo`, one line (zh `看见每个 Agent 在做什么`, en `See what every agent is doing`), `github.com/QingYunA/adhd-todo`. |

The times follow from the app's own clocks (the scene's plan, the ledger lighting a row every 420 ms
and counting a step every 60 ms), so the window segment ends when the week's ledger has counted; the
exact figures are in `track.json`.

Cards fade in and out over 300 ms; the cuts into and out of the window are 200 ms crossfades.

## The data

Made up, in a Fake store; nothing from the user's database. Projects: `adhd-todo`, `picpocket`,
`blog`, `dotfiles`. Titles are short and believable, in the video's language.

| Who | Today | This week | PRs merged today |
| --- | --- | --- | --- |
| You | 3 | 9 | — |
| Claude Code | 4 | 13 | 3 |
| Codex | 3 | 9 | 2 |
| Gemini CLI | 2 | 6 | 1 |
| Cursor | 2 | 5 | 1 |
| GitHub Copilot | 1 | 4 | — |
| OpenCode | 1 | 2 | — |

The seventh, OpenCode, throws from off the right edge. Goods mix every kind (gem, jar, gear, book,
flask, mail, coin, scroll, crate). The week: Monday 8, Tuesday 6, Wednesday 10, Thursday 8, Friday
16 (today), Saturday and Sunday empty. The paragraphs are stored, not asked of a model.

## How it is made

- `docs/design/tools/promo/` holds the render: Go tests behind a build tag (`promo`, in `app/views`) that
  build the Fake store and drive the views with one virtual clock, advanced by 1/60 s, that the app and
  the renderer both read (so the scene, the paragraph's typing and the ledger's count follow it, and no
  production code changes). The test moves and clicks the pointer, sets the camera, and writes each
  finished frame as PNG; then a script assembles cards, frames and sound with ffmpeg.
- One command per language writes both sizes:
  `docs/design/tools/promo/render.sh zh` / `en`.
