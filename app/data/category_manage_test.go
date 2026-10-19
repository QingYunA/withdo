package data

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

var (
	dot       = pixmap.Bitmap{}.Set(5, 5, true).Picture()
	dotGrid   = dot.Grid()
	briefcase = pixmap.MustParseHex("0000600903fc4024024024024024023fc000").Picture()
	house     = pixmap.MustParseHex("0000600901082044024624924924923fc000").Picture()
)

func categoryNames(t *testing.T, s Store) []string {
	t.Helper()
	var out []string
	for i, c := range must[[]Category](t)(s.ListCategories()) {
		if c.Position != i+1 {
			t.Errorf("%s has position %d, want %d", c.Name, c.Position, i+1)
		}
		out = append(out, c.Name)
	}
	return out
}

func wantCategories(t *testing.T, s Store, want ...string) {
	t.Helper()
	if got := strings.Join(categoryNames(t, s), ","); got != strings.Join(want, ",") {
		t.Errorf("categories = %s, want %s", got, strings.Join(want, ","))
	}
}

func TestEveryStoreStartsWithWorkAndLife(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, _ *time.Time) {
		cats := must[[]Category](t)(s.ListCategories())
		if len(cats) != 2 {
			t.Fatalf("categories %+v", cats)
		}
		w, l := cats[0], cats[1]
		if w.ID != workID || w.Builtin != BuiltinWork || w.Name != "Work" || w.Color != "blue" || w.Icon != briefcase || w.Position != 1 {
			t.Errorf("work = %+v", w)
		}
		if l.ID != lifeID || l.Builtin != BuiltinLife || l.Name != "Life" || l.Color != "green" || l.Icon != house || l.Position != 2 {
			t.Errorf("life = %+v", l)
		}
	})
}

func TestCreateUpdateAndMoveCategories(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, _ *time.Time) {
		reading := must[Category](t)(s.CreateCategory(NewCategory{Name: "  Reading ", Icon: dot}))
		if reading.Name != "Reading" || reading.Icon != dot || reading.Builtin != "" || reading.Position != 3 {
			t.Errorf("created %+v", reading)
		}
		if reading.Color == "" {
			t.Error("a new category takes the color its name hashes to")
		}
		garden := must[Category](t)(s.CreateCategory(NewCategory{Name: "Garden", Icon: dot, Color: "teal"}))
		if garden.Color != "teal" {
			t.Errorf("color %q", garden.Color)
		}
		wantCategories(t, s, "Work", "Life", "Reading", "Garden")

		other := dot.Set(0, 0, 'd') // a palette pixel survives the store
		got := must[Category](t)(s.UpdateCategory(reading.ID, CategoryPatch{Name: StrPtr("Books"), Icon: &other, Color: StrPtr("pink")}))
		if got.Name != "Books" || got.Icon != other || got.Color != "pink" {
			t.Errorf("updated %+v", got)
		}
		kept := must[Category](t)(s.UpdateCategory(reading.ID, CategoryPatch{Color: StrPtr("red")}))
		if kept.Name != "Books" || kept.Icon != other {
			t.Errorf("a patch without them changed %+v", kept)
		}
		// Work and Life change their icon and color, not their names.
		w := must[Category](t)(s.UpdateCategory(workID, CategoryPatch{Icon: &dot, Color: StrPtr("red")}))
		if w.Icon != dot || w.Color != "red" || w.Name != "Work" {
			t.Errorf("work = %+v", w)
		}

		if err := s.MoveCategory(garden.ID, 0); err != nil {
			t.Fatal(err)
		}
		wantCategories(t, s, "Garden", "Work", "Life", "Books")
		if err := s.MoveCategory(workID, 3); err != nil { // built-ins move like any other
			t.Fatal(err)
		}
		wantCategories(t, s, "Garden", "Life", "Books", "Work")
		if err := s.MoveCategory(garden.ID, 9); err == nil {
			t.Error("a position past the end was accepted")
		}
	})
}

