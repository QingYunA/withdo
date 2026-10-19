package views

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The screens of the user's categories (docs/design/mockups/v24.html, frames 1 to 8): the sidebar
// with three of them and their context menus, the New and Edit category sheets, the icon editor and
// the import, the task menu's Category page and the toast after a delete. Render with
//
//	scripts/screens.sh TestScreensCategories

// coffeeCup is a multi-color icon: teal steam over a cup in the category's own color.
var coffeeCup = strings.Join([]string{
	"............",
	"...g..g.....",
	"..g..g......",
	"............",
	".########...",
	".#......###.",
	".#......#.#.",
	".#......#.#.",
	".#......###.",
	"..#....#....",
	"...####.....",
	"............",
}, "")

// categoriesApp is the seed with Reading, Health and Coffee (a drawn icon of two colors) made, and a
// few tasks sorted into them.
func categoriesApp(t *testing.T) *App {
	t.Helper()
	app, _, _ := newTestApp(t)
	for _, c := range []struct{ name, color, icon string }{{"Reading", "orange", "library"}, {"Health", "red", "heart"}} {
		icon, ok := theme.CategoryIconNamed(c.icon)
		if !ok {
			t.Fatalf("no icon %q", c.icon)
		}
		cat, err := app.CreateCategory(data.NewCategory{Name: c.name, Icon: icon.Bitmap.Picture(), Color: c.color})
		if err != nil {
			t.Fatal(err)
		}
		if c.name == "Reading" {
			for _, id := range []int{18, 19} {
				_ = app.Update(id, data.Patch{Category: data.IntPtr(cat.ID)})
			}
		}
	}
	cup, err := pixmap.ParseGrid(coffeeCup)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateCategory(data.NewCategory{Name: "Coffee", Icon: cup, Color: "orange"}); err != nil {
		t.Fatal(err)
	}
	return app
}

// coffeePhoto writes a photo-like picture of a dark mug on beige with a soft shadow and noise, as
// the PNG the import screens read, and returns its path.
func coffeePhoto(t *testing.T) string {
	t.Helper()
	const side = 144
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	seed := uint32(7)
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			k := float64(x+y) / (2 * side)
			c := [3]float64{230 - 29*k, 220 - 35*k, 203 - 45*k}
			if d := math.Hypot(float64(x-76)/44, float64(y-118)/9); d < 1 {
				for i := range c {
					c[i] *= 1 - 0.3*(1-d)
				}
			}
			body := x >= 36 && x < 100 && y >= 46 && y < 116 && (y < 102 || math.Hypot(float64(x)-math.Min(math.Max(float64(x), 50), 86), float64(y-102)) < 14)
			handle := x >= 98 && math.Abs(math.Hypot(float64(x-103), float64(y-78))-15) < 5
			switch {
			case body || handle:
				c = [3]float64{59, 52, 49}
			}
			if math.Hypot(float64(x-68)/30, float64(y-47)/7) < 1 {
				c = [3]float64{217, 207, 189}
				if math.Hypot(float64(x-68)/24, float64(y-48)/4) < 1 {
					c = [3]float64{74, 63, 58}
				}
			}
			seed = seed*1664525 + 1013904223
			n := float64(int(seed>>24)%19 - 9)
			img.SetNRGBA(x, y, color.NRGBA{uint8(c[0] + n), uint8(c[1] + n), uint8(c[2] + n), 255})
		}
	}
	path := filepath.Join(t.TempDir(), "coffee.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// importCoffee opens the New category sheet on an import of the coffee photo.
func importCoffee(t *testing.T, tt *ui.Tester, m *Main, name string, outline bool) {
	t.Helper()
	m.openNewCategory(0)
	tt.Frame()
	tt.Type(name)
	s := &m.manage.category
	s.color, s.picked = "orange", true
	imp, err := data.ReadIconFile(coffeePhoto(t))
	if err != nil {
		t.Fatal(err)
	}
	m.iconRead(imp, nil)
	if outline {
		s.imp.outline = true
		s.icon = s.imp.icon()
	}
}

// categoryMenuAt right-clicks a sidebar row and returns the menu it opens, for withMenu.
func categoryMenuAt(tt *ui.Tester, name string) *menuSpec {
	r, _ := tt.Find(name)
	x, y := r.X+24, r.Y+r.H/2
	tt.RightClick(name)
	return &menuSpec{x: x, y: y, items: tt.Menu()}
}

func TestScreensCategories(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			suffix := l.suffix + "-" + a.suffix + ".png"
			app := categoriesApp(t)
			typed := func(en, zh string) string {
				if l.tag == "zh-Hans" {
					return zh
				}
				return en
			}
			manageShot(t, "category-sidebar"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenMine, cat: categoryNamedID(app, "Reading")})
				return nil
			})
			manageShot(t, "category-sidebar-menu"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenMine, cat: categoryNamedID(app, "Reading")})
				tt.Frame()
				return categoryMenuAt(tt, "Reading")
			})
			manageShot(t, "category-sheet-new"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.openNewCategory(0)
				tt.Frame()
				tt.Type(typed("Garden", "花园"))
				return nil
			})
			manageShot(t, "category-sheet-search"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.openNewCategory(0)
				tt.Frame()
				tt.Type(typed("Coffee", "咖啡"))
				_ = tt.Click(m.say.CategorySearchIcons())
				tt.Type("cof")
				_ = tt.Click("coffee")
				return nil
			})
			manageShot(t, "category-editor"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.openNewCategory(0)
				tt.Frame()
				tt.Type(typed("Coffee", "咖啡"))
				cup, _ := pixmap.ParseGrid(coffeeCup)
				s := &m.manage.category
				s.icon, s.color, s.picked = cup, "orange", true
				s.startDraw()
				s.draw.pen = 'g'
				return nil
			})
			manageShot(t, "category-import"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				importCoffee(t, tt, m, typed("Coffee", "咖啡"), false)
				return nil
			})
			manageShot(t, "category-import-outline"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				importCoffee(t, tt, m, typed("Coffee", "咖啡"), true)
				return nil
			})
			manageShot(t, "category-sheet-edit-work"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				work, _ := app.Category(app.workID())
				m.openEditCategory(work)
				return nil
			})
			manageShot(t, "task-menu-category"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				menuAt(tt, "Write the release notes", 120)
				_ = tt.Click(m.say.PropCategory())
				return nil
			})
			manageShot(t, "category-deleted"+suffix, app, a.dark, l.tag, func(tt *ui.Tester, m *Main) *menuSpec {
				tt.Frame()
				_ = tt.RightClick("Reading")
				_ = tt.ChooseMenuItem(m.say.CategoryDelete())
				return nil
			})
		}
	}
}

// categoryNamedID is the id of the category called name, 0 when there is none.
func categoryNamedID(app *App, name string) int {
	for _, c := range app.snap.categories {
		if c.Name == name {
			return c.ID
		}
	}
	return 0
}
