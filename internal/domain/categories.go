package domain

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode/utf8"

	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/QingYunA/adhd-todo/internal/store"
)

// The built-in categories. Work and Life are always there, under these keys, and cannot be renamed
// or deleted: a task in a repository's project counts as Work unless it says otherwise.
const (
	BuiltinWork = "work"
	BuiltinLife = "life"
)

const (
	// MaxCategories is how many categories there can be, the built-in two included.
	MaxCategories = 24
	// MaxCategoryName is the longest category name, in characters.
	MaxCategoryName = 32

	categoryCols = "id, builtin, name, icon, color, position, created_at"
)

// CategoryColors are the keys of the eight colors a category can have, in the order of the app's
// project colors; the app turns a key into a color.
var CategoryColors = []string{"indigo", "yellow", "green", "red", "blue", "pink", "teal", "orange"}

// DefaultCategoryColor is the color a new category gets unless it is given one: the one its name
// hashes to, the way a project's folder gets its color.
func DefaultCategoryColor(name string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return CategoryColors[h.Sum32()%uint32(len(CategoryColors))]
}

// builtinAliases are the other names Work and Life answer to; user categories may not take them.
var builtinAliases = map[string]string{"工作": BuiltinWork, "生活": BuiltinLife}

// Category is a label the user sorts tasks under. Its JSON form matches the database columns.
type Category struct {
	ID int64 `json:"id"`
	// Builtin is "work" or "life" for the two built-in categories, "" for the user's own.
	Builtin string `json:"builtin"`
	// Name is what the user called it; the built-ins are stored as Work and Life and localized by
	// whoever shows them.
	Name string `json:"name"`
	// Icon is the 12 x 12 picture drawn beside the name: 144 characters, 12 rows of 12 with row 0
	// first, "." for an empty pixel, "#" for one in the category's color and a letter a to l for one in
	// a fixed palette color. See pixmap.ParseGrid.
	Icon string `json:"icon"`
	// Color is one of CategoryColors.
	Color string `json:"color"`
	// Position is the place in the list, 1-based and contiguous.
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
}

func scanCategory(s scanner) (Category, error) {
	var c Category
	var builtin *string
	if err := s.Scan(&c.ID, &builtin, &c.Name, &c.Icon, &c.Color, &c.Position, &c.CreatedAt); err != nil {
		return c, err
	}
	if builtin != nil {
		c.Builtin = *builtin
	}
	return c, nil
}

func listCategories(ctx context.Context, q store.Querier) ([]Category, error) {
	rows, err := q.QueryContext(ctx, "SELECT "+categoryCols+" FROM categories ORDER BY position, id")
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return collect(rows, scanCategory)
}

func getCategory(ctx context.Context, q store.Querier, id int64) (*Category, error) {
	return one(q.QueryRowContext(ctx, "SELECT "+categoryCols+" FROM categories WHERE id = ?", id), scanCategory)
}

// ListCategories returns every category in the user's order.
func (c *Core) ListCategories(ctx context.Context) ([]Category, error) {
	return listCategories(ctx, c.db)
}

// CreateCategoryInput describes a new category. Name is trimmed, at most MaxCategoryName characters
// and not taken by another category in any case, nor by an alias of Work or Life (work, life, 工作,
// 生活). Icon is a grid of 144 characters with at least one pixel set. An empty Color takes
// DefaultCategoryColor(Name).
type CreateCategoryInput struct {
	Name  string
	Icon  string
	Color string
}

