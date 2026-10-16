package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// pixelAt renders a 40 × 30 pt drawing at (10, 10) of a 60 × 50 window, one device pixel a point,
// and reports for each point asked whether it holds anything but the window's background.
func pixelAt(t *testing.T, dark bool, draw func(p *ui.Painter, r ui.Rect, tk *Tokens), pts [][2]int) []bool {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		tk := Resolve(c)
		ui.Box(c).FillWidth().FillHeight().Background(ui.Hex("#808080")).Children(func() {
			ui.Box(c).Absolute().Left(10).Top(10).Size(40, 30).Draw(func(p *ui.Painter, r ui.Rect) { draw(p, r, tk) })
		})
	}, 60, 50)
	tt.SetScale(1)
	tt.SetDark(dark)
	tt.Frame()
	img := tt.Image()
	bg := img.RGBAAt(0, 0)
	out := make([]bool, len(pts))
	for i, pt := range pts {
		out[i] = img.RGBAAt(pt[0], pt[1]) != bg
	}
	return out
}

func TestPixelPanelStepsItsCornersAndCastsAHardShadow(t *testing.T) {
	// The panel spans (10, 10) to (50, 40); its shadow (14, 14) to (54, 44).
	cases := []struct {
		name  string
		pt    [2]int
		inked bool
	}{
		{"outside the first step of the top left corner", [2]int{11, 11}, false},
		{"outside the second step of the top left corner", [2]int{12, 10}, false},
		{"the edge inside the second step", [2]int{13, 13}, true},
		{"the left edge", [2]int{10, 20}, true},
		{"the fill", [2]int{30, 25}, true},
		{"the shadow below the bottom right corner", [2]int{48, 42}, true},
		{"no shadow above its own stepped corner", [2]int{52, 12}, false},
		{"no shadow left of the panel", [2]int{8, 30}, false},
	}
	for _, dark := range []bool{false, true} {
		pts := make([][2]int, len(cases))
		for i, tc := range cases {
			pts[i] = tc.pt
		}
		got := pixelAt(t, dark, func(p *ui.Painter, r ui.Rect, tk *Tokens) { DrawPixelPanel(p, r, tk) }, pts)
		for i, tc := range cases {
			if got[i] != tc.inked {
				t.Errorf("dark %v, %s %v: inked %v, want %v", dark, tc.name, tc.pt, got[i], tc.inked)
			}
		}
	}
}

func TestPixelWellStepsItsCornersOnce(t *testing.T) {
	got := pixelAt(t, false, func(p *ui.Painter, r ui.Rect, tk *Tokens) { DrawPixelWell(p, r, tk.BgSubtle) },
		[][2]int{{11, 11}, {12, 12}, {49, 39}, {47, 37}, {30, 25}})
	want := []bool{false, true, false, true, true}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("point %d inked %v, want %v", i, got[i], want[i])
		}
	}
}

func TestPixelEdgeStandsOutFromTheBackground(t *testing.T) {
	for _, tk := range []*Tokens{For(false, ui.Hex("#5B5BD6")), For(true, ui.Hex("#5B5BD6"))} {
		if tk.PixelEdge() == tk.Bg || tk.PixelShadow().A == 0 {
			t.Errorf("dark %v: edge %v on bg %v, shadow %v", tk.Dark, tk.PixelEdge(), tk.Bg, tk.PixelShadow())
		}
	}
}
