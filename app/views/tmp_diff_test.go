package views

import (
	"image"
	"os"
	"testing"
	"time"
)

func dragImg(t *testing.T) *image.RGBA {
	r := newHandRig(t, false)
	r.tt.SetScale(2)
	r.tt.SetDark(true)
	inChinese(r.tt)
	r.tt.SetSize(1100, 820)
	r.tt.Frame()
	x, y := r.grab(t, "Write the release notes")
	for i := 1; i <= 3; i++ {
		r.tt.Move(x+float32(30*i), y+float32(40*i))
		r.after(70 * time.Millisecond)
	}
	return r.tt.Image()
}

func cmpImg(t *testing.T, label string, a, b *image.RGBA) {
	n := 0
	bd := a.Bounds()
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	t.Logf("%s: diff px %d", label, n)
}

func TestTmpDiff(t *testing.T) {
	a := dragImg(t)
	cmpImg(t, "again", a, dragImg(t))
	switch os.Getenv("TMPX") {
	case "select":
		TestScreensSelect(t)
	case "calendar":
		TestScreensCalendar(t)
	case "size":
		r := newHandRig(t, false)
		r.tt.SetScale(2)
		r.tt.SetSize(1100, 900)
		r.tt.Frame()
	case "sizedrag":
		r := newHandRig(t, false)
		r.tt.SetScale(2)
		r.tt.SetSize(1100, 900)
		r.tt.Frame()
		r.dragOnto(t, "Write the release notes", "Port core to Go", true)
		r.after(330 * time.Millisecond)
	case "tick":
		r := newHandRig(t, false)
		r.tt.SetScale(2)
		r.tt.SetSize(1100, 820)
		r.tt.Frame()
		tick(r.tt, "Decide on the activation policy", "Write the release notes", "Draft the UI spec")
		r.tt.Frame()
	case "nothing":
		r := newHandRig(t, false)
		r.tt.SetScale(2)
		r.tt.SetSize(1100, 820)
		r.tt.Frame()
	case "click":
		r := newHandRig(t, false)
		r.tt.SetScale(2)
		r.tt.SetSize(1100, 820)
		r.tt.Frame()
		_ = r.tt.Click("Decide on the activation policy")
		r.tt.Frame()
	case "menu":
		TestScreensMenu(t)
	}
	cmpImg(t, "after "+os.Getenv("TMPX"), a, dragImg(t))
}
