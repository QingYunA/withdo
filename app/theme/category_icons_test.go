package theme

import (
	"image/color"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

func TestEveryCuratedIconExistsAndDraws(t *testing.T) {
	if len(CategoryIconGroups) != 8 {
		t.Fatalf("%d groups, want 8", len(CategoryIconGroups))
	}
	total := 0
	for _, g := range CategoryIconGroups {
		if len(g.Icons) == 0 {
			t.Errorf("group %s is empty", g.Name)
		}
		for _, ic := range g.Icons {
			total++
			got, ok := CategoryIconNamed(ic.Name)
			if !ok || got.Bitmap.Empty() {
				t.Errorf("%s/%s: missing or empty", g.Name, ic.Name)
			}
			if ic.Keywords == "" {
				t.Errorf("%s has no Chinese keywords", ic.Name)
			}
		}
	}
	if total < 36 || total > 48 {
		t.Errorf("%d curated icons, want about 40", total)
	}
}

func TestWorkAndLifeBitmapsAreTheMigrationsDefaults(t *testing.T) {
	for name, hex := range map[string]string{
		"briefcase": "0000600903fc4024024024024024023fc000",
		"home":      "0000600901082044024624924924923fc000",
	} {
		ic, ok := CategoryIconNamed(name)
		if !ok || ic.Bitmap.Hex() != hex {
			t.Errorf("%s = %v, want %s", name, ic.Bitmap.Hex(), hex)
		}
	}
}

func TestLibraryIsSortedAndNamesAreUnique(t *testing.T) {
	if len(CategoryIcons) < 300 {
		t.Fatalf("%d icons", len(CategoryIcons))
	}
	for i := 1; i < len(CategoryIcons); i++ {
		if CategoryIcons[i-1].Name >= CategoryIcons[i].Name {
			t.Fatalf("%s before %s", CategoryIcons[i-1].Name, CategoryIcons[i].Name)
		}
	}
}

func TestSearchReadsEnglishNamesAndChineseKeywords(t *testing.T) {
	names := func(q string) map[string]bool {
		out := map[string]bool{}
		for _, ic := range SearchCategoryIcons(q) {
			out[ic.Name] = true
		}
		return out
	}
	if !names("coffee")["coffee"] || !names("COFFEE")["coffee"] {
		t.Error("coffee is not found by its name")
	}
	if !names("咖啡")["coffee"] {
		t.Error("coffee is not found by its Chinese keyword")
	}
	if !names("arrow down")["arrow-down"] || !names("arrow-down")["arrow-down"] {
		t.Error(`"-" and a space must read the same`)
	}
	if len(SearchCategoryIcons("zzzz nothing")) != 0 {
		t.Error("nonsense matched")
	}
	if len(SearchCategoryIcons("  ")) != 0 {
		t.Error("a blank search lists nothing; the picker shows the curated groups then")
	}
}

func TestCategoryIconIsCachedByPicture(t *testing.T) {
	ic, _ := CategoryIconNamed("heart")
	p := ic.Bitmap.Picture()
	if CategoryIcon(p) == nil || CategoryIcon(p) != CategoryIcon(p) {
		t.Error("the same picture must give the same *ui.SVG")
	}
	if CategoryIcon(p.Invert()) == CategoryIcon(p) {
		t.Error("different pictures shared an SVG")
	}
	if CategoryIcon(p.Invert().Invert()) != CategoryIcon(p) {
		t.Error("equal pictures built apart gave different SVGs")
	}
}

func TestThePaletteIsTheProjectColorsThenWhiteGrayBlackBrown(t *testing.T) {
	for i, c := range ProjectColors {
		if PaletteColors[i] != c {
			t.Errorf("letter %c is not project color %d", 'a'+i, i)
		}
	}
	for i, hex := range []string{"#FFFFFF", "#8E8E93", "#1C1C1E", "#8A5A3B"} {
		if got := PaletteColors[len(ProjectColors)+i]; got != ui.Hex(hex) {
			t.Errorf("letter %c = %v, want %s", 'i'+i, got, hex)
		}
	}
	if len(PaletteColors) != pixmap.Colors || PaletteColor('d') != ProjectColors[3] || PaletteColor('l') != PaletteColors[11] {
		t.Error("PaletteColor does not read the letters")
	}
}

// ui.Image keeps a palette pixel's own color while the TextColor tints the Own pixels: that is how
// the sidebar draws every category icon.
func TestAPicturesPalettePixelsSurviveTheTintOfTheOwnOnes(t *testing.T) {
	pic := pixmap.Picture{}
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			c := pixmap.Own
			if x >= 6 {
				c = 'd' // red
			}
			pic = pic.Set(x, y, c)
		}
	}
	tint := ui.Hex("#3B82F6")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Padding(10).Children(func() { ui.Image(c, CategoryIcon(pic)).Size(48, 48).TextColor(tint) })
	}, 70, 70)
	tt.Frame()
	img := tt.Image()
	scale := img.Bounds().Dx() / 70
	left, right := img.RGBAAt(20*scale, 34*scale), img.RGBAAt(48*scale, 34*scale)
	near := func(p color.RGBA, want ui.Color) bool {
		d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
		return d(p.R, want.R)+d(p.G, want.G)+d(p.B, want.B) < 24
	}
	if !near(left, tint) || !near(right, PaletteColor('d')) {
		t.Errorf("own pixel %v (want the tint), palette pixel %v (want red)", left, right)
	}
}

func TestCategoryColorsAreTheProjectColorsInOrder(t *testing.T) {
	if len(domain.CategoryColors) != len(ProjectColors) {
		t.Fatalf("%d keys for %d colors", len(domain.CategoryColors), len(ProjectColors))
	}
	for i, key := range domain.CategoryColors {
		if CategoryColor(key) != ProjectColors[i] {
			t.Errorf("%s is not project color %d", key, i)
		}
	}
	// A new category gets the color its name hashes to, which is the one a project of that name has.
	for _, name := range []string{"Reading", "Garden", "adhd-todo", "读书", "Work"} {
		if got, want := CategoryColor(domain.DefaultCategoryColor(name)), ProjectColor(name); got != want {
			t.Errorf("%s: category color %v, project color %v", name, got, want)
		}
	}
	if CategoryColor("mauve") != ProjectColors[0] {
		t.Error("a key from a newer build falls back to the first color")
	}
}
