package theme

import (
	"hash/fnv"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// AvatarGrid is the side of an avatar in pixels; avatars are drawn at whole multiples of it where
// they can be (12, 24, 72) and at 18 pt in rows, which is 36 device pixels on a Retina screen.
const AvatarGrid = 12

// Avatar sizes, in points.
const (
	AvatarSmall float32 = 12 // calendar chips
	AvatarRow   float32 = 18 // list rows, cards, the inspector's activity
	AvatarLarge float32 = 60 // empty states: 5 points a pixel
)

// Sprite is a pixel drawing: each rune of a row is a palette key, '.' is transparent. The drawings of
// people and objects are decoded from the PNG files under sprites/.
type Sprite struct {
	Rows    []string
	Palette map[rune]ui.Color
	// dark, when set, is drawn instead in dark mode, for a drawing that is itself dark and would
	// vanish on a dark window.
	dark *Sprite
}

// inDark is the sprite as it is drawn in dark mode.
func (s *Sprite) inDark() *Sprite {
	if s.dark == nil {
		return s
	}
	return s.dark
}

// Width is the sprite's width in pixels.
func (s *Sprite) Width() int { return len(s.Rows[0]) }

// Height is the sprite's height in pixels.
func (s *Sprite) Height() int { return len(s.Rows) }

// Who is the author an avatar stands for.
type Who struct {
	Kind   AvatarKind
	sprite *Sprite
}

// AvatarKind names the known authors.
type AvatarKind int

const (
	AvatarYou AvatarKind = iota
	AvatarAssistant
	AvatarClaude
	AvatarCodex
	AvatarCursor
	AvatarGemini
	AvatarOpencode
	AvatarCopilot
	AvatarQwen
	AvatarKimi
	AvatarWindsurf
	AvatarZed
	AvatarCline
	AvatarOther
)

var (
	spriteYou       = avatarSprites[AvatarYou]
	spriteAssistant = avatarSprites[AvatarAssistant]
	spriteClaude    = avatarSprites[AvatarClaude]
	spriteCodex     = avatarSprites[AvatarCodex]
	spriteCursor    = avatarSprites[AvatarCursor]
	spriteGemini    = avatarSprites[AvatarGemini]
	spriteOpencode  = avatarSprites[AvatarOpencode]
	spriteCopilot   = avatarSprites[AvatarCopilot]
	// The parcel of a hand-off: a brown box with a tape across it, with a pixel of room all round.
	spriteParcel = loadSprite("avatars/parcel")
	// The agents without a drawing of their own get the robot in the color of their brand.
	spriteQwen     = tintedBot("#615CED")
	spriteKimi     = tintedBot("#1F6FEB")
	spriteWindsurf = tintedBot("#0EA5A4")
	spriteZed      = tintedBot("#3F9E4D")
	spriteCline    = tintedBot("#D9488B")
	// botTints color the robot of an agent the app does not know, by its name, so two unknown
	// agents still look different. The first is the gray of an agent with no name, which is the
	// color the robot's file is drawn in.
	botTints = []string{"#8E8E99", "#0EA5A4", "#D9488B", "#E08A1E", "#3F9E4D", "#6B7CD9"}
)

var (
	You       = Who{Kind: AvatarYou, sprite: spriteYou}
	Assistant = Who{Kind: AvatarAssistant, sprite: spriteAssistant}
)

// pal parses "k#RRGGBB k#RRGGBB" into a palette.
func pal(spec string) map[rune]ui.Color {
	out := map[rune]ui.Color{}
	for _, f := range strings.Fields(spec) {
		out[rune(f[0])] = ui.Hex(f[1:])
	}
	return out
}

// AgentAvatar is the avatar of an agent by the name it reported under (the MCP clientInfo name,
// such as "claude-code"). Unknown names get a robot whose color follows the name.
func AgentAvatar(name string) Who {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "claude"):
		return Who{Kind: AvatarClaude, sprite: spriteClaude}
	case strings.Contains(n, "codex"):
		return Who{Kind: AvatarCodex, sprite: spriteCodex}
	case strings.Contains(n, "cursor"):
		return Who{Kind: AvatarCursor, sprite: spriteCursor}
	case strings.Contains(n, "gemini"):
		return Who{Kind: AvatarGemini, sprite: spriteGemini}
	case strings.Contains(n, "opencode"):
		return Who{Kind: AvatarOpencode, sprite: spriteOpencode}
	case strings.Contains(n, "copilot") || strings.Contains(n, "visual studio code"):
		return Who{Kind: AvatarCopilot, sprite: spriteCopilot}
	case strings.Contains(n, "qwen"):
		return Who{Kind: AvatarQwen, sprite: spriteQwen}
	case strings.Contains(n, "kimi"):
		return Who{Kind: AvatarKimi, sprite: spriteKimi}
	case strings.Contains(n, "windsurf"):
		return Who{Kind: AvatarWindsurf, sprite: spriteWindsurf}
	case n == "zed" || strings.HasPrefix(n, "zed ") || strings.HasPrefix(n, "zed-"):
		return Who{Kind: AvatarZed, sprite: spriteZed}
	case strings.Contains(n, "cline"):
		return Who{Kind: AvatarCline, sprite: spriteCline}
	}
	return Who{Kind: AvatarOther, sprite: bot(name)}
}

