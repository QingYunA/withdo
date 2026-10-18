# Sprites

The pictures of the people and objects in the app: PNG files, embedded into the binary with
`go:embed` (`app/theme/spritefiles.go`). No pixel grid of a character or an object is written in Go,
so a picture changes by replacing its file.

## Rules

- **One pixel of the file is one cell of the sprite.** Export at 1x, never scaled.
- **Transparent is empty.** A fully transparent pixel is not drawn; any other pixel is drawn in its
  own color, alpha kept.
- **To replace a picture, give the new file the same name and the same size.** The code names files,
  not pixels; a test loads every file the code names and fails on a missing file or a wrong size, and
  on a file here that nothing names.
- **`<name>.dark.png`** is optional, the same size as `<name>.png`, and is drawn instead of it in dark
  mode. Use it for a picture that is itself dark and would vanish on a dark window.

## Files

| File | Size | Drawn |
|---|---|---|
| `avatars/you.png` | 12 x 12 | the person: task rows and cards, the inspector, the hand-off drag, empty states |
| `avatars/assistant.png` | 12 x 12 | the assistant (a ghost with a crown): the composer, its answers, empty states |
| `avatars/claude.png` | 12 x 12 | the Claude Code avatar |
| `avatars/codex.png` | 12 x 12 | the Codex avatar |
| `avatars/cursor.png` | 12 x 12 | the Cursor avatar |
| `avatars/gemini.png` | 12 x 12 | the Gemini CLI avatar |
| `avatars/opencode.png`, `avatars/opencode.dark.png` | 12 x 12 | the opencode avatar; the dark file in dark mode |
| `avatars/copilot.png`, `avatars/copilot.dark.png` | 12 x 12 | the Copilot (VS Code) avatar; the dark file in dark mode |
| `avatars/bot.png` | 12 x 12 | the robot of every agent without a drawing of its own (Qwen, Kimi, Windsurf, Zed, Cline, unknown agents), the Agents row of the sidebar and Magpie |
| `avatars/parcel.png` | 12 x 12 | the parcel of a hand-off, in the header of "Waiting for an agent" (a pixel of room all round) |

## The robot is a template

`avatars/bot.png` is drawn in gray (`#8E8E99`). Its three tinted colors are that gray (the body), the
gray darkened by 22% (the feet and sides) and the gray lightened by 45% (the antenna); the app
replaces them with the brand color of the agent, darkened and lightened the same way. Every other
color (the whites, the eyes) stays. Keep those three colors exactly when you redraw it.

## The Summary scene

The scene of the Summary page is made of files too. One pixel of a file is one cell of the scene
(3 pt); the scene is 264 x 80 cells, and every picture below is drawn on that grid. They carry a dark
outline, three or four shades lit from the top left and a one-cell drop shadow down and to the right.
They were baked once from `docs/design/mockups/v18.html`; the baking script is not kept, so edit the
PNGs.

| File | Size | Drawn |
|---|---|---|
| `actors/<who>/<pose>.png` | 24 x 24 | the people of the scene. `who` is `you`, `claude`, `codex` or `bot`; poses are `idle`, `lift`, `throw` and `hop`, and `idle2`, which an actor swaps with `idle` while it sways (Claude's claws, Codex's blink). A folder with no file for a pose draws its `idle`. The drop shadow is cut at the file's edge. |
| `actors/bot/*.png` | 24 x 24 | the robot of every agent without a folder of its own. It is a template in the gray of `avatars/bot.png`, tinted the same way (see below). |
| `goods/<kind>.png` | 17 x 17 | a finished task: `gem` (a feature), `jar` (a bug fix), `gear` (a refactor or migration), `book` (docs or design), `flask` (tests), `mail` (a reply), `coin` (money or a bill), `scroll` (a plan or research), `crate` (anything else). 16 x 16 cells and the shadow cell; the shadow is not drawn while a good is in the air. |
| `props/star.png` | 7 x 7 | the gold star of a good whose pull request merged |
| `props/bin.png` | 41 x 17 | the shipping bin (40 x 16 and the shadow cell) |
| `props/lid-open.png`, `props/lid-half.png`, `props/lid-shut.png` | 41 x 7, 41 x 4, 43 x 4 | the lid of the bin: the dark inside seen from behind, the lid half open, the lid shut |
| `scenes/day.png`, `scenes/dusk.png`, `scenes/night.png` | 264 x 80 | the backdrop: sky, hills, grass, path, fence and lamp. A wider column repeats the first and last column outward; a narrower one crops both sides equally. |

Not files: gold dust, fireflies, the `+1` digits, the dip of the bin and every motion; they are
effects, not characters or objects.

The robot of the scene follows the same rule as the avatar: keep its three tinted colors exactly (the
gray `#8E8E99`, that gray darkened by 22 % and lightened by 45 %).
