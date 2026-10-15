package theme

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// inked reports whether a rendered frame holds anything but the background.
func inked(tt *ui.Tester) bool {
	img := tt.Image()
	b := img.Bounds()
	bg := img.RGBAAt(b.Min.X, b.Min.Y)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y) != bg {
				return true
			}
		}
	}
	return false
}

func render(t *testing.T, dark bool, build func(c *ui.Context, tk *Tokens)) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		tk := Resolve(c)
		ui.Column(c).Padding(8).Gap(8).Children(func() { build(c, tk) })
	}, 120, 80)
	tt.SetDark(dark)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	return tt
}

func TestEveryStatusGlyphDraws(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, s := range data.Statuses {
			for _, fill := range []float32{0, 0.5, 1} {
				tt := render(t, dark, func(c *ui.Context, tk *Tokens) { StatusGlyph(c, tk, s, fill) })
				if !inked(tt) {
					t.Errorf("%s (dark %v, fill %v) drew nothing", s, dark, fill)
				}
			}
		}
	}
}

func TestPriorityBadgeClockAndDotDraw(t *testing.T) {
	for _, pr := range []data.Priority{data.P0, data.P1} {
		tt := render(t, false, func(c *ui.Context, tk *Tokens) { PriorityBadge(c, tk, pr) })
		if !inked(tt) || !tt.HasText(string(pr)) {
			t.Errorf("%s badge drew nothing", pr)
		}
	}
	for pr, want := range map[data.Priority]bool{data.P0: true, data.P1: true, data.P2: false, data.P3: false} {
		if ShowsPriority(pr) != want {
			t.Errorf("ShowsPriority(%s) = %v, want %v", pr, !want, want)
		}
	}
	if !inked(render(t, false, func(c *ui.Context, tk *Tokens) { ClockGlyph(c, tk.Text) })) {
		t.Error("clock drew nothing")
	}
	if !inked(render(t, false, func(c *ui.Context, tk *Tokens) { Dot(c, tk.Blocked(), 6) })) {
		t.Error("dot drew nothing")
	}
}

func TestTextStylesRenderAndSectionsAreSentenceCase(t *testing.T) {
	tt := render(t, false, func(c *ui.Context, tk *Tokens) {
		Text(c, Section, tk.Text, "agents")
		Paragraph(c, Body, tk.Text, "a long title that wraps", 2)
		Text(c, Mono, tk.Text, "feat/x")
	})
	if !tt.HasText("agents") || tt.HasText("AGENTS") || !tt.HasText("feat/x") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestResolveFollowsTheAppearance(t *testing.T) {
	var got [2]bool
	for i, dark := range []bool{false, true} {
		i := i
		tt := ui.NewTester(func(c *ui.Context) { got[i] = Resolve(c).Dark }, 40, 40)
		tt.SetDark(dark)
		tt.Frame()
	}
	if got[0] || !got[1] {
		t.Errorf("dark flags %v", got)
	}
}

func TestUnknownStatusFallsBackToTertiary(t *testing.T) {
	tk := For(false, ui.Transparent)
	if tk.Status("nope") != tk.TextTertiary {
		t.Error("unknown status color")
	}
}
