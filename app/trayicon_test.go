package main

import (
	"bytes"
	"image/png"
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
