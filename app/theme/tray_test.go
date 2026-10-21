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

// The menu bar tints an image by its alpha alone, so the template is black where the sprite is dark
// and cut out where it is light (the check mark).
func TestTrayTemplateIconIsBlackWithTheMarkCutOut(t *testing.T) {
	const scale = 2
	img, _, err := image.Decode(bytes.NewReader(TrayTemplateIcon(scale)))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != AvatarGrid*scale || b.Dy() != AvatarGrid*scale {
		t.Fatalf("%d x %d pixels, want %d square", b.Dx(), b.Dy(), AvatarGrid*scale)
	}
	solid, cut := 0, 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			switch {
			case c == color.NRGBA{A: 255}:
				solid++
			case c.A == 0:
				cut++
			default:
				t.Fatalf("pixel (%d,%d) = %v: a template is black or clear", x, y, c)
			}
		}
	}
	// The corners are clear in the sprite already; the mark adds clear cells inside it.
	corners := 0
	for _, row := range traySprite.Rows {
		for _, key := range row {
			if key == '.' {
				corners++
			}
		}
	}
	if cut <= corners*scale*scale || solid == 0 {
		t.Errorf("%d clear and %d black pixels: the check mark is not cut out", cut, solid)
	}
}
