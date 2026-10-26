package views

import "testing"

// The menu bar panel, in light and dark, English and Chinese (-zh), on a backdrop as it floats over
// the desktop. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensPopover
func TestScreensPopover(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			app, _, _ := newTestApp(t)
			p := NewPopover(app)
			height := 500
			p.Resize = func(h int) { height = h }
			tt := newTester(p.View, int(PopoverWidth), height)
			tt.SetScale(2)
			tt.SetDark(a.dark)
			still(tt)
			name := "popover-" + a.suffix + ".png"
			if zh {
				inChinese(tt)
				name = "popover-zh-" + a.suffix + ".png"
			}
			p.Show()
			for i := 0; i < 4; i++ {
				tt.Frame()
				tt.SetSize(int(PopoverWidth), height)
			}
			settle(tt)
			shot(t, name, tt.Image(), true, a.dark)
		}
	}
}