func bot(name string) *Sprite {
	tint := botTints[0]
	if name != "" && name != "agent" {
		h := fnv.New32a()
		_, _ = h.Write([]byte(strings.ToLower(name)))
		tint = botTints[1+int(h.Sum32()%uint32(len(botTints)-1))]
	}
	return botIn(ui.Hex(tint))
}

// tintedBot is the robot in one color, given as hex.
func tintedBot(tint string) *Sprite { return botIn(ui.Hex(tint)) }

// botIn is the robot in one color. Its file is drawn in gray: that gray, its shade (the feet and
// sides) and its light (the antenna) become base, base shaded and base lightened; the whites and the
// eyes keep their colors.
func botIn(base ui.Color) *Sprite {
	template := avatarSprites[AvatarOther]
	from, to := botShades(ui.Hex(botTints[0])), botShades(base)
	pal := make(map[rune]ui.Color, len(template.Palette))
	for k, c := range template.Palette {
		pal[k] = c
		for i := range from {
			if c == from[i] {
				pal[k] = to[i]
			}
		}
	}
	return &Sprite{Rows: template.Rows, Palette: pal}
}

// botShades is the colors of the robot that follow its tint: the body, the shade and the light.
func botShades(base ui.Color) [3]ui.Color {
	return [3]ui.Color{base, base.Mix(ui.Hex("#000000"), 0.22), base.Mix(ui.Hex("#FFFFFF"), 0.45)}
}

// AccentBot is the robot in the accent color: the sidebar's Agents row, which stands for all of
// them. The sprites are made once per color.
func AccentBot(accent ui.Color) Who {
	accentBots.mu.Lock()
	defer accentBots.mu.Unlock()
	sp, ok := accentBots.by[accent]
	if !ok {
		sp = botIn(accent)
		accentBots.by[accent] = sp
	}
	return Who{Kind: AvatarOther, sprite: sp}
}

var accentBots = struct {
	mu sync.Mutex
	by map[ui.Color]*Sprite
}{by: map[ui.Color]*Sprite{}}

// Sprite returns the drawing behind an avatar.
func (w Who) Sprite() *Sprite {
	if w.sprite == nil {
		return spriteYou
	}
	return w.sprite
}

// AvatarIcon draws an avatar size points square.
func AvatarIcon(c *ui.Context, w Who, size float32) ui.Element {
	return SpriteIcon(c, w.Sprite(), size)
}

// SpriteIcon draws a sprite size points tall, as wide as its pixels make it.
func SpriteIcon(c *ui.Context, s *Sprite, size float32) ui.Element {
	u := size / float32(s.Height())
	if Resolve(c).Dark {
		s = s.inDark()
	}
	return ui.Box(c).Size(u*float32(s.Width()), size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, r, s)
	})
}

// DrawSprite fills r with the sprite's pixels, runs of one color as one rectangle. A rectangle that
// is not in the sprite's proportion stretches its pixels: the hop squashes the sprite this way.
func DrawSprite(p *ui.Painter, r ui.Rect, s *Sprite) {
	ux, uy := r.W/float32(s.Width()), r.H/float32(s.Height())
	for y, row := range s.Rows {
		runes := []rune(row)
		for x := 0; x < len(runes); {
			ch := runes[x]
			end := x + 1
			for end < len(runes) && runes[end] == ch {
				end++
			}
			if col, ok := s.Palette[ch]; ok {
				p.Fill(ui.Rect{X: r.X + float32(x)*ux, Y: r.Y + float32(y)*uy, W: float32(end-x) * ux, H: uy}, col, 0)
			}
			x = end
		}
	}
}

// DuoIcon is you and the assistant side by side, size points tall, for empty states.
func DuoIcon(c *ui.Context, size float32) ui.Element {
	return ui.Row(c).Gap(size / 6).Children(func() {
		AvatarIcon(c, You, size)
		AvatarIcon(c, Assistant, size)
	})
}

// Parcel is the drawing of a task in hand, at ParcelSize points (2 pt a pixel).
const ParcelSize float32 = 24

// ParcelGlyph is the parcel in the 18 pt slot of a group header: Waiting for an agent.
func ParcelGlyph(c *ui.Context) ui.Element {
	return ui.Box(c).Size(GlyphSize, GlyphSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawParcel(p, ui.Rect{X: r.X + (r.W-ParcelSize)/2, Y: r.Y + (r.H-ParcelSize)/2, W: ParcelSize, H: ParcelSize})
	})
}

// DrawParcel fills r with the parcel.
func DrawParcel(p *ui.Painter, r ui.Rect) { DrawSprite(p, r, spriteParcel) }
