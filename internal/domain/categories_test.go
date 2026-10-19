package domain_test

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// dot is the smallest icon a category may have: one pixel, as the 144-character grid.
var dot = pixmap.Bitmap{}.Set(5, 5, true).Grid()

func newCategory(t *testing.T, c *domain.Core, name string) *domain.Category {
	t.Helper()
	return must[*domain.Category](t)(c.CreateCategory(ctx, domain.CreateCategoryInput{Name: name, Icon: dot}))
}

func categoryNames(t *testing.T, c *domain.Core) []string {
	t.Helper()
	var out []string
	for i, cat := range must[[]domain.Category](t)(c.ListCategories(ctx)) {
		eq(t, "position of "+cat.Name, cat.Position, i+1)
		out = append(out, cat.Name)
	}
	return out
}

func wantNames(t *testing.T, c *domain.Core, want ...string) {
	t.Helper()
	if got := strings.Join(categoryNames(t, c), ","); got != strings.Join(want, ",") {
		t.Errorf("categories = %s, want %s", got, strings.Join(want, ","))
	}
}

func TestANewDatabaseHasWorkAndLife(t *testing.T) {
	c, _ := newCore(t)
	cats := must[[]domain.Category](t)(c.ListCategories(ctx))
	eq(t, "count", len(cats), 2)
	eq(t, "work id", cats[0].ID, int64(1))
	eq(t, "work builtin", cats[0].Builtin, domain.BuiltinWork)
	eq(t, "work name", cats[0].Name, "Work")
	eq(t, "life builtin", cats[1].Builtin, domain.BuiltinLife)
	briefcase := must[pixmap.Bitmap](t)(pixmap.ParseHex("0000600903fc4024024024024024023fc000"))
	eq(t, "work icon", cats[0].Icon, briefcase.Grid())
	eq(t, "work color", cats[0].Color, "blue")
	eq(t, "life color", cats[1].Color, "green")
}

func TestCreateCategoryAppendsAndTrims(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "  Reading ")
	eq(t, "name", reading.Name, "Reading")
	eq(t, "position", reading.Position, 3)
	eq(t, "builtin", reading.Builtin, "")
	eq(t, "icon", reading.Icon, dot)
	newCategory(t, c, "Garden")
	wantNames(t, c, "Work", "Life", "Reading", "Garden")
}

func TestACategoryTakesTheColorItsNameHashesToUnlessGivenOne(t *testing.T) {
	c, _ := newCore(t)
	for _, name := range []string{"Reading", "Garden", "Gym", "读书"} {
		want := domain.DefaultCategoryColor(name)
		if !contains(strings.Join(domain.CategoryColors, " "), want) {
			t.Fatalf("%s: %q is not a category color", name, want)
		}
		eq(t, name+" color", newCategory(t, c, name).Color, want)
	}
	// A given color wins, and the hash is the one that colors a project's folder: FNV-1a over the
	// bytes of the name (theme.ProjectColor tests the same names against theme.ProjectColors).
	got := must[*domain.Category](t)(c.CreateCategory(ctx, domain.CreateCategoryInput{Name: "Pets", Icon: dot, Color: "pink"}))
	eq(t, "given color", got.Color, "pink")
	eq(t, "fnv of Reading", domain.DefaultCategoryColor("Reading"), domain.CategoryColors[fnv32a("Reading")%8])
	eq(t, "eight colors", len(domain.CategoryColors), 8)
}

func fnv32a(s string) uint32 {
	h := uint32(2166136261)
	for _, b := range []byte(s) {
		h = (h ^ uint32(b)) * 16777619
	}
	return h
}

