package views

import (
	"image"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// The category of a task is set by hand in the inspector and in the task menu, like its priority.

func TestInspectorCategoryRowSitsUnderPriorityAndOpensTheCategoryPage(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(3) // Fix tray race: an agent's task, nobody classified it
	tt.Frame()
	if !above(t, tt, "Priority", "Category") || !above(t, tt, "Category", "Project") {
		t.Errorf("Category goes between Priority and Project: %q", tt.Texts())
	}
	click(t, tt, "Category: None")
	for _, want := range []string{"Find a category", "Set category to Work", "Set category to Life", "Set category to None"} {
		if !tt.HasText(want) {
			t.Errorf("the Category value should open the category page, missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Set status to Done") {
		t.Error("the Category value opened more than its own page")
	}
	click(t, tt, "Set category to Life")
	if got, _ := app.Task(3); got.CategoryID != lifeID {
		t.Errorf("category %d", got.CategoryID)
	}
	if !tt.HasText("Category: Life") {
		t.Errorf("the row does not show the new value: %q", tt.Texts())
	}
	click(t, tt, "Category: Life")
	click(t, tt, "Set category to None")
	if got, _ := app.Task(3); got.CategoryID != 0 {
		t.Errorf("category %d after None", got.CategoryID)
	}
}

func TestTaskMenuMainPageHasACategoryRowThatOpensTheCategoryPage(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	reading := addCategory(t, app, "Reading", "orange")
	rightClick(t, tt, "Fix tray race on launch")
	if !tt.HasText("Category") || tt.HasText("Set category to Work") {
		t.Errorf("the main page has one Category row, not a strip: %q", tt.Texts())
	}
	if !above(t, tt, "Priority", "Category") || !above(t, tt, "Category", "Move to…") {
		t.Errorf("the row sits under the priority strip, above Move to: %q", tt.Texts())
	}
	click(t, tt, "Category")
	for _, want := range []string{"Find a category", "Set category to Work", "Set category to Life", "Set category to Reading", "Set category to None"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, pair := range [][2]string{{"Set category to Life", "Set category to Reading"}, {"Set category to Reading", "Set category to None"}} {
		if !above(t, tt, pair[0], pair[1]) {
			t.Errorf("%q should be above %q", pair[0], pair[1])
		}
	}
	click(t, tt, "Set category to Reading")
	if got, _ := app.Task(3); got.CategoryID != reading.ID {
		t.Errorf("category %d, want Reading", got.CategoryID)
	}
	if tt.HasText("Find a category") {
		t.Error("the menu stays open after a choice")
	}
}

func TestTheCategoryRowOfTheTaskMenuNamesTheCurrentCategory(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	_ = app.Update(3, data.Patch{Category: data.IntPtr(lifeID)})
	rightClick(t, tt, "Fix tray race on launch")
	if n := countText(tt, "Life"); n != 1 {
		t.Errorf("%d times Life: the hint of the Category row, in %q", n, tt.Texts())
	}
}

func TestTheCategoryPageChecksTheCurrentOneAndNoneClears(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	addCategory(t, app, "Reading", "orange")
	_ = app.Update(3, data.Patch{Category: data.IntPtr(lifeID)})
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	life, _ := tt.Find("Set category to Life")
	reading, _ := tt.Find("Set category to Reading")
	slot := func(r ui.Rect) ui.Rect { return ui.Rect{X: r.X + r.W - 36, Y: r.Y + 4, W: 28, H: r.H - 8} }
	if !paintedAnything(tt.Image(), 1100, slot(life)) {
		t.Error("no check at the right of Life, the current category")
	}
	if paintedAnything(tt.Image(), 1100, slot(reading)) {
		t.Error("a check at the right of Reading")
	}
	click(t, tt, "Set category to None")
	if got, _ := app.Task(3); got.CategoryID != 0 {
		t.Errorf("category %d after None", got.CategoryID)
	}
}

func TestTheCategoryPageFiltersByWhatIsTyped(t *testing.T) {
	tt, _, app := projectTester(t, 1)
	addCategory(t, app, "Reading", "orange")
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	tt.Type("read")
	if !tt.HasText("Set category to Reading") || tt.HasText("Set category to Work") || tt.HasText("Set category to None") {
		t.Errorf("filtered to Reading: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if got, _ := app.Task(3); got.CategoryID == 0 {
		t.Error("Enter did not choose the highlighted row")
	}
}

func TestTheCategoryPageHasNoMatchLine(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	tt.Type("zzz")
	if !tt.HasText("No category matches") {
		t.Errorf("no word for an empty filter: %q", tt.Texts())
	}
}

func TestCategoryGivesATaskTheChipOfThatCategory(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	rightClick(t, tt, "Write the release notes")
	click(t, tt, "Category")
	click(t, tt, "Set category to Work")
	tt.Frame()
	if !tt.HasText("Show only Work") {
		t.Errorf("a classified task has no Work chip: %q", tt.Texts())
	}
}

func TestInChineseTheCategoryRowSpeaksChinese(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	inChinese(tt)
	m.OpenTask(18)
	tt.Frame()
	for _, want := range []string{"工作", "生活", "分类"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestTheCategoryPageDrawsEachIconInItsColor(t *testing.T) {
	tt, m, app := projectTester(t, 1)
	addCategory(t, app, "Reading", "orange")
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	for _, cat := range app.snap.categories {
		label, key := "Set category to "+categoryTitle(m.say, cat), cat.Color
		if cat.Builtin == data.BuiltinWork {
			key = "indigo" // the built-in Work icon is drawn in iris, whatever color the category is stored with
		}
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("no %q", label)
		}
		if !paintedIn(tt.Image(), 1100, ui.Rect{X: r.X, Y: r.Y, W: 30, H: r.H}, theme.CategoryColor(key), 40) {
			t.Errorf("%s: no %s pixel in its icon", label, key)
		}
	}
}

// A category row draws its icon once: a second, one-color copy stacks the colored one under it and
// pushes it out of the 18 pt slot.
func TestTheCategoryPageDrawsEachIconOnce(t *testing.T) {
	tt, _, _ := projectTester(t, 1)
	rightClick(t, tt, "Fix tray race on launch")
	click(t, tt, "Category")
	r, ok := tt.Find("Set category to Life")
	if !ok {
		t.Fatalf("no Life row in %q", tt.Texts())
	}
	if h := inkHeight(tt.Image(), 1100, ui.Rect{X: r.X + 4, Y: r.Y + 1, W: 26, H: r.H - 2}); h > 18 {
		t.Errorf("Life's icon is drawn %v pt tall, more than its 18 pt slot: drawn twice", h)
	}
}

// inkHeight is how tall, in points, the drawing in r is: from its top to its bottom pixel that
// differs from the pixel at r's top left corner.
func inkHeight(img *image.RGBA, width float32, r ui.Rect) float32 {
	scale := float32(img.Bounds().Dx()) / width
	base := img.RGBAAt(int(r.X*scale), int(r.Y*scale))
	top, bottom := -1, -1
	for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
		for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
			p := img.RGBAAt(x, y)
			if absInt(int(p.R)-int(base.R))+absInt(int(p.G)-int(base.G))+absInt(int(p.B)-int(base.B)) > 60 {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	if top < 0 {
		return 0
	}
	return float32(bottom-top+1) / scale
}

func TestTheInspectorCategoryValueHasTheIconOfItsCategory(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	m.OpenTask(3) // Fix tray race: an agent's task, nobody classified it
	tt.Frame()
	none := inkWidth(t, tt, "Category: None")
	_ = app.Update(3, data.Patch{Category: data.IntPtr(lifeID)})
	tt.Frame()
	// Life is a shorter word than None; with the house before it, it draws wider.
	if life := inkWidth(t, tt, "Category: Life"); life < none+12 {
		t.Errorf("Life draws %v pt against None's %v pt: no icon before it", life, none)
	}
}

// A category the user made, which a task can be sorted into by an agent or by the window, reads by
// its own name and bitmap in the inspector; it is not on the strip, which stays Work, Life and None.
func TestTheInspectorShowsATaskInAUserCategoryByName(t *testing.T) {
	tt, m, app := newMainTester(t, 1100)
	reading, err := app.Store.CreateCategory(data.NewCategory{Name: "Reading", Icon: pixmap.Bitmap{}.Set(5, 5, true).Picture()})
	if err != nil {
		t.Fatal(err)
	}
	_ = app.Update(18, data.Patch{Category: data.IntPtr(reading.ID)})
	m.OpenTask(18)
	tt.Frame()
	if !tt.HasText("Category: Reading") {
		t.Fatalf("the inspector does not name the category: %q", tt.Texts())
	}
	if id := app.categoryOf(data.Task{CategoryID: reading.ID}); id != reading.ID {
		t.Errorf("categoryOf = %d, want the task's own", id)
	}
	if w := app.workID(); w != workID {
		t.Errorf("workID = %d, want %d", w, workID)
	}
}

// inkWidth is how wide, in points, the drawing inside an element is: from its leftmost to its
// rightmost pixel that differs from the element's own background, read just under its top edge.
func inkWidth(t *testing.T, tt *ui.Tester, label string) float32 {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("no %q in %q", label, tt.Texts())
	}
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / 1100
	base := img.RGBAAt(int((r.X+r.W/2)*scale), int((r.Y+1)*scale))
	left, right := -1, -1
	for y := int((r.Y + 1) * scale); y < int((r.Y+r.H-1)*scale); y++ {
		for x := int((r.X + 8) * scale); x < int((r.X+r.W-8)*scale); x++ {
			p := img.RGBAAt(x, y)
			if absInt(int(p.R)-int(base.R))+absInt(int(p.G)-int(base.G))+absInt(int(p.B)-int(base.B)) > 60 {
				if left < 0 || x < left {
					left = x
				}
				right = max(right, x)
			}
		}
	}
	if left < 0 {
		return 0
	}
	return float32(right-left+1) / scale
}

// The seeded stores put Work and Life at ids 1 and 2, as a new database does.
const workID, lifeID = 1, 2

// addCategory makes a category with a one-pixel icon.
func addCategory(t *testing.T, app *App, name, color string) data.Category {
	t.Helper()
	c, err := app.CreateCategory(data.NewCategory{Name: name, Icon: pixmap.Bitmap{}.Set(3, 3, true).Picture(), Color: color})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
