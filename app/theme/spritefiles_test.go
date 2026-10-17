package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEveryAvatarKindHasAFileOfTwelveSquare(t *testing.T) {
	for kind := AvatarYou; kind <= AvatarOther; kind++ {
		name, ok := avatarFiles[kind]
		if !ok {
			t.Errorf("avatar kind %d names no file", kind)
			continue
		}
		checkSpriteFile(t, name+".png")
		if _, err := fs.Stat(spriteFS, "sprites/"+name+".dark.png"); err == nil {
			checkSpriteFile(t, name+".dark.png")
		}
	}
	checkSpriteFile(t, "avatars/parcel.png")
}

func checkSpriteFile(t *testing.T, file string) {
	t.Helper()
	data, err := spriteFS.ReadFile("sprites/" + file)
	if err != nil {
		t.Errorf("%s: %v", file, err)
		return
	}
	s, err := decodeSprite(data)
	if err != nil {
		t.Errorf("%s: %v", file, err)
		return
	}
	if s.Width() != AvatarGrid || s.Height() != AvatarGrid {
		t.Errorf("%s is %d x %d pixels, want %d x %d", file, s.Width(), s.Height(), AvatarGrid, AvatarGrid)
	}
}

func TestEveryFileUnderSpritesIsNamedByTheCode(t *testing.T) {
	named := spriteFileNames()
	var found []string
	err := fs.WalkDir(spriteFS, "sprites", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "README.md" {
			return err
		}
		found = append(found, strings.TrimPrefix(path, "sprites/"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		if !slices.Contains(named, f) {
			t.Errorf("sprites/%s is named by nothing: use it or delete it", f)
		}
	}
	for _, f := range named {
		if !slices.Contains(found, f) {
			t.Errorf("sprites/%s is named but not embedded", f)
		}
	}
}

func TestDecodeSpriteKeysDistinctColorsAndLeavesAlphaZeroEmpty(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	s, err := decodeSprite(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if s.Rows[0] != "a.b" {
		t.Fatalf("rows %q", s.Rows)
	}
	if s.Palette['a'] != (ui.Color{R: 10, G: 20, B: 30, A: 255}) || s.Palette['b'].A != 128 {
		t.Errorf("palette %v", s.Palette)
	}
	if _, err := decodeSprite([]byte("not a png")); err == nil {
		t.Error("garbage decoded")
	}
}

func TestTheRobotFileHoldsTheThreeColorsThatTakeATint(t *testing.T) {
	palette := avatarSprites[AvatarOther].Palette
	for i, shade := range botShades(ui.Hex(botTints[0])) {
		found := false
		for _, c := range palette {
			found = found || c == shade
		}
		if !found {
			t.Errorf("bot.png has no pixel of color %d of the gray robot (%v)", i, shade)
		}
	}
}
