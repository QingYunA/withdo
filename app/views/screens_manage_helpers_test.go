package views

import (
	"image"
	"image/draw"
	"testing"

	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The system draws a context menu itself, so the tester's image never holds one. For the design
// review the menu is drawn here in the app's own tokens, from the labels the tester reports, and
// set on the window where it opened. It is a picture of the menu, not the menu.

// menuSpec describes a menu to draw: its items, which of them open a submenu or are checked, and a
// submenu to show beside one of them.
type menuSpec struct {
	x, y    float32 // where it opens, in points
	items   []string
	subs    map[string]bool
	checked map[string]bool
	flyout  *flyoutSpec
}

// flyoutSpec is an open submenu, beside the item named parent.
type flyoutSpec struct {
	parent  string
	items   []string
	checked map[string]bool
}

const (
	menuRow   float32 = 24
	menuSep   float32 = 9
	menuWidth float32 = 240
)

func menuHeight(items []string) float32 {
	h := float32(10)
	for _, it := range items {
		if it == "-" {
			h += menuSep
		} else {
			h += menuRow
		}
	}
	return h
}

// menuImage renders one menu at 2x.
func menuImage(items []string, subs, checked map[string]bool, dark bool, lang string) *image.RGBA {
	w, h := int(menuWidth), int(menuHeight(items))
	tt := newTester(func(c *ui.Context) {
		t := theme.Resolve(c)
		ui.Column(c).Fill().Padding(5, 0).Background(t.Bg).Border(1, t.BgSelected.Mix(t.Text, 0.15)).Radius(8).Children(func() {
			for _, it := range items {
				if it == "-" {
					ui.Box(c).Height(menuSep).Shrink(0).Padding(4, 10).Children(func() {
						ui.Box(c).Height(1).Background(t.Border)
					})
					continue
				}
				row := ui.Row(c).Height(menuRow).Shrink(0).Margin(0, 5).Padding(0, 8).Gap(6).AlignItems(ui.Center).Radius(5)
				row.Children(func() {
					mark := ""
					if checked[it] {
						mark = "✓"
					}
					theme.Text(c, theme.Body, t.Text, mark).Width(12).Shrink(0)
					theme.Text(c, theme.Body, t.Text, it).Grow(1).MinWidth(0).Ellipsis("…")
					if subs[it] {
						theme.Text(c, theme.Body, t.TextSecondary, "›").Shrink(0)
					}
				})
			}
		})
	}, w, h)
	tt.SetScale(2)
	tt.SetDark(dark)
	if lang == "zh-Hans" {
		inChinese(tt)
	}
	still(tt)
	tt.Frame()
	return tt.Image()
}

// withMenu sets the menu (and its submenu) on a copy of the window image.
func withMenu(win *image.RGBA, m menuSpec, dark bool, lang string) *image.RGBA {
	out := image.NewRGBA(win.Bounds())
	draw.Draw(out, out.Bounds(), win, image.Point{}, draw.Src)
	at := func(img *image.RGBA, x, y float32) {
		r := image.Rect(int(x*2), int(y*2), int(x*2)+img.Bounds().Dx(), int(y*2)+img.Bounds().Dy())
		draw.Draw(out, r, img, image.Point{}, draw.Over)
	}
	at(menuImage(m.items, m.subs, m.checked, dark, lang), m.x, m.y)
	if f := m.flyout; f != nil {
		offset := float32(5)
		for _, it := range m.items {
			if it == f.parent {
				break
			}
			if it == "-" {
				offset += menuSep
			} else {
				offset += menuRow
			}
		}
		x := m.x + menuWidth - 4
		if x+menuWidth > 1100 { // no room at the right: it opens to the left
			x = m.x - menuWidth + 4
		}
		at(menuImage(f.items, nil, f.checked, dark, lang), x, m.y+offset-5)
	}
	return out
}

// manageShot renders the window at 2x in a language; setup may return a menu to draw over it.
func manageShot(t *testing.T, name string, app *App, dark bool, lang string, setup func(tt *ui.Tester, m *Main) *menuSpec) {
	t.Helper()
	m := NewMain(app)
	tt := newTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	if lang == "zh-Hans" {
		inChinese(tt)
	}
	var menu *menuSpec
	if setup != nil {
		menu = setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	img := tt.Image()
	if menu != nil {
		img = withMenu(img, *menu, dark, lang)
	}
	shot(t, name, img, false, dark)
}

// languages are the two the screens come in, with the suffix their files carry.
var languages = []struct{ tag, suffix string }{{"en", ""}, {"zh-Hans", "-zh"}}
