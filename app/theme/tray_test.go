package theme

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"testing"
)

func TestTrayIconIsTheSpriteScaledUpWhole(t *testing.T) {
	for _, scale := range []int{1, 2, 3} {
		img, _, err := image.Decode(bytes.NewReader(TrayIcon(scale)))
		if err != nil {
			t.Fatalf("scale %d: %v", scale, err)
		}
		n := AvatarGrid * scale
		if b := img.Bounds(); b.Dx() != n || b.Dy() != n {
			t.Fatalf("scale %d: %d x %d pixels, want %d x %d", scale, b.Dx(), b.Dy(), n, n)
		}
		opaque := 0
		for cy, row := range traySprite.Rows {
			for cx, key := range []rune(row) {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						got := color.NRGBAModel.Convert(img.At(cx*scale+dx, cy*scale+dy)).(color.NRGBA)
						want := color.NRGBA{}
						if key != '.' {
							c := traySprite.Palette[key]
							want = color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
						}
						if got != want {
							t.Fatalf("scale %d: pixel (%d,%d) = %v, want %v", scale, cx*scale+dx, cy*scale+dy, got, want)
						}
					}
				}
				if key != '.' {
					opaque++
				}
			}
		}
		if opaque == 0 || opaque == AvatarGrid*AvatarGrid {
			t.Errorf("the tray sprite has %d opaque cells of %d: its corners should be empty and its middle drawn", opaque, AvatarGrid*AvatarGrid)
		}
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
			t.Errorf("scale %d: the corner is not transparent", scale)
		}
	}
}
