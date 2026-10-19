package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The screens of the user's categories (docs/design/mockups/v24.html, frames 1 to 6 and 8): the sidebar
// with two of them and their context menus, the New and Edit category sheets, the task menu's Category
// page and the toast after a delete. Render with
//
//	scripts/screens.sh TestScreensCategories

// categoriesApp is the seed with Reading and Health made, and a few tasks sorted into them.
func categoriesApp(t *testing.T) *App {
	t.Helper()
	app, _, _ := newTestApp(t)
	for _, c := range []struct{ name, color, icon string }{{"Reading", "orange", "library"}, {"Health", "red", "heart"}} {
		icon, ok := theme.CategoryIconNamed(c.icon)
		if !ok {
			t.Fatalf("no icon %q", c.icon)
		}
		cat, err := app.CreateCategory(data.NewCategory{Name: c.name, Icon: icon.Bitmap, Color: c.color})
		if err != nil {
			t.Fatal(err)
		}
		if c.name == "Reading" {
			for _, id := range []int{18, 19} {
				_ = app.Update(id, data.Patch{Category: data.IntPtr(cat.ID)})
			}
		}
	}
	return app
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
