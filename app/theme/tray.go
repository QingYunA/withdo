package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// traySprite is the picture in the notification area of Windows: a 12 x 12 file like an avatar.
var traySprite = loadSprite("tray")

// TrayIcon is the tray picture as a PNG, every cell of the sprite scale pixels square, so that the
// edges stay hard.
func TrayIcon(scale int) []byte {
	scale = max(1, scale)
	s := traySprite
	img := image.NewNRGBA(image.Rect(0, 0, s.Width()*scale, s.Height()*scale))
	for cy, row := range s.Rows {
		for cx, key := range []rune(row) {
			if key == '.' {
				continue
			}
			c := s.Palette[key]
			px := color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetNRGBA(cx*scale+dx, cy*scale+dy, px)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("theme: " + err.Error())
	}
	return buf.Bytes()
}
