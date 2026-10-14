package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestTrayIconsAreTemplateGlyphs(t *testing.T) {
	plain, blocked := trayIcon(false), trayIcon(true)
	if bytes.Equal(plain, blocked) {
		t.Fatal("the blocked icon looks like the plain one")
	}
	for name, data := range map[string][]byte{"plain": plain, "blocked": blocked} {
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
			t.Errorf("%s is %v, want 32x32", name, b)
		}
		_, _, _, a := img.At(16, 3).RGBA() // on the ring
		if a == 0 {
			t.Errorf("%s has no ring at the top", name)
		}
	}
	img, _ := png.Decode(bytes.NewReader(blocked))
	if _, _, _, a := img.At(25, 7).RGBA(); a == 0 {
		t.Error("the blocked icon has no dot")
	}
}

// TestScreensTray writes the tray glyphs, enlarged on menu bars, for the design review.
func TestScreensTray(t *testing.T) {
	dir := os.Getenv("SCREENS_OUT")
	if dir == "" {
		t.Skip("SCREENS_OUT is not set")
	}
	const scale, cell = 6, 32
	out := image.NewRGBA(image.Rect(0, 0, cell*scale*4, cell*scale))
	for i, tc := range []struct {
		blocked  bool
		bar, ink color.RGBA
	}{
		{false, color.RGBA{236, 236, 240, 255}, color.RGBA{30, 30, 32, 255}},
		{true, color.RGBA{236, 236, 240, 255}, color.RGBA{30, 30, 32, 255}},
		{false, color.RGBA{44, 44, 48, 255}, color.RGBA{240, 240, 242, 255}},
		{true, color.RGBA{44, 44, 48, 255}, color.RGBA{240, 240, 242, 255}},
	} {
		img, _ := png.Decode(bytes.NewReader(trayIcon(tc.blocked)))
		for y := 0; y < cell*scale; y++ {
			for x := 0; x < cell*scale; x++ {
				_, _, _, a := img.At(x/scale, y/scale).RGBA()
				k := float64(a) / 65535
				mix := func(bg, fg uint8) uint8 { return uint8(float64(bg)*(1-k) + float64(fg)*k) }
				out.SetRGBA(i*cell*scale+x, y, color.RGBA{mix(tc.bar.R, tc.ink.R), mix(tc.bar.G, tc.ink.G), mix(tc.bar.B, tc.ink.B), 255})
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "tray-icons.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		t.Fatal(err)
	}
}
