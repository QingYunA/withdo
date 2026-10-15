package theme

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/egoist/mygo/ui"
)

func TestEveryVendoredLogoParsesAndIsStripped(t *testing.T) {
	files, err := logoFiles.ReadDir("logos")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".svg") {
			continue
		}
		n++
		b, err := logoFiles.ReadFile("logos/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ui.ParseSVG(b); err != nil {
			t.Errorf("%s does not parse: %v", f.Name(), err)
		}
		if s := string(b); strings.Contains(s, "<title") || strings.Contains(s, "1em") {
			t.Errorf("%s still has its title or its 1em size", f.Name())
		}
	}
	if n != len(logoFile) {
		t.Errorf("%d svg files, %d logos wired", n, len(logoFile))
	}
	for _, name := range []string{"LICENSE", "README.md"} {
		if _, err := logoFiles.ReadFile("logos/" + name); err == nil {
			t.Errorf("%s is embedded; only the svg files should be", name)
		}
	}
}

func TestEveryPresetHasALogoThatDraws(t *testing.T) {
	for _, p := range providers.Presets {
		for _, dark := range []bool{false, true} {
			tt := render(t, dark, func(c *ui.Context, tk *Tokens) {
				ProviderLogo(c, p.ID, LogoTile).TextColor(tk.Text)
			})
			if !inked(tt) {
				t.Errorf("%s (dark %v) drew nothing", p.ID, dark)
			}
		}
	}
	for id := range logoFile {
		if _, ok := providers.PresetByID(id); !ok {
			t.Errorf("a logo for %q, which is no preset", id)
		}
	}
}

func TestAMonoLogoFollowsTheTextColor(t *testing.T) {
	ink := func(dark bool) ui.Color {
		tt := render(t, dark, func(c *ui.Context, tk *Tokens) { ProviderLogo(c, "openai", 64).TextColor(tk.Text) })
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
