package theme

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // registers the PNG decoder
	"io/fs"
	"sort"

	"github.com/egoist/mygo/ui"
)

// The pictures of people and objects are PNG files under sprites/ (see sprites/README.md): one
// pixel of the file is one cell of the sprite, a transparent pixel is empty, and any other pixel is
// drawn in its own color. To change a picture, replace its file with one of the same name and size.
//
//go:embed sprites
var spriteFS embed.FS

// spriteFiles records every file the code has named, for the test that no file goes unused.
var spriteFiles []string

// loadSprite decodes sprites/<name>.png, and sprites/<name>.dark.png when it exists, which is drawn
// instead in dark mode. A file that is missing or cannot be decoded is a mistake in the program.
func loadSprite(name string) *Sprite {
	s := mustDecodeSprite(name + ".png")
	if _, err := fs.Stat(spriteFS, "sprites/"+name+".dark.png"); err == nil {
		s.dark = mustDecodeSprite(name + ".dark.png")
	}
	return s
}

func mustDecodeSprite(file string) *Sprite {
	data, err := spriteFS.ReadFile("sprites/" + file)
	if err != nil {
		panic("theme: " + err.Error())
	}
	s, err := decodeSprite(data)
	if err != nil {
		panic(fmt.Sprintf("theme: sprites/%s: %v", file, err))
	}
	spriteFiles = append(spriteFiles, file)
	return s
}

// decodeSprite turns a PNG into a sprite: each distinct color is a palette key, in the order the
// colors first appear, and a pixel with no alpha is '.'.
func decodeSprite(data []byte) (*Sprite, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	keys := map[ui.Color]rune{}
	s := &Sprite{Palette: map[rune]ui.Color{}}
	next := 'a'
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := make([]rune, 0, b.Dx())
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				row = append(row, '.')
				continue
			}
			col := ui.Color{R: c.R, G: c.G, B: c.B, A: c.A}
			k, ok := keys[col]
			if !ok {
				k, next = next, next+1
				keys[col] = k
				s.Palette[k] = col
			}
			row = append(row, k)
		}
		s.Rows = append(s.Rows, string(row))
	}
	return s, nil
}

// avatarFiles names the file behind each avatar. The agents without a drawing of their own, and
// AvatarOther, share the robot, which is tinted (see botIn).
var avatarFiles = map[AvatarKind]string{
	AvatarYou:       "avatars/you",
	AvatarAssistant: "avatars/assistant",
	AvatarClaude:    "avatars/claude",
	AvatarCodex:     "avatars/codex",
	AvatarCursor:    "avatars/cursor",
	AvatarGemini:    "avatars/gemini",
	AvatarOpencode:  "avatars/opencode",
	AvatarCopilot:   "avatars/copilot",
	AvatarQwen:      "avatars/bot",
	AvatarKimi:      "avatars/bot",
	AvatarWindsurf:  "avatars/bot",
	AvatarZed:       "avatars/bot",
	AvatarCline:     "avatars/bot",
	AvatarOther:     "avatars/bot",
}

// avatarSprites is the sprite of each avatar kind. The kinds that share a file share the sprite.
var avatarSprites = func() map[AvatarKind]*Sprite {
	byFile := map[string]*Sprite{}
	out := make(map[AvatarKind]*Sprite, len(avatarFiles))
	for kind, name := range avatarFiles {
		if byFile[name] == nil {
			byFile[name] = loadSprite(name)
		}
		out[kind] = byFile[name]
	}
	return out
}()

// spriteFileNames lists every file the code names, sorted: the avatars it has loaded and the files
// of the Summary scene.
func spriteFileNames() []string {
	out := append([]string(nil), spriteFiles...)
	for name := range sceneFileSizes() {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
