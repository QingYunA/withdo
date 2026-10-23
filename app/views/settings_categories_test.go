package views

import (
	"slices"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Settings > General > Categories is where categories are made, edited, moved and deleted, now that the
// sidebar has no rows for them.

func categoriesRig(t *testing.T) (*settingsRig, data.Category) {
	t.Helper()
	r := newSettingsRig(t)
	reading := addCategory(t, r.app, "Reading", "orange")
	r.tt.SetSize(680, 1400)
	r.tt.Frame()
	return r, reading
}

func TestSettingsGeneralListsTheCategoriesInOrderWithAWayToAddOne(t *testing.T) {
	r, _ := categoriesRig(t)
	has(t, r.tt, "Categories", "Work", "Life", "Reading", "New category…")
	if !above(t, r.tt, "Work", "Life") || !above(t, r.tt, "Life", "Reading") || !above(t, r.tt, "Reading", "New category…") {
		t.Errorf("order: %q", r.tt.Texts())
	}
	has(t, r.tt, "Edit Work", "Edit Reading", "Move Reading up", "Move Work down", "Delete Reading")
	hasNot(t, r.tt, "Delete Work", "Move Work up", "Move Reading down") // Work and Life stay; nothing moves past an end
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
	click(t, r.tt, "Edit Work")
	r.tt.Frame()
	has(t, r.tt, "Work and Life keep their names.", "Save")
}

func TestSettingsMoveReordersCategoriesBuiltInsIncluded(t *testing.T) {
	r, _ := categoriesRig(t)
	click(t, r.tt, "Move Work down")
	r.tt.Frame()
	if got, want := categoryNames(r.app), []string{"Life", "Work", "Reading"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	if !above(t, r.tt, "Life", "Work") {
		t.Errorf("the list did not follow: %q", r.tt.Texts())
	}
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
	click(t, r.tt, "Delete Reading")
	r.tt.Frame()
	has(t, r.tt, "Delete “Reading”? Its tasks become unsorted.", "Cancel")
	if _, ok := r.app.Category(reading.ID); !ok {
		t.Fatal("the first click deleted it")
	}
	click(t, r.tt, "Cancel")
	r.tt.Frame()
	has(t, r.tt, "Reading")
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
	has(t, r.tt, "类别", "工作", "生活", "新建类别…", "编辑 工作", "上移 Reading")
}
