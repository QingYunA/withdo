package theme

import (
	"hash/fnv"
	"strings"

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

// Sprite is a pixel drawing: each rune of a row is a palette key, '.' is transparent.
type Sprite struct {
	Rows    []string
	Palette map[rune]ui.Color
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
	AvatarOther
)

var (
	spriteYou = &Sprite{Palette: pal("h#3A3550 s#F3C7A0 e#2B2838 b#5B5BD6 d#4646B8 c#E88B8B"), Rows: []string{
		"...hhhhhh...",
		"..hhhhhhhh..",
		".hhhhhhhhhh.",
		".hhssssssh..",
		".hsesssses..",
		"..ssssssss..",
		"..scsssscs..",
		"...ssssss...",
		"..bbbbbbbb..",
		".bbbbddbbbb.",
		".bbbbddbbbb.",
		".bbbbbbbbbb.",
	}}
	spriteAssistant = &Sprite{Palette: pal("a#F5C542 p#7C6CF0 d#5B4FD0 w#FFFFFF e#231F3A k#F0A3C0"), Rows: []string{
		".....a......",
		"....aaa.....",
		".....a......",
		"...pppppp...",
		"..pppppppp..",
		".pppwwppwwp.",
		".pppweppwep.",
		".pkpppppppk.",
		".pppppddppp.",
		".pppppppppp.",
		".pp.pp.pp.p.",
		".p..p..p..p.",
	}}
	spriteClaude = &Sprite{Palette: pal("o#D97757 d#B85C3E e#2B1D17"), Rows: []string{
		"............",
		"............",
		"..oooooooo..",
		"..oooooooo..",
		"..oeooooeo..",
		"oooeooooeooo",
		"oooooooooooo",
		"..oooooooo..",
		"..oooooooo..",
		"..d.d..d.d..",
		"..d.d..d.d..",
		"............",
	}}
	spriteCodex = &Sprite{Palette: pal("b#5B7CFA d#4A5EE0 w#FFFFFF"), Rows: []string{
		"............",
		"...bbb.bbb..",
		"..bbbbbbbbb.",
		".bbbbbbbbbbb",
		".bwbbbbbbbbb",
		"bbbwbbbbbbbb",
		"bbbbwbbbbbbb",
		"bbbwbbbbbbbb",
		"bbwbbbwwwbbb",
		".ddddddddddd",
		"..dddd.dddd.",
		"............",
	}}
	spriteCursor = &Sprite{Palette: pal("k#26262B w#FFFFFF"), Rows: []string{
		".k..........",
		".kk.........",
		".kwk........",
		".kwwk.......",
		".kwwwk......",
		".kwwwwk.....",
		".kwwwwwk....",
		".kwwwwwwk...",
		".kwwwkkkk...",
		".kwkwk......",
		".kk.kwk.....",
		".....kk.....",
	}}
	spriteGemini = &Sprite{Palette: pal("b#4F8EF7 v#9B6CF0 l#8FC4FF"), Rows: []string{
		".....b......",
		".....b......",
		"....bbb.....",
		"....bbb.....",
		"..lbbbbbv...",
		"lllbbbbbvvv.",
		"..lbbbbbv...",
		"....vvv.....",
		"....vvv.....",
		".....v......",
		".....v......",
		"............",
	}}
	botRows = []string{
		".....a......",
		".....m......",
		"..mmmmmmmm..",
		"..mmmmmmmm..",
		".dmwwmmwwmd.",
		".dmwemmwemd.",
		".dmmmmmmmmd.",
		"..mmeeeemm..",
		"..mmmmmmmm..",
		"...dddddd...",
		"..dd....dd..",
		"............",
	}
	// botTints color the robot of an agent the app does not know, by its name, so two unknown
	// agents still look different. The first is the gray of an agent with no name.
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
	base := ui.Hex(tint)
	return &Sprite{Rows: botRows, Palette: map[rune]ui.Color{
		'm': base, 'd': base.Mix(ui.Hex("#000000"), 0.22), 'a': base.Mix(ui.Hex("#FFFFFF"), 0.45),
		'w': ui.Hex("#FFFFFF"), 'e': ui.Hex("#2A2A33"),
	}}
}

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
	return ui.Box(c).Size(u*float32(s.Width()), size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		DrawSprite(p, r, s)
	})
}

// DrawSprite fills r with the sprite's pixels, runs of one color as one rectangle.
func DrawSprite(p *ui.Painter, r ui.Rect, s *Sprite) {
	u := r.H / float32(s.Height())
	for y, row := range s.Rows {
		runes := []rune(row)
		for x := 0; x < len(runes); {
			ch := runes[x]
			end := x + 1
			for end < len(runes) && runes[end] == ch {
				end++
			}
			if col, ok := s.Palette[ch]; ok {
				p.Fill(ui.Rect{X: r.X + float32(x)*u, Y: r.Y + float32(y)*u, W: float32(end-x) * u, H: u}, col, 0)
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
