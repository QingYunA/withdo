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
