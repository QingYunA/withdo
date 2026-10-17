package theme

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/egoist/mygo/ui"
)

func TestEveryMarkIsAnEighteenSquareOfItsPalette(t *testing.T) {
	if len(providerMarks) == 0 {
		t.Fatal("no marks")
	}
	for id, s := range providerMarks {
		if s.Height() != 18 {
			t.Errorf("%s is %d rows, want 18", id, s.Height())
		}
		for y, row := range s.Rows {
			if w := len([]rune(row)); w != 18 {
				t.Errorf("%s row %d is %d wide, want 18", id, y, w)
			}
			for _, r := range row {
				if _, ok := s.Palette[r]; !ok && r != '.' {
					t.Errorf("%s row %d: %q is not in the palette", id, y, r)
				}
			}
		}
		if _, ok := providers.PresetByID(id); !ok {
			t.Errorf("a mark for %q, which is no preset", id)
		}
	}
}

func TestAOneColorMarkOnlyUsesK(t *testing.T) {
	for id := range oneColorMarks {
		s, ok := providerMarks[id]
		if !ok {
			t.Errorf("%s is one-color but has no mark", id)
			continue
		}
		if len(s.Palette) != 1 || s.Palette['k'] == (ui.Color{}) {
			t.Errorf("%s palette %v, want only k", id, s.Palette)
		}
		for _, row := range s.Rows {
			for _, r := range row {
				if r != 'k' && r != '.' {
					t.Fatalf("%s uses %q", id, r)
				}
			}
		}
	}
}

func TestEveryPresetHasALogo(t *testing.T) {
	for _, p := range providers.Presets {
		_, mark := providerMarks[p.ID]
		if !mark && p.ID != "anthropic" && p.ID != "magpie" && p.ID != "custom" {
			t.Errorf("%s has no logo", p.ID)
		}
	}
}

func TestAnthropicIsTheCrab(t *testing.T) {
	if _, ok := providerMarks["anthropic"]; ok {
		t.Error("anthropic has a sampled mark; it is the Claude Code crab")
	}
	if AgentAvatar("claude").Sprite() != spriteClaude {
		t.Fatal("the sidebar's Claude is not spriteClaude")
	}
	for _, dark := range []bool{false, true} {
		got := render(t, dark, func(c *ui.Context, tk *Tokens) { ProviderLogo(c, "anthropic", LogoTile, tk.Text) })
		want := render(t, dark, func(c *ui.Context, tk *Tokens) { AvatarIcon(c, AgentAvatar("claude"), LogoTile) })
		if string(got.Image().Pix) != string(want.Image().Pix) {
			t.Errorf("anthropic (dark %v) is not drawn as the crab", dark)
		}
	}
}

func TestEveryPresetHasALogoThatDraws(t *testing.T) {
	for _, p := range providers.Presets {
		for _, dark := range []bool{false, true} {
			tt := render(t, dark, func(c *ui.Context, tk *Tokens) { ProviderLogo(c, p.ID, LogoTile, tk.Text) })
			if !inked(tt) {
				t.Errorf("%s (dark %v) drew nothing", p.ID, dark)
			}
		}
	}
}

func TestALogoIsDrawnAtTheRequestedSize(t *testing.T) {
	for _, id := range []string{"deepseek", "openai", "anthropic", "magpie", "custom", "nobody"} {
		for _, size := range []float32{LogoRow, LogoTile} {
			var got ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				got = ProviderLogo(c, id, size, Resolve(c).Text).Bounds()
			}, 120, 80)
			tt.Frame()
			tt.Frame()
			if got.W != size || got.H != size {
				t.Errorf("%s at %v is %v x %v", id, size, got.W, got.H)
			}
		}
	}
}

func TestAMonoLogoFollowsTheInk(t *testing.T) {
	ink := func(dark bool) ui.Color {
		tt := render(t, dark, func(c *ui.Context, tk *Tokens) { ProviderLogo(c, "openai", 72, tk.Text) })
		img, bg := tt.Image(), tt.Image().RGBAAt(0, 0)
		var best ui.Color
		score := -1
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				px := img.RGBAAt(x, y)
				d := absDiff(px.R, bg.R) + absDiff(px.G, bg.G) + absDiff(px.B, bg.B)
				if d > score {
					score, best = d, ui.RGB(px.R, px.G, px.B)
				}
			}
		}
		return best
	}
	light, dark := ink(false), ink(true)
	if light.R > 80 || dark.R < 175 {
		t.Errorf("the OpenAI mark is %v in light and %v in dark; want dark ink, then light ink", light, dark)
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
