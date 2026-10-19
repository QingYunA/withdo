package data

import (
	"fmt"
	"slices"
	"sort"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// The briefcase and the house, as migration 012 seeds Work and Life.
const (
	workIconHex = "0000600903fc4024024024024024023fc000"
	lifeIconHex = "0000600901082044024624924924923fc000"
)

// builtinCategories are Work (1, blue) and Life (2, green) as a new database has them.
func builtinCategories() []Category {
	return []Category{
		{ID: 1, Builtin: BuiltinWork, Name: "Work", Icon: pixmap.MustParseHex(workIconHex), Color: "blue", Position: 1},
		{ID: 2, Builtin: BuiltinLife, Name: "Life", Icon: pixmap.MustParseHex(lifeIconHex), Color: "green", Position: 2},
	}
}

func (f *Fake) category(id int) (*Category, bool) {
	for i := range f.categories {
		if f.categories[i].ID == id {
			return &f.categories[i], true
		}
	}
	return nil, false
}

// checkCategoryRef is the check the database makes of a task's category: it must exist; 0 is none.
func (f *Fake) checkCategoryRef(id int) error {
	if _, ok := f.category(id); id != 0 && !ok {
		return fmt.Errorf("category %d not found", id)
	}
	return nil
}

// domainCategories are the categories as the domain layer checks names against.
func (f *Fake) domainCategories() []domain.Category {
	out := make([]domain.Category, len(f.categories))
	for i, c := range f.categories {
		out[i] = domain.Category{ID: int64(c.ID), Builtin: c.Builtin, Name: c.Name}
	}
	return out
}

func (f *Fake) ListCategories() ([]Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sortedCategories(), nil
}

func (f *Fake) sortedCategories() []Category {
	out := slices.Clone(f.categories)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out
}

func (f *Fake) CreateCategory(in NewCategory) (Category, error) {
	f.mu.Lock()
	name, err := domain.CheckCategoryName(f.domainCategories(), 0, in.Name)
	if err != nil {
		f.mu.Unlock()
		return Category{}, categoryProblem(err)
	}
	color := in.Color
	if color == "" {
		color = domain.DefaultCategoryColor(name)
	}
	switch {
	case in.Icon.Empty():
		err = ErrIconEmpty
	case !slices.Contains(domain.CategoryColors, color):
		err = ErrCategoryColor
	case len(f.categories) >= MaxCategories:
		err = ErrTooManyCategories
	}
	if err != nil {
		f.mu.Unlock()
		return Category{}, err
	}
	c := Category{ID: f.newID(), Name: name, Icon: in.Icon, Color: color, Position: len(f.categories) + 1}
	f.categories = append(f.categories, c)
	f.mu.Unlock()
	f.notify()
	return c, nil
}

func (f *Fake) UpdateCategory(id int, p CategoryPatch) (Category, error) {
	f.mu.Lock()
	c, ok := f.category(id)
	if !ok {
		f.mu.Unlock()
		return Category{}, fmt.Errorf("category %d not found", id)
	}
	next := *c
	var err error
	if p.Name != nil {
		if c.Builtin != "" {
			err = ErrCategoryBuiltinName
		} else {
			var name string
			if name, err = domain.CheckCategoryName(f.domainCategories(), int64(id), *p.Name); err == nil {
				next.Name = name
			} else {
				err = categoryProblem(err)
			}
		}
	}
	if err == nil && p.Icon != nil {
		next.Icon = *p.Icon
		if next.Icon.Empty() {
			err = ErrIconEmpty
		}
	}
	if err == nil && p.Color != nil {
		next.Color = *p.Color
		if !slices.Contains(domain.CategoryColors, next.Color) {
			err = ErrCategoryColor
		}
	}
	if err != nil {
		f.mu.Unlock()
		return Category{}, err
	}
	*c = next
	f.mu.Unlock()
	f.notify()
	return next, nil
}

func (f *Fake) MoveCategory(id, index int) error {
	f.mu.Lock()
	order := f.sortedCategories()
	from := slices.IndexFunc(order, func(c Category) bool { return c.ID == id })
	switch {
	case from < 0:
		f.mu.Unlock()
		return fmt.Errorf("category %d not found", id)
	case index < 0 || index >= len(order):
		f.mu.Unlock()
		return fmt.Errorf("index: must be between 0 and %d", len(order)-1)
	}
	moved := order[from]
	order = slices.Delete(order, from, from+1)
	order = slices.Insert(order, index, moved)
	for i := range order {
		order[i].Position = i + 1
	}
	f.categories = order
	f.mu.Unlock()
	f.notify()
	return nil
}

func (f *Fake) DeleteCategory(id int) (DeletedCategory, error) {
	f.mu.Lock()
	c, ok := f.category(id)
	switch {
	case !ok:
		f.mu.Unlock()
		return DeletedCategory{}, fmt.Errorf("category %d not found", id)
	case c.Builtin != "":
		f.mu.Unlock()
		return DeletedCategory{}, ErrCategoryBuiltin
	}
	deleted := DeletedCategory{Category: *c}
	var ids []int
	for i := range f.tasks {
		if f.tasks[i].CategoryID == id {
			ids = append(ids, f.tasks[i].ID)
			f.tasks[i].CategoryID = 0
		}
	}
	deleted.Tasks, deleted.raw = len(ids), ids
	f.categories = slices.DeleteFunc(f.categories, func(c Category) bool { return c.ID == id })
	for i, c := range f.sortedCategories() {
		kept, _ := f.category(c.ID)
		kept.Position = i + 1
	}
	f.mu.Unlock()
	f.notify()
	return deleted, nil
}

func (f *Fake) RestoreCategory(d DeletedCategory) (Category, error) {
	f.mu.Lock()
	cat := d.Category
	if _, ok := f.category(cat.ID); ok {
		f.mu.Unlock()
		return Category{}, fmt.Errorf("category %d already exists", cat.ID)
	}
	if len(f.categories) >= MaxCategories {
		f.mu.Unlock()
		return Category{}, ErrTooManyCategories
	}
	if _, err := domain.CheckCategoryName(f.domainCategories(), 0, cat.Name); err != nil {
		f.mu.Unlock()
		return Category{}, categoryProblem(err)
	}
	at := min(max(cat.Position, 1), len(f.categories)+1)
	for i := range f.categories {
		if f.categories[i].Position >= at {
			f.categories[i].Position++
		}
	}
	cat.Position = at
	f.categories = append(f.categories, cat)
	ids, _ := d.raw.([]int)
	for _, id := range ids {
		if t, ok := f.find(id); ok && t.CategoryID == 0 {
			t.CategoryID = cat.ID
		}
	}
	f.mu.Unlock()
	f.notify()
	return cat, nil
}
