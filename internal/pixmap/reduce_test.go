package pixmap

import (
	"image"
	"image/color"
	"testing"
)

// testPalette is the palette of the brief, in the order of the letters a to l.
var testPalette = func() (p [Colors]color.RGBA) {
	for i, hex := range []uint32{0x5B5BD6, 0xE0A800, 0x30A46C, 0xE5484D, 0x3B82F6, 0xD9488B, 0x0EA5A4, 0xE08A1E, 0xFFFFFF, 0x8E8E93, 0x1C1C1E, 0x8A5A3B} {
		p[i] = color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}
	}
	return p
}()

var (
	red  = color.NRGBA{0xE5, 0x48, 0x4D, 255} // letter d
	blue = color.NRGBA{0x3B, 0x82, 0xF6, 255} // letter e
)

// canvas is a w x h picture, transparent or filled with bg, that fill paints on.
func canvas(w, h int, bg color.NRGBA, fill func(x, y int) (color.NRGBA, bool)) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := bg
			if f, ok := fill(x, y); ok {
				c = f
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// inset is the 80 x 80 square in the middle of the 100 x 100 pictures below.
func inset(x, y int) bool { return x >= 10 && x < 90 && y >= 10 && y < 90 }

func paint(p Picture) string {
	s := ""
	for y := 0; y < Size; y++ {
		s += p.Grid()[y*Size:(y+1)*Size] + "\n"
	}
	return s
}

func TestATransparentShapeFillsTheTenByTenInTheMiddleOfTheGrid(t *testing.T) {
	// A 30 x 30 red square in the corner of a 100 x 100 transparent picture: the crop finds it, the
	// grid is 12 x 12 with a one-cell margin, and one color for every cell becomes Own.
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) { return red, x < 30 && y < 30 })
	r, ok := Reduce(img)
	if !ok {
		t.Fatal("a shape on transparency has ink")
	}
	p := r.Picture(r.Otsu(), testPalette)
	if p.Count() != 100 || p.At(0, 0) != 0 || p.At(1, 1) != Own || p.At(10, 10) != Own || p.At(11, 11) != 0 {
		t.Errorf("a square fills the 10 x 10 with Own pixels:\n%s", paint(p))
	}
}

func TestAShapeInAPhotoIsFoundOnItsFlatBackground(t *testing.T) {
	// A dark mug on beige with noise: the background is the median of the border, ink is the distance
	// from it, and the cells of the background stay empty.
	seed := uint32(7)
	noise := func() int { seed = seed*1664525 + 1013904223; return int(seed>>24)%17 - 8 }
	bg := color.NRGBA{220, 205, 180, 255}
	img := image.NewNRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			c := bg
			if x >= 30 && x < 90 && y >= 40 && y < 100 {
				c = color.NRGBA{60, 50, 40, 255}
			}
			n := noise()
			img.SetNRGBA(x, y, color.NRGBA{uint8(int(c.R) + n), uint8(int(c.G) + n), uint8(int(c.B) + n), 255})
		}
	}
	r, ok := Reduce(img)
	if !ok {
		t.Fatal("no ink")
	}
	p := r.Picture(r.Otsu(), testPalette)
	if p.Count() != 100 || p.At(1, 1) != Own || p.At(0, 6) != 0 {
		t.Errorf("a dark square on a flat background is the 10 x 10:\n%s", paint(p))
	}
	// A narrower shape leaves the margin of its own square empty.
	narrow := canvas(120, 120, bg, func(x, y int) (color.NRGBA, bool) {
		return color.NRGBA{60, 50, 40, 255}, x >= 50 && x < 70 && y >= 20 && y < 100
	})
	r, _ = Reduce(narrow)
	p = r.Picture(r.Otsu(), testPalette)
	if p.At(5, 3) == 0 || p.At(5, 8) == 0 || p.At(1, 5) != 0 || p.At(10, 5) != 0 {
		t.Errorf("a tall bar stays in the middle columns:\n%s", paint(p))
	}
}

