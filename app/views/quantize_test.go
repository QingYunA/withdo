package views

import (
	"image"
	"image/color"
	"testing"
)

// colorDist is the largest per-channel gap between two opaque colors.
func colorDist(a, b color.Color) int {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	d := 0
	for _, p := range [][2]uint32{{ar, br}, {ag, bg}, {ab, bb}} {
		g := int(p[0]>>8) - int(p[1]>>8)
		if g < 0 {
			g = -g
		}
		if g > d {
			d = g
		}
	}
	return d
}

func TestQuantizeKeepsASmallPaletteExactly(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			// 200 distinct colors, none of them repeated in a way that would let a bad
			// quantizer merge them
			n := (x + y*40) % 200
			src.SetRGBA(x, y, color.RGBA{uint8(n), uint8(255 - n), uint8(n * 7 % 256), 255})
		}
	}
	out := quantizeScreen(src)
	if len(out.Palette) > 256 {
		t.Fatalf("palette has %d colors", len(out.Palette))
	}
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			if d := colorDist(src.At(x, y), out.At(x, y)); d != 0 {
				t.Fatalf("pixel %d,%d changed by %d", x, y, d)
			}
		}
	}
}

func TestQuantizeGradientStaysClose(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y * 2), uint8((x + y) / 2), 255})
		}
	}
	out := quantizeScreen(src)
	if len(out.Palette) > 256 {
		t.Fatalf("palette has %d colors", len(out.Palette))
	}
	worst := 0
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			if d := colorDist(src.At(x, y), out.At(x, y)); d > worst {
				worst = d
			}
		}
	}
	if worst > 24 {
		t.Fatalf("a pixel is %d levels off", worst)
	}
}

func TestQuantizeIsDeterministic(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			src.SetRGBA(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), uint8(x * y % 256), 255})
		}
	}
	a, b := quantizeScreen(src), quantizeScreen(src)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatal("two runs picked different indexes")
		}
	}
	for i := range a.Palette {
		if a.Palette[i] != b.Palette[i] {
			t.Fatal("two runs picked different palettes")
		}
	}
}
