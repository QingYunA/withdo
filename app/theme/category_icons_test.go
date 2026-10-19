package theme

import (
	"testing"

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

func TestCategoryIconIsCachedByBitmap(t *testing.T) {
	ic, _ := CategoryIconNamed("heart")
	if CategoryIcon(ic.Bitmap) == nil || CategoryIcon(ic.Bitmap) != CategoryIcon(ic.Bitmap) {
		t.Error("the same bitmap must give the same *ui.SVG")
	}
	if CategoryIcon(ic.Bitmap.Invert()) == CategoryIcon(ic.Bitmap) {
		t.Error("different bitmaps shared an SVG")
	}
	if CategoryIcon(ic.Bitmap.Invert().Invert()) != CategoryIcon(ic.Bitmap) {
		t.Error("equal bitmaps built apart gave different SVGs")
	}
}

// The sidebar keeps drawing Work and Life from the vendored SVGs until the categories carry their
// own bitmaps; the two must paint the same pixels, or every screenshot with a category row changes.
func TestBitmapWorkAndLifeIconsPaintLikeTheVendoredSVGs(t *testing.T) {
	paint := func(s *ui.SVG, size float32) *ui.Tester {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Padding(8).Children(func() {
				ui.Icon(c, s).Size(size, size).TextColor(ui.Hex("#222222"))
			})
		}, 60, 60)
		tt.Frame()
		return tt
	}
	for _, name := range []string{"briefcase", "home"} {
		ic, _ := CategoryIconNamed(name)
		old := map[string]*ui.SVG{"briefcase": IconWork, "home": IconLife}[name]
		for _, size := range []float32{IconSmall, IconRow, IconBar} {
			a, b := paint(old, size).Image(), paint(CategoryIcon(ic.Bitmap), size).Image()
			if a.Bounds() != b.Bounds() || string(a.Pix) != string(b.Pix) {
				t.Errorf("%s at %v pt paints differently from the vendored SVG", name, size)
			}
		}
	}
}