func TestCreateCategoryChecksItsInput(t *testing.T) {
	c, _ := newCore(t)
	newCategory(t, c, "Reading")
	cases := map[string]struct {
		in    domain.CreateCategoryInput
		field string
		msg   string
	}{
		"empty":            {domain.CreateCategoryInput{Name: "  ", Icon: dot}, "name", "must not be empty"},
		"too long":         {domain.CreateCategoryInput{Name: strings.Repeat("长", 33), Icon: dot}, "name", "at most 32 characters"},
		"same name":        {domain.CreateCategoryInput{Name: "Reading", Icon: dot}, "name", `a category named "Reading" already exists`},
		"other case":       {domain.CreateCategoryInput{Name: "reAding", Icon: dot}, "name", "already exists"},
		"built-in key":     {domain.CreateCategoryInput{Name: "work", Icon: dot}, "name", "already exists"},
		"built-in Chinese": {domain.CreateCategoryInput{Name: "工作", Icon: dot}, "name", "already exists"},
		"life Chinese":     {domain.CreateCategoryInput{Name: "生活", Icon: dot}, "name", "already exists"},
		"no icon":          {domain.CreateCategoryInput{Name: "Blank"}, "icon", "must be 144 characters of . and #"},
		"no pixel":         {domain.CreateCategoryInput{Name: "Blank", Icon: strings.Repeat(".", 144)}, "icon", "must have at least one pixel"},
		"short icon":       {domain.CreateCategoryInput{Name: "Blank", Icon: dot[:100]}, "icon", "must be 144 characters of . and #"},
		"palette letter":   {domain.CreateCategoryInput{Name: "Blank", Icon: dot[:5] + "a" + dot[6:]}, "icon", "must be 144 characters of . and #"},
		"bad color":        {domain.CreateCategoryInput{Name: "Blank", Icon: dot, Color: "mauve"}, "color", "must be one of indigo, yellow, green, red, blue, pink, teal, orange"},
	}
	for name, tc := range cases {
		_, err := c.CreateCategory(ctx, tc.in)
		if err == nil || !strings.Contains(err.Error(), tc.field+": ") || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("%s: err = %v, want %s: %s", name, err, tc.field, tc.msg)
		}
	}
	if _, err := c.CreateCategory(ctx, domain.CreateCategoryInput{Name: "Reading", Icon: dot}); !domain.IsConflict(err) {
		t.Errorf("a taken name is a conflict, got %v", err)
	}
	if _, err := c.CreateCategory(ctx, domain.CreateCategoryInput{Name: strings.Repeat("a", 32), Icon: dot}); err != nil {
		t.Errorf("32 characters is allowed: %v", err)
	}
}

func TestAtMostTwentyFourCategories(t *testing.T) {
	c, _ := newCore(t)
	for i := 3; i <= 24; i++ {
		newCategory(t, c, strings.Repeat("c", i)) // distinct names
	}
	_, err := c.CreateCategory(ctx, domain.CreateCategoryInput{Name: "one more", Icon: dot})
	if err == nil || !strings.Contains(err.Error(), "category: at most 24 categories; delete one first") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateCategoryRenamesAndRedraws(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	newCategory(t, c, "Garden")
	other := pixmap.Bitmap{}.Set(5, 5, true).Set(0, 0, true).Grid()
	got := must[*domain.Category](t)(c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Name: ptr("reading list"), Icon: &other}))
	eq(t, "name", got.Name, "reading list")
	eq(t, "icon", got.Icon, other)
	eq(t, "position", got.Position, 3)
	eq(t, "an omitted color stays", got.Color, reading.Color)
	// Only the case changes: not a clash with itself.
	got = must[*domain.Category](t)(c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Name: ptr("Reading List")}))
	eq(t, "recased", got.Name, "Reading List")
	eq(t, "an omitted icon stays", got.Icon, other)
	if _, err := c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Name: ptr("garden")}); !domain.IsConflict(err) {
		t.Errorf("rename onto another category: %v", err)
	}
	_, err := c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Name: ptr(" ")})
	wantFieldError(t, err, "name")
	blank := strings.Repeat(".", 144)
	_, err = c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Icon: &blank})
	wantFieldError(t, err, "icon")
	_, err = c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Color: ptr("mauve")})
	wantFieldError(t, err, "color")
	recolored := must[*domain.Category](t)(c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Color: ptr("teal")}))
	eq(t, "color", recolored.Color, "teal")
	eq(t, "name stays", recolored.Name, "Reading List")
	if _, err := c.UpdateCategory(ctx, 99, domain.CategoryPatch{Name: ptr("x")}); !domain.IsNotFound(err) {
		t.Errorf("missing category: %v", err)
	}
}