// CreateCategory adds a category at the end of the list.
func (c *Core) CreateCategory(ctx context.Context, in CreateCategoryInput) (*Category, error) {
	var out *Category
	err := c.write(ctx, func(q store.Querier) error {
		name, err := cleanCategoryName(in.Name)
		if err != nil {
			return err
		}
		if err := checkCategoryIcon(in.Icon); err != nil {
			return err
		}
		color := in.Color
		if color == "" {
			color = DefaultCategoryColor(name)
		}
		if err := checkCategoryColor(color); err != nil {
			return err
		}
		all, err := listCategories(ctx, q)
		if err != nil {
			return err
		}
		if len(all) >= MaxCategories {
			return invalid("category", "at most %d categories; delete one first", MaxCategories)
		}
		if err := checkCategoryName(all, 0, name); err != nil {
			return err
		}
		newID, err := c.syncID(ctx, q, "categories")
		if err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, "INSERT INTO categories (id, name, icon, color, position, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			newID, name, in.Icon, color, len(all)+1, c.stamp())
		if err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
		out, err = getCategory(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return out, nil
}

// CategoryPatch changes a category; nil fields stay. The built-in categories keep their names but
// can change their icon and color.
type CategoryPatch struct {
	Name  *string
	Icon  *string
	Color *string
}

// UpdateCategory renames, redraws or recolors a category.
func (c *Core) UpdateCategory(ctx context.Context, id int64, patch CategoryPatch) (*Category, error) {
	var out *Category
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getCategory(ctx, q, id)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("category_id", "category", id)
		}
		next := *cur
		if patch.Name != nil {
			if cur.Builtin != "" {
				return invalid("name", "Work and Life keep their names")
			}
			if next.Name, err = cleanCategoryName(*patch.Name); err != nil {
				return err
			}
			all, err := listCategories(ctx, q)
			if err != nil {
				return err
			}
			if err := checkCategoryName(all, id, next.Name); err != nil {
				return err
			}
		}
		if patch.Icon != nil {
			if err := checkCategoryIcon(*patch.Icon); err != nil {
				return err
			}
			next.Icon = *patch.Icon
		}
		if patch.Color != nil {
			if err := checkCategoryColor(*patch.Color); err != nil {
				return err
			}
			next.Color = *patch.Color
		}
		if _, err := q.ExecContext(ctx, "UPDATE categories SET name = ?, icon = ?, color = ? WHERE id = ?", next.Name, next.Icon, next.Color, id); err != nil {
			return fmt.Errorf("update category: %w", err)
		}
		out, err = getCategory(ctx, q, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("update category %d: %w", id, err)
	}
	return out, nil
}

// MoveCategory puts a category at index (0-based) in the list; the others close ranks around it.
func (c *Core) MoveCategory(ctx context.Context, id int64, index int) error {
	err := c.write(ctx, func(q store.Querier) error {
		all, err := listCategories(ctx, q)
		if err != nil {
			return err
		}
		from := -1
		for i, cat := range all {
			if cat.ID == id {
				from = i
			}
		}
		if from < 0 {
			return notFound("category_id", "category", id)
		}
		if index < 0 || index >= len(all) {
			return invalid("index", "must be between 0 and %d", len(all)-1)
		}
		moved := all[from]
		order := append(append([]Category{}, all[:from]...), all[from+1:]...)
		order = append(order[:index], append([]Category{moved}, order[index:]...)...)
		return renumberCategories(ctx, q, order)
	})
	if err != nil {
		return fmt.Errorf("move category %d: %w", id, err)
	}
	return nil
}

// renumberCategories writes positions 1..n in the given order, touching only the rows that change.
func renumberCategories(ctx context.Context, q store.Querier, order []Category) error {
	for i, cat := range order {
		if cat.Position == i+1 {
			continue
		}
		if _, err := q.ExecContext(ctx, "UPDATE categories SET position = ? WHERE id = ?", i+1, cat.ID); err != nil {
			return fmt.Errorf("renumber categories: %w", err)
		}
	}
	return nil
}

// DeletedCategory is what DeleteCategory removed: enough to put it back.
type DeletedCategory struct {
	Category Category
	// TaskIDs are the tasks that were sorted under it and are unsorted now.
	TaskIDs []int64
}

// DeleteCategory removes a user's category; its tasks become unsorted. Work and Life cannot go.
func (c *Core) DeleteCategory(ctx context.Context, id int64) (DeletedCategory, error) {
	var out DeletedCategory
	err := c.write(ctx, func(q store.Querier) error {
		cur, err := getCategory(ctx, q, id)
		if err != nil {
			return err
		}
		if cur == nil {
			return notFound("category_id", "category", id)
		}
		if cur.Builtin != "" {
			return invalid("category_id", "Work and Life cannot be deleted")
		}
		rows, err := q.QueryContext(ctx, "SELECT id FROM tasks WHERE category_id = ? ORDER BY id", id)
		if err != nil {
			return fmt.Errorf("list tasks of category: %w", err)
		}
		ids, err := collect(rows, func(s scanner) (int64, error) {
			var n int64
			return n, s.Scan(&n)
		})
		if err != nil {
			return err
		}
		// ON DELETE SET NULL unsorts the tasks.
		if _, err := q.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id); err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		rest, err := listCategories(ctx, q)
		if err != nil {
			return err
		}
		out = DeletedCategory{Category: *cur, TaskIDs: ids}
		return renumberCategories(ctx, q, rest)
	})
	if err != nil {
		return DeletedCategory{}, fmt.Errorf("delete category %d: %w", id, err)
	}
	return out, nil
}

// RestoreCategory is the Undo of DeleteCategory: the category comes back with its id at its old
// place in the list (or at the end when the list got shorter), and goes back to the tasks that
// were sorted under it and are still unsorted. A name or an id that is taken meanwhile is a conflict.
func (c *Core) RestoreCategory(ctx context.Context, d DeletedCategory) (*Category, error) {
	var out *Category
	cat := d.Category
	err := c.write(ctx, func(q store.Querier) error {
		if cur, err := getCategory(ctx, q, cat.ID); err != nil {
			return err
		} else if cur != nil {
			return conflict("category_id", "category %d already exists", cat.ID)
		}
		all, err := listCategories(ctx, q)
		if err != nil {
			return err
		}
		if len(all) >= MaxCategories {
			return invalid("category", "at most %d categories; delete one first", MaxCategories)
		}
		if err := checkCategoryName(all, 0, cat.Name); err != nil {
			return err
		}
		at := min(max(cat.Position, 1), len(all)+1)
		if _, err := q.ExecContext(ctx, "INSERT INTO categories (id, builtin, name, icon, color, position, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			cat.ID, nullIfEmpty(cat.Builtin), cat.Name, cat.Icon, cat.Color, at, cat.CreatedAt); err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
		// Open a gap at the restored position: everything from there on moves down one.
		if _, err := q.ExecContext(ctx, "UPDATE categories SET position = position + 1 WHERE id != ? AND position >= ?", cat.ID, at); err != nil {
			return fmt.Errorf("renumber categories: %w", err)
		}
		for _, id := range d.TaskIDs {
			if _, err := q.ExecContext(ctx, "UPDATE tasks SET category_id = ? WHERE id = ? AND category_id IS NULL", cat.ID, id); err != nil {
				return fmt.Errorf("sort task %d back: %w", id, err)
			}
		}
		out, err = getCategory(ctx, q, cat.ID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("restore category %d: %w", cat.ID, err)
	}
	return out, nil
}

// ResolveCategory finds a category by name, ignoring case. Work and Life also answer to their keys
// and their Chinese names, so "work" and "工作" mean the same as "Work".
func (c *Core) ResolveCategory(ctx context.Context, name string) (int64, error) {
	all, err := listCategories(ctx, c.db)
	if err != nil {
		return 0, err
	}
	return resolveCategory(all, name)
}

func resolveCategory(all []Category, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, invalid("category", "must not be empty")
	}
	alias := builtinAliases[name]
	names := make([]string, len(all))
	for i, cat := range all {
		names[i] = cat.Name
		if strings.EqualFold(cat.Name, name) || cat.Builtin != "" && (strings.EqualFold(cat.Builtin, name) || cat.Builtin == alias) {
			return cat.ID, nil
		}
	}
	return 0, invalid("category", "no category named %q; the categories are %s", name, strings.Join(names, ", "))
}

// CheckCategoryName trims a category name and checks it against the categories there are: self is
// the one being renamed, 0 for a new category. It is what CreateCategory and UpdateCategory ask, for
// a store that keeps its categories elsewhere.
func CheckCategoryName(all []Category, self int64, name string) (string, error) {
	name, err := cleanCategoryName(name)
	if err != nil {
		return "", err
	}
	return name, checkCategoryName(all, self, name)
}

func cleanCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", invalid("name", "must not be empty")
	case utf8.RuneCountInString(name) > MaxCategoryName:
		return "", invalid("name", "must be at most %d characters", MaxCategoryName)
	}
	return name, nil
}

// checkCategoryIcon wants the grid of pixmap.Grid with at least one pixel set.
func checkCategoryIcon(icon string) error {
	p, err := pixmap.ParseGrid(icon)
	if err != nil {
		return invalid("icon", "must be %d characters of ., # and the palette letters a to l", pixmap.GridLen)
	}
	if p.Empty() {
		return invalid("icon", "must have at least one pixel")
	}
	return nil
}

func checkCategoryColor(color string) error {
	for _, k := range CategoryColors {
		if k == color {
			return nil
		}
	}
	return invalid("color", "must be one of %s", strings.Join(CategoryColors, ", "))
}

// checkCategoryName refuses a name another category (not self) already has, in any case, or that
// is an alias of Work or Life.
func checkCategoryName(all []Category, self int64, name string) error {
	if _, ok := builtinAliases[name]; ok {
		return conflict("name", "a category named %q already exists", name)
	}
	for _, cat := range all {
		if cat.ID != self && (strings.EqualFold(cat.Name, name) || cat.Builtin != "" && strings.EqualFold(cat.Builtin, name)) {
			return conflict("name", "a category named %q already exists", name)
		}
	}
	return nil
}
