# Promo video: the day's shipping

Decided 2026-10-18 (discussion of v21–v23). A short video of Summary for the README, the release page
and social posts: the day's shipping with a crowd of agents, then the week. Rendered offline frame by
frame from the app's own views, so it is smooth, reproducible and uses only made-up data.

## Format

- 16:9. Two sizes from one render: 3840 x 2160 and 1920 x 1080, H.264 MP4, 60 fps, about 28 s.
- Two languages: Chinese (`zh-Hans`, the window in Chinese) and English. Same frames otherwise.
- Sound: only the Summary scene's own sounds (`app/theme/sounds/*.wav`, at the times
  `theme.DayPlan.SoundCues` / `WeekPlan.SoundCues` give). No music, no voice.
- The video files are not committed. They go to `out/promo/` (ignored) and are attached to a
  release or the README by hand.

## The picture

- The window, rendered at 2x (2200 x 1440), with its rounded corners and a soft shadow, centred on
  a backdrop that fills the frame: the dusk scene (`scenes/dusk.png`) scaled up with nearest
  neighbour and dimmed, so the window is the subject.
- A pixel mouse pointer (a 12 x 12 arrow drawn at 3 pt a cell, white with a dark outline) moves
  between clicks on an eased path and dips a cell when it clicks. It is the user's pointer, so a
  pointer is right here (unlike a figure for Cursor).
- Title and end cards: the same backdrop, not dimmed, with native text (SF Pro / PingFang) and pixel
  art from `app/theme/sprites/`.

## Storyboard

| Time | What |
| --- | --- |
| 0–2.5 s | Title card: zh `你和 Agent，今天做了什么`, en `What you and your agents got done today`; the six figures stand in a row under it and hop once, one after another. |
| 2.5–4 s | Cut to the window on Mine. The pointer moves to `Summary` in the sidebar and clicks. |
| 4–12 s | Today, the scene plays (dusk sky): 7 people, 16 goods, the lid, lands, stars, everyone hops. |
| 12–16 s | The assistant's paragraph types in, the ledger rows light up and count. |
| 16–17 s | The pointer clicks `This week`. |
| 17–23 s | The week (night): about 48 goods fall into 7 stacks, Friday the highest, the busiest one hops. |
| 23–28 s | End card: the app icon (`app/resources/icon.png`), `adhd-todo`, one line (zh `看见每个 Agent 在做什么`, en `See what every agent is doing`), `github.com/QingYunA/adhd-todo`. |

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

- `docs/design/tools/promo/` holds the render: a Go test behind a build tag (`promo`) that builds the
  Fake store, drives the views with a clock the test advances by 1/60 s (the scene's `freeze`, and the
  same for the paragraph's typing and the ledger's count), and writes each frame as PNG; then a
  script assembles cards, frames, pointer and sound with ffmpeg.
- One command per language writes both sizes:
  `docs/design/tools/promo/render.sh zh` / `en`.