func TestBuiltInCategoriesKeepTheirNamesButCanChangeIcons(t *testing.T) {
	c, _ := newCore(t)
	_, err := c.UpdateCategory(ctx, 1, domain.CategoryPatch{Name: ptr("Job")})
	if err == nil || !strings.Contains(err.Error(), "name: Work and Life keep their names") {
		t.Fatalf("err = %v", err)
	}
	got := must[*domain.Category](t)(c.UpdateCategory(ctx, 1, domain.CategoryPatch{Icon: &dot, Color: ptr("red")}))
	eq(t, "icon", got.Icon, dot)
	eq(t, "color", got.Color, "red")
	eq(t, "name", got.Name, "Work")
	_, err = c.DeleteCategory(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "category_id: Work and Life cannot be deleted") {
		t.Fatalf("err = %v", err)
	}
}

func TestMoveCategoryRenumbersInOneStep(t *testing.T) {
	c, _ := newCore(t)
	a, b := newCategory(t, c, "A"), newCategory(t, c, "B")
	noErr(t, c.MoveCategory(ctx, b.ID, 0))
	wantNames(t, c, "B", "Work", "Life", "A")
	noErr(t, c.MoveCategory(ctx, b.ID, 3))
	wantNames(t, c, "Work", "Life", "A", "B")
	noErr(t, c.MoveCategory(ctx, a.ID, 2))
	wantNames(t, c, "Work", "Life", "A", "B")
	noErr(t, c.MoveCategory(ctx, 1, 1)) // built-ins move too
	wantNames(t, c, "Life", "Work", "A", "B")
	wantFieldError(t, c.MoveCategory(ctx, a.ID, 4), "index")
	wantFieldError(t, c.MoveCategory(ctx, a.ID, -1), "index")
	if err := c.MoveCategory(ctx, 99, 0); !domain.IsNotFound(err) {
		t.Errorf("missing category: %v", err)
	}
}

func TestDeleteCategoryLeavesItsTasksUnsortedAndRenumbers(t *testing.T) {
	c, _ := newCore(t)
	a, b := newCategory(t, c, "A"), newCategory(t, c, "B")
	inA1 := mustTask(t, c, domain.CreateTaskInput{Title: "a1", CategoryID: &a.ID})
	inA2 := mustTask(t, c, domain.CreateTaskInput{Title: "a2", CategoryID: &a.ID})
	inB := mustTask(t, c, domain.CreateTaskInput{Title: "b", CategoryID: &b.ID})

	deleted := must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, a.ID))
	eq(t, "deleted name", deleted.Category.Name, "A")
	eq(t, "task ids", len(deleted.TaskIDs), 2)
	wantNames(t, c, "Work", "Life", "B")
	for _, id := range []int64{inA1.ID, inA2.ID} {
		got := must[*domain.Task](t)(c.GetTask(ctx, id))
		if got.CategoryID != nil || got.Category != nil {
			t.Errorf("task %d still has category %v %v", id, got.CategoryID, got.Category)
		}
	}
	eqPtr(t, "the other category's task is untouched", must[*domain.Task](t)(c.GetTask(ctx, inB.ID)).CategoryID, &b.ID)
	if _, err := c.DeleteCategory(ctx, a.ID); !domain.IsNotFound(err) {
		t.Errorf("second delete: %v", err)
	}
}