func TestCategoryProblemsAreTheStoresOwnErrors(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, _ *time.Time) {
		must[Category](t)(s.CreateCategory(NewCategory{Name: "Reading", Icon: dot}))
		for name, tc := range map[string]struct {
			in   NewCategory
			want error
		}{
			"empty":      {NewCategory{Name: "  ", Icon: dot}, ErrNameEmpty},
			"too long":   {NewCategory{Name: strings.Repeat("x", 33), Icon: dot}, ErrCategoryNameLong},
			"taken":      {NewCategory{Name: "reading", Icon: dot}, ErrCategoryNameTaken},
			"alias":      {NewCategory{Name: "工作", Icon: dot}, ErrCategoryNameTaken},
			"built-in":   {NewCategory{Name: "WORK", Icon: dot}, ErrCategoryNameTaken},
			"blank icon": {NewCategory{Name: "Blank"}, ErrIconEmpty},
			"bad color":  {NewCategory{Name: "Blank", Icon: dot, Color: "mauve"}, ErrCategoryColor},
		} {
			if _, err := s.CreateCategory(tc.in); !errors.Is(err, tc.want) {
				t.Errorf("%s: err = %v, want %v", name, err, tc.want)
			}
		}
		if _, err := s.UpdateCategory(workID, CategoryPatch{Name: StrPtr("Job")}); !errors.Is(err, ErrCategoryBuiltinName) {
			t.Errorf("renaming Work: %v", err)
		}
		if _, err := s.UpdateCategory(workID+10, CategoryPatch{Name: StrPtr("x")}); err == nil {
			t.Error("a category that does not exist was renamed")
		}
		if _, err := s.DeleteCategory(lifeID); !errors.Is(err, ErrCategoryBuiltin) {
			t.Errorf("deleting Life: %v", err)
		}
		for i := 4; i <= MaxCategories; i++ {
			must[Category](t)(s.CreateCategory(NewCategory{Name: strings.Repeat("c", i), Icon: dot}))
		}
		if _, err := s.CreateCategory(NewCategory{Name: "one more", Icon: dot}); !errors.Is(err, ErrTooManyCategories) {
			t.Errorf("the 25th category: %v", err)
		}
	})
}

func TestDeletingACategoryUnsortsItsTasksAndUndoSortsThemBack(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, _ *time.Time) {
		reading := must[Category](t)(s.CreateCategory(NewCategory{Name: "Reading", Icon: dot}))
		garden := must[Category](t)(s.CreateCategory(NewCategory{Name: "Garden", Icon: dot}))
		a := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", Category: reading.ID}))
		b := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "b", Category: reading.ID}))
		c := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "c", Category: garden.ID}))
		moved := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "moved", Category: reading.ID}))

		deleted := must[DeletedCategory](t)(s.DeleteCategory(reading.ID))
		if deleted.Category.Name != "Reading" || deleted.Tasks != 3 {
			t.Errorf("deleted %+v", deleted)
		}
		wantCategories(t, s, "Work", "Life", "Garden")
		for _, id := range []int{a.ID, b.ID, moved.ID} {
			if got := must[TaskDetail](t)(s.GetTask(id)).Task.CategoryID; got != 0 {
				t.Errorf("task %d still sorted into %d", id, got)
			}
		}
		// Sorted into Garden meanwhile: the undo does not take it away from there.
		must[Task](t)(s.UpdateTask(moved.ID, Patch{Category: IntPtr(garden.ID)}))

		back := must[Category](t)(s.RestoreCategory(deleted))
		if back.ID != reading.ID || back.Position != 3 || back.Icon != dot || back.Color != reading.Color {
			t.Errorf("restored %+v, was %+v", back, reading)
		}
		wantCategories(t, s, "Work", "Life", "Reading", "Garden")
		for _, id := range []int{a.ID, b.ID} {
			if got := must[TaskDetail](t)(s.GetTask(id)).Task.CategoryID; got != reading.ID {
				t.Errorf("task %d is in %d after the undo", id, got)
			}
		}
		if got := must[TaskDetail](t)(s.GetTask(moved.ID)).Task.CategoryID; got != garden.ID {
			t.Errorf("the task sorted elsewhere meanwhile is in %d", got)
		}
		if got := must[TaskDetail](t)(s.GetTask(c.ID)).Task.CategoryID; got != garden.ID {
			t.Errorf("an untouched task is in %d", got)
		}
		if _, err := s.RestoreCategory(deleted); err == nil {
			t.Error("a category that is back was restored twice")
		}
	})
}

// A deleted task comes back unsorted when its category went meanwhile, like its project.
func TestRestoringATaskWhoseCategoryIsGoneLeavesItUnsorted(t *testing.T) {
	withStore(t, func(t *testing.T, s Store, _ *time.Time) {
		reading := must[Category](t)(s.CreateCategory(NewCategory{Name: "Reading", Icon: dot}))
		task := must[Task](t)(s.CreateTask(CreateTaskInput{Title: "a", Category: reading.ID}))
		snap := must[TaskSnapshot](t)(s.SnapshotTask(task.ID))
		if err := s.DeleteTask(task.ID); err != nil {
			t.Fatal(err)
		}
		must[DeletedCategory](t)(s.DeleteCategory(reading.ID))
		if got := must[Task](t)(s.RestoreTask(snap)); got.CategoryID != 0 {
			t.Errorf("restored into %d", got.CategoryID)
		}
	})
}