func TestABlankPictureHasNothingToTrace(t *testing.T) {
	if _, ok := Reduce(canvas(40, 40, color.NRGBA{255, 255, 255, 255}, func(x, y int) (color.NRGBA, bool) { return color.NRGBA{}, false })); ok {
		t.Error("a flat picture has ink")
	}
	if _, ok := Reduce(canvas(40, 40, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) { return color.NRGBA{}, false })); ok {
		t.Error("a transparent picture has ink")
	}
}

func TestTwoColorsMapToTwoPaletteLetters(t *testing.T) {
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) {
		if x < 50 {
			return red, inset(x, y)
		}
		return blue, inset(x, y)
	})
	r, _ := Reduce(img)
	p := r.Picture(r.Otsu(), testPalette)
	if p.At(2, 5) != 'd' || p.At(9, 5) != 'e' || p.At(5, 5) != 'd' || p.At(6, 5) != 'e' {
		t.Errorf("red is letter d and blue letter e:\n%s", paint(p))
	}
}

func TestAColorThatCoversEightyPercentTurnsIntoTheCategoryColor(t *testing.T) {
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) {
		if x < 82 { // 72 of the 80 columns
			return red, inset(x, y)
		}
		return blue, inset(x, y)
	})
	r, _ := Reduce(img)
	p := r.Picture(r.Otsu(), testPalette)
	if p.At(2, 5) != Own || p.At(10, 5) != 'e' || p.Count() != 100 {
		t.Errorf("90 percent red is Own and the rest keeps its letter:\n%s", paint(p))
	}
	// 70 percent is not enough.
	img = canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) {
		if x < 66 { // 56 of the 80 columns
			return red, inset(x, y)
		}
		return blue, inset(x, y)
	})
	r, _ = Reduce(img)
	if p = r.Picture(r.Otsu(), testPalette); p.At(2, 5) != 'd' {
		t.Errorf("70 percent keeps its letter:\n%s", paint(p))
	}
}

func TestAPhotoColorTakesTheNearestPaletteColor(t *testing.T) {
	// Near-black is the palette's black, not its brown or gray, and the 80 percent rule makes it Own.
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) {
		if x < 50 {
			return color.NRGBA{40, 40, 44, 255}, inset(x, y)
		}
		return color.NRGBA{150, 98, 62, 255}, inset(x, y) // a brown, not quite the palette's
	})
	r, _ := Reduce(img)
	p := r.Picture(r.Otsu(), testPalette)
	if p.At(2, 5) != 'k' || p.At(9, 5) != 'l' {
		t.Errorf("black and brown:\n%s", paint(p))
	}
}

func TestTheThresholdDecidesHowFaintACellMayBe(t *testing.T) {
	// The left half is solid, the right half is 30 percent opaque.
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) {
		if x < 50 {
			return red, inset(x, y)
		}
		return color.NRGBA{red.R, red.G, red.B, 77}, inset(x, y)
	})
	r, _ := Reduce(img)
	if got := r.Otsu(); got <= 30 || got >= 100 {
		t.Errorf("Otsu = %d, between the two inks", got)
	}
	if got := r.Picture(r.Otsu(), testPalette).Count(); got != 50 {
		t.Errorf("at Otsu only the solid half stays, got %d", got)
	}
	if got := r.Picture(20, testPalette).Count(); got != 100 {
		t.Errorf("at 20 percent the faint half stays too, got %d", got)
	}
	if got := r.Picture(100, testPalette).Count(); got != 50 {
		t.Errorf("at 100 percent a solid cell still stays, got %d", got)
	}
}

func TestOutlineOfAnImportedPictureIsHollow(t *testing.T) {
	img := canvas(100, 100, color.NRGBA{}, func(x, y int) (color.NRGBA, bool) { return red, inset(x, y) })
	r, _ := Reduce(img)
	if got := r.Picture(r.Otsu(), testPalette).Outline().Count(); got != 36 {
		t.Errorf("the ring of a 10 x 10 square is 36 cells, got %d", got)
	}
}