func TestRestoreCategoryPutsItBackWithItsTasks(t *testing.T) {
	c, _ := newCore(t)
	a, b := newCategory(t, c, "A"), newCategory(t, c, "B")
	t1 := mustTask(t, c, domain.CreateTaskInput{Title: "a1", CategoryID: &a.ID})
	t2 := mustTask(t, c, domain.CreateTaskInput{Title: "a2", CategoryID: &a.ID})
	gone := mustTask(t, c, domain.CreateTaskInput{Title: "gone", CategoryID: &a.ID})
	moved := mustTask(t, c, domain.CreateTaskInput{Title: "moved", CategoryID: &a.ID})
	deleted := must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, a.ID))
	noErr(t, c.DeleteTask(ctx, gone.ID))
	// Sorted into B meanwhile: the undo must not take it back.
	must[*domain.Task](t)(c.UpdateTask(ctx, moved.ID, domain.TaskPatch{CategoryID: domain.Set(b.ID)}))

	got := must[*domain.Category](t)(c.RestoreCategory(ctx, deleted))
	eq(t, "id", got.ID, a.ID)
	eq(t, "position", got.Position, 3)
	eq(t, "color", got.Color, a.Color)
	eq(t, "icon", got.Icon, a.Icon)
	wantNames(t, c, "Work", "Life", "A", "B")
	eqPtr(t, "t1", must[*domain.Task](t)(c.GetTask(ctx, t1.ID)).CategoryID, &a.ID)
	eqPtr(t, "t2", must[*domain.Task](t)(c.GetTask(ctx, t2.ID)).CategoryID, &a.ID)
	eqPtr(t, "moved", must[*domain.Task](t)(c.GetTask(ctx, moved.ID)).CategoryID, &b.ID)
	if got := must[*domain.Task](t)(c.GetTask(ctx, t1.ID)); got.Category == nil || *got.Category != "A" {
		t.Errorf("category name = %v", got.Category)
	}
}

func TestRestoreCategoryRefusesWhatNowClashes(t *testing.T) {
	c, _ := newCore(t)
	a := newCategory(t, c, "A")
	deleted := must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, a.ID))
	newCategory(t, c, "a") // the name was taken meanwhile
	if _, err := c.RestoreCategory(ctx, deleted); !domain.IsConflict(err) {
		t.Errorf("err = %v, want a conflict", err)
	}
	// The id was taken: a restored category is the same row, not a look-alike.
	c2, _ := newCore(t)
	x := newCategory(t, c2, "X")
	if _, err := c2.RestoreCategory(ctx, domain.DeletedCategory{Category: *x}); !domain.IsConflict(err) {
		t.Errorf("a category that is still there: %v", err)
	}
}

func TestRestoreCategoryClampsItsPosition(t *testing.T) {
	c, _ := newCore(t)
	newCategory(t, c, "A")
	b := newCategory(t, c, "B")
	deleted := must[domain.DeletedCategory](t)(c.DeleteCategory(ctx, b.ID))
	noErr(t, c.MoveCategory(ctx, 2, 0))
	deleted.Category.Position = 40
	got := must[*domain.Category](t)(c.RestoreCategory(ctx, deleted))
	eq(t, "position", got.Position, 4)
	wantNames(t, c, "Life", "Work", "A", "B")
}

func TestResolveCategoryReadsNamesAndBuiltInAliases(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	for q, want := range map[string]int64{
		"Work": 1, "work": 1, "WORK": 1, " work ": 1, "工作": 1, "Life": 2, "life": 2, "生活": 2,
		"Reading": reading.ID, "reading": reading.ID, "READING": reading.ID,
	} {
		got, err := c.ResolveCategory(ctx, q)
		if err != nil || got != want {
			t.Errorf("ResolveCategory(%q) = %d, %v; want %d", q, got, err, want)
		}
	}
	for _, q := range []string{"play", "", "  ", "read"} {
		_, err := c.ResolveCategory(ctx, q)
		if q != "" && q != "  " && (err == nil || !strings.Contains(err.Error(), `no category named "`+q+`"; the categories are Work, Life, Reading`)) {
			t.Errorf("ResolveCategory(%q) err = %v", q, err)
		}
		if err == nil {
			t.Errorf("ResolveCategory(%q) succeeded", q)
		}
	}
}

func TestResolveCategoryFollowsARenamedCategoryNotTheOldName(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	must[*domain.Category](t)(c.UpdateCategory(ctx, reading.ID, domain.CategoryPatch{Name: ptr("Books")}))
	if _, err := c.ResolveCategory(ctx, "Reading"); err == nil {
		t.Error("the old name still resolves")
	}
	if id, err := c.ResolveCategory(ctx, "books"); err != nil || id != reading.ID {
		t.Errorf("new name: %d %v", id, err)
	}
}
