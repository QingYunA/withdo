package views

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The context menus of the categories in the sidebar. Mine's has only New category…; a category's
// also edits, moves and, unless it is Work or Life, deletes it.

// mineMenu is the context menu of Mine and of Unsorted.
func (m *Main) mineMenu(mn *ui.Menu) {
	if mn.Item(m.say.CategoryNew()).Chosen() {
		m.openNewCategory(0)
	}
}

// categoryMenu is the context menu of a category's row.
func (m *Main) categoryMenu(c *ui.Context, mn *ui.Menu, cat data.Category) {
	m.mineMenu(mn)
	if mn.Item(m.say.CategoryEdit()).Chosen() {
		m.openEditCategory(cat)
	}
	at := slices.IndexFunc(m.app.snap.categories, func(x data.Category) bool { return x.ID == cat.ID })
	if at > 0 && mn.Item(m.say.CategoryMoveUp()).Chosen() {
		_ = m.app.MoveCategory(cat.ID, at-1)
	}
	if at < len(m.app.snap.categories)-1 && mn.Item(m.say.CategoryMoveDown()).Chosen() {
		_ = m.app.MoveCategory(cat.ID, at+1)
	}
	if cat.Builtin != "" {
		return
	}
	mn.Separator()
	if mn.Item(m.say.CategoryDelete()).Chosen() {
		m.deleteCategory(c, cat)
	}
}

// deleteCategory deletes at once: its tasks become unsorted, and the toast offers Undo, which puts
// the category back with its icon, its color and its place.
func (m *Main) deleteCategory(c *ui.Context, cat data.Category) {
	d, err := m.app.DeleteCategory(cat.ID)
	if err != nil {
		return
	}
	title := m.say.CategoryDeletedEmpty(cat.Name)
	if d.Tasks > 0 {
		title = m.say.CategoryDeleted(cat.Name, d.Tasks)
	}
	m.offerUndo(c, undoEntry{title: title, undo: func() { _ = m.app.RestoreCategory(d) }})
}
