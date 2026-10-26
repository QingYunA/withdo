package pngdiff

import (
	"image"
	"image/color"
	"testing"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestPicturesWithinTheToleranceAreTheSame(t *testing.T) {
	a := solid(4, 4, color.RGBA{100, 100, 100, 255})
	b := solid(4, 4, color.RGBA{100, 100, 100, 255})
	b.SetRGBA(1, 1, color.RGBA{101, 103, 98, 255})
	if !Same(a, b) {
		t.Fatal("a difference of 3 levels should count as the same")
	}
}

func TestPicturesPastTheToleranceDiffer(t *testing.T) {
	a := solid(4, 4, color.RGBA{100, 100, 100, 255})
	b := solid(4, 4, color.RGBA{100, 100, 100, 255})
	b.SetRGBA(2, 3, color.RGBA{100, 100, 104, 255})
	if Same(a, b) {
		t.Fatal("a difference of 4 levels should differ")
	}
	c := solid(4, 4, color.RGBA{100, 100, 100, 255})
	c.SetRGBA(0, 0, color.RGBA{100, 100, 100, 250})
	if Same(a, c) {
		t.Fatal("alpha counts as a channel")
	}
}

func TestPicturesOfDifferentBoundsDiffer(t *testing.T) {
	if Same(solid(4, 4, color.RGBA{}), solid(4, 5, color.RGBA{})) {
		t.Fatal("different bounds should differ")
	}
}
