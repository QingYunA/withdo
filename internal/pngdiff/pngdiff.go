// Package pngdiff compares rendered pictures with a small tolerance. The renderer rotates text
// (a dragged row's tilt) with antialiasing that, under parallel load, moves a pixel by one level in
// one channel between runs; nothing visible changes, but the 256-color palette of the PNG shifts
// and its bytes differ. Same treats such pictures as equal.
package pngdiff

import (
	"image"
	"image/png"
	"os"
)

// Tolerance is the most an 8-bit channel may differ and still count as the same.
const Tolerance = 3

// Same reports whether a and b have the same bounds and no 8-bit channel (red, green, blue or
// alpha) of any pixel differs by more than Tolerance.
func Same(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	bd := a.Bounds()
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if far(ar, br) || far(ag, bg) || far(ab, bb) || far(aa, ba) {
				return false
			}
		}
	}
	return true
}

// far reports whether two 16-bit channels are more than Tolerance apart in 8 bits.
func far(a, b uint32) bool {
	x, y := int(a>>8), int(b>>8)
	return x-y > Tolerance || y-x > Tolerance
}

// SameFiles decodes the PNGs at pathA and pathB and reports whether Same.
func SameFiles(pathA, pathB string) (bool, error) {
	a, err := decode(pathA)
	if err != nil {
		return false, err
	}
	b, err := decode(pathB)
	if err != nil {
		return false, err
	}
	return Same(a, b), nil
}

func decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
