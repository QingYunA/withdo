package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// trayIcon draws the menu bar glyph: a circle with a small check, outline style, black on
// transparent for macOS to tint. With blocked set, a filled dot sits at the top right, cut out of
// the ring so it stays crisp. The canvas is 32 px, a 16 pt icon at 2x.
func trayIcon(blocked bool) []byte {
	const size, samples = 32, 4
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			hit := 0
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := float64(x) + (float64(sx)+0.5)/samples
					py := float64(y) + (float64(sy)+0.5)/samples
					if inked(px, py, blocked) {
						hit++
					}
				}
			}
			if hit > 0 {
				img.SetNRGBA(x, y, color.NRGBA{A: uint8(255 * hit / (samples * samples))})
			}
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// inked reports whether the point is part of the glyph.
func inked(x, y float64, blocked bool) bool {
	const dotX, dotY, dotR, gap = 25.2, 6.8, 5.0, 3.2
	if blocked {
		if math.Hypot(x-dotX, y-dotY) <= dotR {
			return true
		}
		if math.Hypot(x-dotX, y-dotY) <= dotR+gap {
			return false
		}
	}
	if math.Abs(math.Hypot(x-16, y-16)-12.6) <= 1.15 {
		return true
	}
	return segDist(x, y, 10.2, 16.6, 14.4, 20.8) <= 1.2 || segDist(x, y, 14.4, 20.8, 22.2, 11.6) <= 1.2
}

// segDist is the distance from (px, py) to the segment (ax, ay)-(bx, by).
func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}
