package views

import (
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The category sheet makes and edits categories; the task menu and Settings > General open it (the
// Settings tests are in settings_categories_test.go). See docs/design/mockups/v24.html.

func categoryNames(app *App) []string {
	var out []string
	for _, c := range app.snap.categories {
		out = append(out, c.Name)
	}
	return out
}

func categoryNamed(t *testing.T, app *App, name string) data.Category {
	t.Helper()
	for _, c := range app.snap.categories {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no category %q in %v", name, categoryNames(app))
	return data.Category{}
}

// openNewCategory is right-click a task, Category, New category…: the sheet is the same one the
// Settings window opens, and the task menu reaches it from the main window.
func openNewCategory(t *testing.T, tt *ui.Tester) {
	t.Helper()
	tt.Frame()
	rightClick(t, tt, "Write the release notes")
	click(t, tt, "Category")
	click(t, tt, "New category…")
	tt.Frame()
}

func TestNewCategoryFromTheTaskMenuSearchPickAndCreate(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	openNewCategory(t, tt)
	for _, want := range []string{"New category", "Name", "Icon", "Color", "Search icons", "Work", "Study", "Cancel", "Create"} {
		if !tt.HasText(want) {
			t.Errorf("the sheet lacks %q in %q", want, tt.Texts())
		}
	}
	tt.Type("Coffee") // the name has the focus
	click(t, tt, "Search icons")
	tt.Type("coffee")
	if tt.HasText("Study") {
		t.Errorf("a search still shows the groups: %q", tt.Texts())
	}
	click(t, tt, "coffee") // the cell's name
	click(t, tt, "Create")
	cat := categoryNamed(t, app, "Coffee")
	if want, _ := theme.CategoryIconNamed("coffee"); cat.Icon != want.Bitmap.Picture() {
		t.Errorf("icon %v, want the coffee cup", cat.Icon)
	}
	if want := data.DefaultCategoryColor("Coffee"); cat.Color != want {
		t.Errorf("color %q, want the one its name hashes to, %q", cat.Color, want)
	}
	if tt.HasText("Search icons") {
		t.Error("the sheet stays open after Create")
	}
	if m.nav != (nav{}) {
		t.Errorf("nav %+v: a new category opens no page", m.nav)
	}
}

func TestSearchIconsSaysWhenNothingMatches(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	openNewCategory(t, tt)
	click(t, tt, "Search icons")
	tt.Type("cofx")
	if !tt.HasText("No icon matches") {
		t.Errorf("no word for an empty search: %q", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("co")
	if tt.HasText("No icon matches") || !tt.HasText("coffee") {
		t.Errorf("co should list the icons with co in their name: %q", tt.Texts())
	}
}

func TestEnterInTheNameCreatesAndEscCancels(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	openNewCategory(t, tt)
	tt.Key(0, ui.KeyEnter) // nothing typed: nothing is made
	if len(app.snap.categories) != 2 || !tt.HasText("Create") {
		t.Fatalf("a category without a name: %v", categoryNames(app))
	}
	tt.Type("Reading")
	tt.Key(0, ui.KeyEnter)
	if !slices.Contains(categoryNames(app), "Reading") || tt.HasText("Create") {
		t.Errorf("categories %v, sheet open %v", categoryNames(app), tt.HasText("Create"))
	}
	openNewCategory(t, tt)
	tt.Type("Nope")
	tt.Key(0, ui.KeyEscape)
	if slices.Contains(categoryNames(app), "Nope") || tt.HasText("Create") {
		t.Error("Esc did not cancel")
	}
}

func TestANewCategoryKeepsAPickedColorWhileTheNameChanges(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	openNewCategory(t, tt)
	tt.Type("A")
	click(t, tt, "Color: Red")
	name, _ := tt.Find("Name")
	tt.ClickAt(name.X+40, name.Y+name.H+18) // back into the name field
	tt.Type("bc")
	click(t, tt, "Create")
	if got := categoryNamed(t, app, "Abc").Color; got != "red" {
		t.Errorf("color %q, want the picked red", got)
	}
}

func TestANewCategoryFollowsTheHashOfItsNameUntilAColorIsPicked(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	openNewCategory(t, tt)
	tt.Type("Garden")
	tt.Type(" walks")
	click(t, tt, "Create")
	if got, want := categoryNamed(t, app, "Garden walks").Color, data.DefaultCategoryColor("Garden walks"); got != want {
		t.Errorf("color %q, want %q", got, want)
	}
}

func TestTheSheetSaysUnderTheNameWhatIsWrongAndStaysOpen(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	for _, c := range []struct{ name, line string }{
		{"work", "A category with that name already exists."},
		{"生活", "A category with that name already exists."}, // Work and Life answer to their Chinese names too
		{strings.Repeat("x", 33), "A name can have at most 32 characters."},
	} {
		openNewCategory(t, tt)
		tt.Type(c.name)
		tt.Key(0, ui.KeyEnter)
		if !tt.HasText(c.line) || !tt.HasText("Create") {
			t.Errorf("%q: no %q, or the sheet closed: %q", c.name, c.line, tt.Texts())
		}
		tt.Key(0, ui.KeyEscape)
	}
	if len(app.snap.categories) != 2 {
		t.Errorf("categories %v", categoryNames(app))
	}
}

func TestTheSheetSaysTheListIsFull(t *testing.T) {
	app, _, _ := newTestApp(t)
	for i := len(app.snap.categories); i < data.MaxCategories; i++ {
		addCategory(t, app, "Category "+itoa(i), "blue")
	}
	tt, _, _ := newTesterFor(app, 1100)
	openNewCategory(t, tt)
	tt.Type("One too many")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("There can be at most 24 categories. Delete one first.") {
		t.Errorf("no word about the full list: %q", tt.Texts())
	}
}

func TestNewCategoryFromTheTaskMenuIsGivenToTheTask(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	if !tt.HasText("New category…") {
		t.Fatalf("no New category… on the Category page: %q", tt.Texts())
	}
	click(t, tt, "New category…")
	tt.Type("Errands")
	click(t, tt, "Create")
	errands := categoryNamed(t, app, "Errands")
	if got, _ := app.Task(3); got.CategoryID != errands.ID {
		t.Errorf("task category %d, want the new %d", got.CategoryID, errands.ID)
	}
	if m.nav != (nav{screen: screenProject, projectID: 1}) {
		t.Errorf("nav %+v: a category made from the task menu opens no page", m.nav)
	}
}

func TestTheSheetSpeaksChinese(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	tt.Frame()
	rightClick(t, tt, "Write the release notes")
	click(t, tt, "分类")
	click(t, tt, "新建类别…")
	tt.Frame()
	for _, want := range []string{"新建类别", "名称", "图标", "颜色", "搜索图标", "取消", "创建", "学习", "理财"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	click(t, tt, "搜索图标")
	tt.Type("cofx")
	if !tt.HasText("没有匹配的图标") {
		t.Errorf("no word for an empty search: %q", tt.Texts())
	}
}
