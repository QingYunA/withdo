package views

import (
	"errors"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/pixmap"
	"github.com/egoist/mygo/ui"
)

// categorySheet is the New category and Edit category sheet: a name, an icon and a color.
type categorySheet struct {
	open    bool
	id      int // the category being edited, 0 for a new one
	task    int // the task a new category is given to, 0 for none
	name    string
	icon    pixmap.Picture
	color   string // the key of one of data.CategoryColors
	picked  bool   // the color was chosen, so it no longer follows the name
	query   string // what is typed in Search icons
	problem error  // what the last Create or Save found wrong with the name
}

// openNewCategory opens the sheet for a new category, on the first curated icon and the color its
// name hashes to. A category made from the task menu is given to task; from the sidebar (task 0)
// its page opens.
func (m *Main) openNewCategory(task int) {
	m.manage.category = categorySheet{open: true, task: task, icon: theme.CategoryIconGroups[0].Icons[0].Bitmap.Picture()}
}

// openEditCategory opens the sheet on a category as it is.
func (m *Main) openEditCategory(cat data.Category) {
	m.manage.category = categorySheet{open: true, id: cat.ID, name: cat.Name, icon: cat.Icon, color: cat.Color, picked: true}
}

// editsBuiltin reports whether the sheet edits Work or Life, whose names are fixed.
func (m *Main) editsBuiltin() bool {
	cat, ok := m.app.Category(m.manage.category.id)
	return ok && cat.Builtin != ""
}

// categorySheet draws the sheet while it is open.
func (m *Main) categorySheet(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.category
	if !s.open {
		return
	}
	if !s.picked {
		s.color = data.DefaultCategoryColor(strings.TrimSpace(s.name))
	}
	title, primary := m.say.CategorySheetNew(), m.say.ManageCreate()
	if s.id != 0 {
		title, primary = m.say.CategorySheetEdit(), m.say.ManageSave()
	}
	sheet(c, t, &s.open, func() {
		theme.Text(c, theme.Title, t.Text, title)
		labeled(c, t, m.say.ManageFieldName(), func() { m.categoryNameField(c, t, s) })
		labeled(c, t, m.say.CategoryFieldIcon(), func() { m.categoryPreview(c, t, s) })
		labeled(c, t, m.say.CategoryFieldColor(), func() { m.categorySwatches(c, t, s) })
		m.categoryPicker(c, t, s)
		cancelled, chosen := sheetButtons(c, m.say.ManageCancel(), primary, m.editsBuiltin() || strings.TrimSpace(s.name) != "", false)
		if cancelled {
			s.open = false
		}
		if chosen {
			m.saveCategory()
		}
	})
}

// categoryNameField is the name, with a line under it: what is wrong with it, or why Work and Life
// cannot be renamed, which show their names in the user's language and take no text.
func (m *Main) categoryNameField(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	if m.editsBuiltin() {
		cat, _ := m.app.Category(s.id)
		ui.Row(c.Key("category-name-well")).Height(30).Shrink(0).Padding(0, 10).Radius(theme.RadiusControl).Background(t.BgSubtle).
			AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.Body, t.TextSecondary, categoryTitle(m.say, cat))
		})
		theme.Paragraph(c, theme.Meta, t.TextSecondary, m.say.CategoryBuiltinKeepsName(), 0)
		return
	}
	wellInput(c, t, "category-name", &s.name, m.say.ManageFieldName(), "", theme.Body, true, m.categorySheetKey)
	if line := m.categoryProblemLine(); line != "" {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, line, 0)
	}
}

// categoryProblemLine words what the last Create or Save found wrong with the name.
func (m *Main) categoryProblemLine() string {
	switch err := m.manage.category.problem; {
	case errors.Is(err, data.ErrCategoryNameLong):
		return m.say.CategoryErrLong(data.MaxCategoryName)
	case errors.Is(err, data.ErrCategoryNameTaken):
		return m.say.CategoryErrTaken()
	case errors.Is(err, data.ErrTooManyCategories):
		return m.say.CategoryErrTooMany(data.MaxCategories)
	}
	return ""
}

// categorySheetKey is the keyboard of the name: Enter creates or saves.
func (m *Main) categorySheetKey(k ui.TextKey) bool {
	if k.Mods != 0 || k.Key != ui.KeyEnter {
		return false
	}
	m.saveCategory()
	return true
}

// categoryPreview is the icon in the chosen color at 36, 18 and 12 pt: the sizes of the pages, the
// rows and the sidebar.
func (m *Main) categoryPreview(c *ui.Context, t *theme.Tokens, s *categorySheet) {
	shown := data.Category{Icon: s.icon, Color: s.color}
	ui.Row(c).Shrink(0).AlignSelf(ui.Start).Padding(10, 16).Gap(14).Radius(theme.RadiusPanel).Background(t.BgSubtle).AlignItems(ui.Center).Children(func() {
		for _, size := range []float32{theme.IconLarge, theme.IconRow, theme.IconSmall} {
			categoryGlyph(c, shown, size)
		}
	})
}

// saveCategory makes the category or saves the edit. A problem with the name stays under the field;
// any other failure closes the sheet and shows the banner.
func (m *Main) saveCategory() {
	s := &m.manage.category
	name := strings.TrimSpace(s.name)
	if name == "" && !m.editsBuiltin() {
		return
	}
	var made data.Category
	var err error
	if s.id == 0 {
		made, err = m.app.CreateCategory(data.NewCategory{Name: name, Icon: s.icon, Color: s.color})
	} else {
		err = m.app.UpdateCategory(s.id, m.categoryPatch(name))
	}
	switch {
	case errors.Is(err, data.ErrCategoryNameLong), errors.Is(err, data.ErrCategoryNameTaken), errors.Is(err, data.ErrTooManyCategories):
		s.problem = err
	case err != nil:
		m.app.snap.err = err
		s.open = false
	default:
		s.open = false
		if s.id == 0 {
			m.afterNewCategory(made, s.task)
		}
	}
}

// categoryPatch is what Save changes: the icon and the color, and the name unless it is Work's or
// Life's.
func (m *Main) categoryPatch(name string) data.CategoryPatch {
	s := m.manage.category
	patch := data.CategoryPatch{Icon: &s.icon, Color: &s.color}
	if !m.editsBuiltin() {
		patch.Name = &name
	}
	return patch
}

// afterNewCategory gives the new category to the task it was made for, or else opens its page.
func (m *Main) afterNewCategory(cat data.Category, task int) {
	if task != 0 {
		_ = m.app.Update(task, data.Patch{Category: data.IntPtr(cat.ID)})
		return
	}
	m.goTo(nav{screen: screenMine, cat: cat.ID})
}
