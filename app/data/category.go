package data

import (
	"errors"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
)

// Category is a label the user sorts tasks under. Work and Life are built in (Builtin says which)
// and always there; they keep their names, which the app words in the user's language, and cannot
// be deleted. The rest are the user's own.
type Category struct {
	ID int
	// Builtin is BuiltinWork or BuiltinLife for the two built-in categories, "" for the user's own.
	Builtin string
	// Name is what the user called it; a built-in category is stored as Work or Life.
	Name string
	// Icon is the 12 x 12 picture drawn beside the name, in the category's color.
	Icon pixmap.Bitmap
	// Color is one of domain.CategoryColors, which theme.CategoryColor turns into a color.
	Color string
	// Position is the place in the list, 1-based and contiguous.
	Position int
}

// The built-in categories.
const (
	BuiltinWork = domain.BuiltinWork
	BuiltinLife = domain.BuiltinLife
)

// Limits of a category's name and of the list.
const (
	MaxCategories   = domain.MaxCategories
	MaxCategoryName = domain.MaxCategoryName
)

// CategoryOf is the id of the category a task counts under: its own when it has one, else workID
// (the built-in Work) when its project has a repository (hasRepo says), else 0. An explicit
// category always wins.
func CategoryOf(t Task, hasRepo func(*int) bool, workID int) int {
	switch {
	case t.CategoryID != 0:
		return t.CategoryID
	case hasRepo != nil && t.ProjectID != nil && hasRepo(t.ProjectID):
		return workID
	}
	return 0
}

// NewCategory is what the category sheet sends. An empty Color takes the one the name hashes to.
type NewCategory struct {
	Name  string
	Icon  pixmap.Bitmap
	Color string
}

// CategoryPatch changes a category; a nil field is left alone. A built-in category keeps its name.
type CategoryPatch struct {
	Name  *string
	Icon  *pixmap.Bitmap
	Color *string
}

// DeletedCategory is what DeleteCategory removed, so RestoreCategory can put it back: the category
// as it was, how many tasks it unsorted, and which they were.
type DeletedCategory struct {
	Category Category
	Tasks    int
	raw      any // the store's own form
}

// Problems with a category's input, for the views to word; ErrNameEmpty is the empty name.
var (
	ErrCategoryNameLong    = errors.New("category name must be at most 32 characters")
	ErrCategoryNameTaken   = errors.New("a category with that name already exists")
	ErrCategoryBuiltinName = errors.New("Work and Life keep their names")
	ErrCategoryBuiltin     = errors.New("Work and Life cannot be deleted")
	ErrTooManyCategories   = errors.New("there can be at most 24 categories")
	ErrIconEmpty           = errors.New("the icon needs at least one pixel")
	ErrCategoryColor       = errors.New("unknown category color")
)

// categoryProblem words a domain failure of a category operation as the Manager promises; other
// errors pass through.
func categoryProblem(err error) error {
	var fe *domain.FieldError
	if err == nil || !errors.As(err, &fe) {
		return err
	}
	switch {
	case fe.Field == "name" && fe.Conflict:
		return ErrCategoryNameTaken
	case fe.Field == "name" && strings.Contains(fe.Msg, "not be empty"):
		return ErrNameEmpty
	case fe.Field == "name" && strings.Contains(fe.Msg, "at most"):
		return ErrCategoryNameLong
	case fe.Field == "name":
		return ErrCategoryBuiltinName
	case fe.Field == "category" && !fe.NotFound:
		return ErrTooManyCategories
	case fe.Field == "category_id" && !fe.NotFound && !fe.Conflict:
		return ErrCategoryBuiltin
	case fe.Field == "icon":
		return ErrIconEmpty
	case fe.Field == "color":
		return ErrCategoryColor
	}
	return err
}
