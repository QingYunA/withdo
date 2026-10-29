package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Settings > Categories is where categories are made, edited, moved and deleted.

func categoriesRig(t *testing.T) (*settingsRig, data.Category) {
	t.Helper()
	r := newSettingsRig(t)
	reading := addCategory(t, r.app, "Reading", "orange")
	r.pane(t, "Categories")
	// The category editor is taller than the window; the tests make room for its buttons.
	r.tt.SetSize(800, 1400)
	r.tt.Frame()
	return r, reading
}

// hoverCategory puts the pointer over a category row, which shows its tools for the next frame.
func hoverCategory(t *testing.T, r *settingsRig, name string) {
	t.Helper()
	rect, ok := r.tt.Find(name)
	if !ok {
		t.Fatalf("no category row %q in %q", name, r.tt.Texts())
	}
	r.tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	r.tt.Frame()
}

func TestSettingsCategoriesListInOrderWithAWayToAddOne(t *testing.T) {
	r, _ := categoriesRig(t)
	has(t, r.tt, "Categories", "Work", "Life", "Reading", "New category…", "12 tasks", "2 tasks")
	if !above(t, r.tt, "Work", "Life") || !above(t, r.tt, "Life", "Reading") || !above(t, r.tt, "Reading", "New category…") {
		t.Errorf("order: %q", r.tt.Texts())
	}
	hasNot(t, r.tt, "Edit Work", "Edit Reading", "Move Reading up", "Move Work down", "Delete Reading")
	hoverCategory(t, r, "Work")
	has(t, r.tt, "Edit Work", "Move Work down")
	hasNot(t, r.tt, "Delete Work", "Move Work up", "Edit Life") // Work and Life stay; nothing moves past an end
	hoverCategory(t, r, "Reading")
	has(t, r.tt, "Edit Reading", "Move Reading up", "Delete Reading")
	hasNot(t, r.tt, "Edit Work")
}

// The arrows, Edit and Delete belong to the row under the pointer, or to the row that has the
// keyboard, and nowhere else.
func TestCategoryActionsAppearOnHoverOrKeyboardFocus(t *testing.T) {
	r, _ := categoriesRig(t)
	hasNot(t, r.tt, "Edit Reading", "Move Reading up", "Delete Reading")
	hoverCategory(t, r, "Reading")
	has(t, r.tt, "Edit Reading", "Move Reading up", "Delete Reading")
	r.tt.Move(700, 900) // the pane's empty room
	r.tt.Frame()
	hasNot(t, r.tt, "Edit Reading", "Move Reading up", "Delete Reading")

	// A click gives the row the keyboard; the pointer may leave and the tools stay.
	if err := r.tt.Click("Reading"); err != nil {
		t.Fatal(err)
	}
	r.tt.Move(700, 900)
	r.tt.Frame()
	if !r.tt.Focused("Reading") {
		t.Fatal("clicking a category row did not focus it")
	}
	has(t, r.tt, "Edit Reading", "Move Reading up", "Delete Reading")
	r.tt.Key(0, ui.KeyTab)
	if !r.tt.Focused("Move Reading up") {
		t.Errorf("Tab did not reach the row's first tool")
	}
}

func TestSettingsNewCategoryOpensTheSheetAndAddsToTheList(t *testing.T) {
	r, _ := categoriesRig(t)
	click(t, r.tt, "New category…")
	r.tt.Frame()
	has(t, r.tt, "New category", "Name", "Icon", "Color", "Create")
	r.tt.Type("Garden")
	click(t, r.tt, "Create")
	r.tt.Frame()
	categoryNamed(t, r.app, "Garden")
	has(t, r.tt, "Garden")
	hasNot(t, r.tt, "Search icons")
}

func TestSettingsEditRenamesAndRecolorsACategory(t *testing.T) {
	r, reading := categoriesRig(t)
	hoverCategory(t, r, "Reading")
	click(t, r.tt, "Edit Reading")
	r.tt.Frame()
	has(t, r.tt, "Edit category", "Save")
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("Books")
	click(t, r.tt, "Color: Red")
	click(t, r.tt, "Save")
	r.tt.Frame()
	got, _ := r.app.Category(reading.ID)
	if got.Name != "Books" || got.Color != "red" {
		t.Errorf("category %+v, want Books in red", got)
	}
	has(t, r.tt, "Books")
}

func TestSettingsEditOfWorkKeepsItsNameReadOnly(t *testing.T) {
	r, _ := categoriesRig(t)
	hoverCategory(t, r, "Work")
	click(t, r.tt, "Edit Work")
	r.tt.Frame()
	has(t, r.tt, "Work and Life keep their names.", "Save")
}

func TestSettingsMoveReordersCategoriesBuiltInsIncluded(t *testing.T) {
	r, _ := categoriesRig(t)
	hoverCategory(t, r, "Work")
	click(t, r.tt, "Move Work down")
	r.tt.Frame()
	if got, want := categoryNames(r.app), []string{"Life", "Work", "Reading"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if !above(t, r.tt, "Life", "Work") {
		t.Errorf("the list did not follow: %q", r.tt.Texts())
	}
	hoverCategory(t, r, "Reading")
	click(t, r.tt, "Move Reading up")
	r.tt.Frame()
	if got, want := categoryNames(r.app), []string{"Life", "Reading", "Work"}; !slices.Equal(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

func TestSettingsDeleteAsksThenUnsortsTheTasksAndUndoBringsTheCategoryBack(t *testing.T) {
	r, reading := categoriesRig(t)
	for _, id := range []int{18, 19} {
		_ = r.app.Update(id, data.Patch{Category: data.IntPtr(reading.ID)})
	}
	hoverCategory(t, r, "Reading")
	click(t, r.tt, "Delete Reading")
	r.tt.Frame()
	has(t, r.tt, "Delete “Reading”? Its tasks become unsorted.", "Cancel")
	if _, ok := r.app.Category(reading.ID); !ok {
		t.Fatal("the first click deleted it")
	}
	click(t, r.tt, "Cancel")
	r.tt.Frame()
	has(t, r.tt, "Reading")
	hoverCategory(t, r, "Reading")
	click(t, r.tt, "Delete Reading")
	r.tt.Frame()
	click(t, r.tt, "Delete Reading") // the confirm link carries the same label
	r.tt.Frame()
	if _, ok := r.app.Category(reading.ID); ok {
		t.Fatal("the category is still there")
	}
	for _, id := range []int{18, 19} {
		if got, _ := r.app.Task(id); got.CategoryID != 0 {
			t.Errorf("task %d is in category %d", id, got.CategoryID)
		}
	}
	has(t, r.tt, "Deleted “Reading” · 2 tasks unsorted", "Undo")
	click(t, r.tt, "Undo")
	r.tt.Frame()
	back, ok := r.app.Category(reading.ID)
	if !ok || back.Name != "Reading" || back.Color != "orange" {
		t.Fatalf("restored %+v %v", back, ok)
	}
	for _, id := range []int{18, 19} {
		if got, _ := r.app.Task(id); got.CategoryID != reading.ID {
			t.Errorf("task %d is in category %d after Undo", id, got.CategoryID)
		}
	}
}

func TestSettingsCategoriesSpeakChinese(t *testing.T) {
	r, _ := categoriesRig(t)
	inChinese(r.tt)
	r.tt.Frame()
	has(t, r.tt, "类别", "工作", "生活", "12 个任务", "新建类别…")
	hoverCategory(t, r, "工作")
	has(t, r.tt, "编辑 工作", "下移 工作")
	hoverCategory(t, r, "Reading")
	has(t, r.tt, "上移 Reading")
}
