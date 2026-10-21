package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/egoist/mygo/ui"
)

// traySprite is the picture in the notification area of Windows: a 12 x 12 file like an avatar.
var traySprite = loadSprite("tray")

// TrayIcon is the tray picture as a PNG, every cell of the sprite scale pixels square, so that the
// edges stay hard.
func TrayIcon(scale int) []byte {
	return traySpritePNG(scale, func(c ui.Color) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A} })
}

// TrayTemplateIcon is the tray picture for the macOS menu bar, which tints an image by its alpha
// alone: the dark cells of the sprite are drawn in black and the light ones, the check mark, are cut
// out, so the icon follows the menu bar's light or dark appearance.
func TrayTemplateIcon(scale int) []byte {
	return traySpritePNG(scale, func(c ui.Color) color.NRGBA {
		if (299*int(c.R)+587*int(c.G)+114*int(c.B))/1000 >= 128 {
			return color.NRGBA{}
		}
		return color.NRGBA{A: 255}
	})
}

// traySpritePNG draws the sprite with every cell scale pixels square, each cell in the color pixel
// gives it.
func traySpritePNG(scale int, pixel func(ui.Color) color.NRGBA) []byte {
	scale = max(1, scale)
	s := traySprite
	img := image.NewNRGBA(image.Rect(0, 0, s.Width()*scale, s.Height()*scale))
	for cy, row := range s.Rows {
		for cx, key := range []rune(row) {
			if key == '.' {
				continue
			}
			px := pixel(s.Palette[key])
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
