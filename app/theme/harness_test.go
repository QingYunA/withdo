package theme

import (
	"image/color"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo/ui"
)

// harnessMark draws a mark in a frame of its own and returns the pixels.
func harnessMark(t *testing.T, id string, size float32, dark bool) *ui.Tester {
	t.Helper()
	return render(t, dark, func(c *ui.Context, tk *Tokens) { HarnessMark(c, id, size) })
}

// TestEveryHarnessMarkIsASquareOfItsGrid: every agent of the catalog has the two grids the
// Settings pane asks for, and each grid is a valid sprite of that side.
func TestEveryHarnessMarkIsASquareOfItsGrid(t *testing.T) {
	for _, a := range agentsetup.Agents {
		for _, c := range []struct {
			size float32
			side int
		}{{24, 24}, {18, 18}} {
			tt := harnessMark(t, a.ID, c.size, false)
			if !inked(tt) {
				t.Errorf("%s at %v drew nothing", a.ID, c.size)
			}
		}
	}
	for id, s := range harnessMarks24 {
		checkHarnessSprite(t, id, s, 24)
	}
	for id, s := range harnessMarks24Dark {
		checkHarnessSprite(t, id, s, 24)
	}
	for id, s := range harnessMarks18 {
		checkHarnessSprite(t, id, s, 18)
	}
	for id, s := range harnessMarks18Dark {
		checkHarnessSprite(t, id, s, 18)
	}
	for _, a := range agentsetup.Agents {
		if a.ID == "claude" {
			continue
		}
		for _, set := range []map[string]*Sprite{harnessMarks24, harnessMarks24Dark, harnessMarks18, harnessMarks18Dark} {
			if set[a.ID] == nil {
				t.Errorf("%s has no mark in one of the four sets", a.ID)
			}
		}
	}
}

func checkHarnessSprite(t *testing.T, id string, s *Sprite, side int) {
	t.Helper()
	if s.Height() != side || s.Width() != side {
		t.Errorf("%s is %d x %d, want %d square", id, s.Width(), s.Height(), side)
	}
	if len(s.Palette) == 0 || len(s.Palette) > 24 {
		t.Errorf("%s has %d colours, want 1 to 24", id, len(s.Palette))
	}
	for y, row := range s.Rows {
		if len([]rune(row)) != side {
			t.Errorf("%s row %d is %d wide", id, y, len([]rune(row)))
		}
		for _, r := range row {
			if r != '.' && s.Palette[r] == (ui.Color{}) {
				t.Errorf("%s row %d uses %q, which is not in the palette", id, y, r)
			}
		}
	}
}

// TestClaudeKeepsTheCrab: the one harness whose mark is the app's own mascot, at both sizes.
func TestClaudeKeepsTheCrab(t *testing.T) {
	for _, size := range []float32{24, 18} {
		for _, dark := range []bool{false, true} {
			got := harnessMark(t, "claude", size, dark)
			want := render(t, dark, func(c *ui.Context, tk *Tokens) { AvatarIcon(c, AgentAvatar("claude"), size) })
			if string(got.Image().Pix) != string(want.Image().Pix) {
				t.Errorf("Claude Code at %v (dark %v) is not the crab", size, dark)
			}
		}
	}
}

// harnessInk is the pixels a mark drew, leaving out the frame's background (which follows the
// appearance itself, so whole frames cannot be compared across it).
func harnessInk(tt *ui.Tester) []color.RGBA {
	img := tt.Image()
	b := img.Bounds()
	bg := img.RGBAAt(b.Min.X, b.Min.Y)
	var out []color.RGBA
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if px != bg {
				out = append(out, px)
			}
		}
	}
	return out
}

func sameInk(a, b []color.RGBA) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestMonochromeHarnessMarksReverseInDarkMode: a mark drawn in the text color turns light on a
// dark window, while the crab stays the crab.
func TestMonochromeHarnessMarksReverseInDarkMode(t *testing.T) {
	for _, id := range []string{"opencode", "zed", "windsurf", "cline", "deepseek", "kimi"} {
		light := harnessInk(harnessMark(t, id, 24, false))
		dark := harnessInk(harnessMark(t, id, 24, true))
		if sameInk(light, dark) {
			t.Errorf("%s is the same on a light and a dark window", id)
		}
	}
	crabLight := harnessInk(harnessMark(t, "claude", 24, false))
	crabDark := harnessInk(harnessMark(t, "claude", 24, true))
	if !sameInk(crabLight, crabDark) {
		t.Error("the crab changed with the appearance")
	}
}

// TestAHarnessMarkIsDrawnAtTheRequestedSize: the Settings rows slot the mark into an exact square.
func TestAHarnessMarkIsDrawnAtTheRequestedSize(t *testing.T) {
	for _, a := range agentsetup.Agents {
		for _, size := range []float32{24, 18} {
			var got ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				got = HarnessMark(c, a.ID, size).Bounds()
			}, 60, 60)
			tt.Frame()
			tt.Frame()
			if got.W != size || got.H != size {
				t.Errorf("%s at %v is %v x %v", a.ID, size, got.W, got.H)
			}
		}
	}
}

// TestAColouredHarnessMarkKeepsItsBrandColours: Gemini's gradient and Amp's red survive the
// palette; neither turns into the text color.
func TestAColouredHarnessMarkKeepsItsBrandColours(t *testing.T) {
	if s := harnessMarks24["amp"]; s == nil || len(s.Palette) < 2 {
		t.Error("Amp has no red gradient")
	}
	if s := harnessMarks24["gemini"]; s == nil || len(s.Palette) < 4 {
		t.Error("Gemini lost its gradient")
	}
}
