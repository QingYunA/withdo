package views

import (
	"slices"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The category sheet makes and edits categories; the sidebar's context menus open it, move and
// delete them. See docs/design/mockups/v24.html.

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

// openNewCategory is right-click Mine, New category….
func openNewCategory(t *testing.T, tt *ui.Tester) {
	t.Helper()
	tt.Frame()
	if err := tt.RightClick("Mine"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("New category…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

func TestMineMenuOffersNewCategory(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("Mine")
	if got, want := tt.Menu(), []string{"New category…"}; !slices.Equal(got, want) {
		t.Errorf("menu %q, want %q", got, want)
	}
}

func TestNewCategoryFromTheSidebarSearchPickAndCreate(t *testing.T) {
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
	if want, _ := theme.CategoryIconNamed("coffee"); cat.Icon != want.Bitmap {
		t.Errorf("icon %v, want the coffee cup", cat.Icon)
	}
	if want := data.DefaultCategoryColor("Coffee"); cat.Color != want {
		t.Errorf("color %q, want the one its name hashes to, %q", cat.Color, want)
	}
	if tt.HasText("Search icons") {
		t.Error("the sheet stays open after Create")
	}
	if m.nav != (nav{screen: screenMine, cat: cat.ID}) {
		t.Errorf("nav %+v: the new category's page opens", m.nav)
	}
	row, ok := tt.Find("Coffee")
	if !ok {
		t.Fatalf("no sidebar row: %q", tt.Texts())
	}
	slot := ui.Rect{X: row.X - sidebarRowGap - theme.AvatarSmall, Y: row.Y, W: theme.AvatarSmall, H: row.H}
	if !paintedIn(tt.Image(), 1100, slot, theme.CategoryColor(cat.Color), 30) {
		t.Error("the row's icon is not drawn in the category's color")
	}
	if !above(t, tt, "Life", "Coffee") || !above(t, tt, "Coffee", "Unsorted") {
		t.Errorf("a new category goes after Life, before Unsorted: %q", tt.Texts())
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

func TestEditingWorkShowsItsNameReadOnlyAndSavesTheIcon(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("Work")
	if err := tt.ChooseMenuItem("Edit category…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"Edit category", "Work and Life keep their names.", "Save"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	tt.Type("Office") // the name has no input to take it
	if tt.HasText("Office") {
		t.Errorf("the built-in name took text: %q", tt.Texts())
	}
	laptop, _ := theme.CategoryIconNamed("laptop")
	click(t, tt, "laptop")
	click(t, tt, "Color: Red")
	click(t, tt, "Save")
	work := categoryNamed(t, app, "Work")
	if work.Icon != laptop.Bitmap || work.Color != "red" {
		t.Errorf("Work after Save: icon %v color %q", work.Icon, work.Color)
	}
	if tt.HasText("Edit category") {
		t.Error("the sheet stays open after Save")
	}
}

func TestEditingACategoryOfTheUserRenamesIt(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "Reading", "orange")
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	_ = tt.RightClick("Reading")
	if err := tt.ChooseMenuItem("Edit category…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Work and Life keep their names.") {
		t.Error("the note of the built-ins is on a category of the user's")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Books")
	tt.Key(0, ui.KeyEnter)
	got, _ := app.Category(reading.ID)
	if got.Name != "Books" || got.Color != "orange" {
		t.Errorf("category %+v: renamed, with its color kept", got)
	}
}

func TestCategoryMenuHasMoveUpAndDownExceptAtTheEnds(t *testing.T) {
	app, _, _ := newTestApp(t)
	addCategory(t, app, "Reading", "orange")
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	for label, want := range map[string][]string{
		"Work":    {"New category…", "Edit category…", "Move down"},
		"Life":    {"New category…", "Edit category…", "Move up", "Move down"},
		"Reading": {"New category…", "Edit category…", "Move up", "-", "Delete category"},
	} {
		_ = tt.RightClick(label)
		if got := tt.Menu(); !slices.Equal(got, want) {
			t.Errorf("menu of %s %q, want %q", label, got, want)
		}
		tt.Key(0, ui.KeyEscape)
	}
}

func TestMoveDownAndUpReorderTheSidebarBuiltInsIncluded(t *testing.T) {
	tt, _, app := newMainTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("Work")
	if err := tt.ChooseMenuItem("Move down"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !above(t, tt, "Life", "Work") || !slices.Equal(categoryNames(app), []string{"Life", "Work"}) {
		t.Errorf("Work should be under Life: %v", categoryNames(app))
	}
	_ = tt.RightClick("Work")
	if err := tt.ChooseMenuItem("Move up"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !above(t, tt, "Work", "Life") {
		t.Errorf("Work should be above Life again: %v", categoryNames(app))
	}
}

func TestDeletingACategoryUnsortsItsTasksAndUndoBringsItBack(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "Reading", "orange")
	for _, id := range []int{18, 19} {
		_ = app.Update(id, data.Patch{Category: data.IntPtr(reading.ID)})
	}
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenMine, cat: reading.ID})
	tt.Frame()
	_ = tt.RightClick("Reading")
	if err := tt.ChooseMenuItem("Delete category"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := app.Category(reading.ID); ok {
		t.Fatal("the category is still there")
	}
	for _, id := range []int{18, 19} {
		if got, _ := app.Task(id); got.CategoryID != 0 {
			t.Errorf("task %d is in category %d", id, got.CategoryID)
		}
	}
	if !tt.HasText("Deleted “Reading” · 2 tasks unsorted") || !tt.HasText("Undo") {
		t.Errorf("no toast with Undo: %q", tt.Texts())
	}
	if m.nav != (nav{}) {
		t.Errorf("nav %+v: the page of the deleted category is gone", m.nav)
	}
	click(t, tt, "Undo")
	back, ok := app.Category(reading.ID)
	if !ok || back.Name != "Reading" || back.Color != "orange" || back.Position != 3 {
		t.Fatalf("restored %+v %v", back, ok)
	}
	for _, id := range []int{18, 19} {
		if got, _ := app.Task(id); got.CategoryID != reading.ID {
			t.Errorf("task %d is in category %d after Undo", id, got.CategoryID)
		}
	}
}

func TestDeletingAnEmptyCategoryHasNoCountInTheToast(t *testing.T) {
	app, _, _ := newTestApp(t)
	addCategory(t, app, "Reading", "orange")
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	_ = tt.RightClick("Reading")
	_ = tt.ChooseMenuItem("Delete category")
	tt.Frame()
	if !tt.HasText("Deleted “Reading”") {
		t.Errorf("toast: %q", tt.Texts())
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
		t.Errorf("nav %+v: a category made from the task menu does not open its page", m.nav)
	}
}

func TestTheSheetSpeaksChinese(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	tt.Frame()
	_ = tt.RightClick("我的")
	if got, want := tt.Menu(), []string{"新建类别…"}; !slices.Equal(got, want) {
		t.Fatalf("menu %q, want %q", got, want)
	}
	_ = tt.ChooseMenuItem("新建类别…")
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
	tt.Key(0, ui.KeyEscape)
	_ = tt.RightClick("工作")
	if got, want := tt.Menu(), []string{"新建类别…", "编辑类别…", "下移"}; !slices.Equal(got, want) {
		t.Errorf("menu of 工作 %q, want %q", got, want)
	}
	_ = tt.ChooseMenuItem("编辑类别…")
	tt.Frame()
	if !tt.HasText("工作和生活不能改名。") || !tt.HasText("保存") {
		t.Errorf("edit sheet: %q", tt.Texts())
	}
}

func TestTheDeleteToastSpeaksChinese(t *testing.T) {
	app, _, _ := newTestApp(t)
	reading := addCategory(t, app, "阅读", "orange")
	_ = app.Update(18, data.Patch{Category: data.IntPtr(reading.ID)})
	tt, _, _ := newTesterFor(app, 1100)
	inChinese(tt)
	tt.Frame()
	_ = tt.RightClick("阅读")
	_ = tt.ChooseMenuItem("删除类别")
	tt.Frame()
	if !tt.HasText("已删除“阅读” · 1 个任务变为未分类") || !tt.HasText("撤销") {
		t.Errorf("toast: %q", tt.Texts())
	}
}
