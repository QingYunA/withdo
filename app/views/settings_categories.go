package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// deletedNoticeFor is how long the line about a deleted category, with its Undo, stays in the group.
const deletedNoticeFor = 6 * time.Second

// categoriesState is Settings > General > Categories: the sheet that makes and edits a category (the
// one the task menu opens in the main window, hosted by a Main of its own), the category whose delete
// waits for a second click, and the one just deleted with the time it was.
type categoriesState struct {
	sheet     *Main
	confirm   int // the category whose delete asks first, 0 for none
	deleted   data.DeletedCategory
	deletedAt time.Time
}

// categoriesGroup lists the categories in their order, each with its icon in its color, its name and
// what can be done to it: move up or down, edit (the sheet: name, icon, color) and, but for Work and
// Life, delete. The last row makes a new one. This is where categories are managed, now that the
// sidebar has no rows for them.
func (s *Settings) categoriesGroup(c *ui.Context, t *theme.Tokens) {
	cats := s.app.snap.categories
	if !s.cats.deletedAt.IsZero() && s.app.Now().Sub(s.cats.deletedAt) > deletedNoticeFor {
		s.cats.deletedAt = time.Time{}
	}
	groupLabel(c, t, s.say.SettingsCategories())
	group(c, t, "categories", func() {
		for i, cat := range cats {
			s.categoryRow(c, t, cat, i, len(cats))
		}
		if !s.cats.deletedAt.IsZero() {
			s.deletedLine(c, t, len(cats) == 0)
		}
		settingsRow(c, t, "category-new", "", "", len(cats) == 0 && s.cats.deletedAt.IsZero(), func() {
			if label := s.say.CategoryNew(); settingsLink(c, t, "category-new-link", label, label).Clicked() {
				s.categorySheet().openNewCategory(0)
			}
		})
	})
}

// categorySheet is the Main that hosts the sheet, made when it is first needed.
func (s *Settings) categorySheet() *Main {
	if s.cats.sheet == nil {
		s.cats.sheet = NewMain(s.app)
	}
	return s.cats.sheet
}

// categorySheetView draws the category sheet over the pane while it is open.
func (s *Settings) categorySheetView(c *ui.Context, t *theme.Tokens) {
	if s.cats.sheet == nil || !s.cats.sheet.manage.category.open {
		return
	}
	s.cats.sheet.say, s.cats.sheet.clock = s.say, s.clock
	s.cats.sheet.categorySheet(c, t)
}

// categoryRow is one category: its icon and name, and at the right the arrows, Edit and Delete.
func (s *Settings) categoryRow(c *ui.Context, t *theme.Tokens, cat data.Category, at, n int) {
	name := categoryTitle(s.say, cat)
	key := "category-" + itoa(cat.ID)
	lead := func() { categoryGlyph(c, cat, theme.IconRow) }
	if s.cats.confirm == cat.ID {
		settingsRowLead(c, t, key, s.say.SettingsCategoryDeleteAsk(name), "", at == 0, func() {
			ui.Row(c.Key(key + "-ask")).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				if label := s.say.ManageCancel(); settingsLink(c, t, key+"-cancel", label, label).Clicked() {
					s.cats.confirm = 0
				}
				if label := s.say.ManageDeleteConfirm(); settingsLink(c, t, key+"-delete", label, s.say.A11yCategoryDelete(name)).Clicked() {
					s.deleteCategory(cat.ID)
				}
			})
		}, lead)
		return
	}
	settingsRowLead(c, t, key, name, "", at == 0, func() {
		ui.Row(c.Key(key + "-tools")).Shrink(0).Gap(2).AlignItems(ui.Center).Children(func() {
			if at > 0 && s.categoryTool(c, t, key+"-up", theme.IconSend, s.say.A11yCategoryUp(name)) {
				_ = s.app.MoveCategory(cat.ID, at-1)
			}
			if at < n-1 && s.categoryTool(c, t, key+"-down", theme.IconArrowDown, s.say.A11yCategoryDown(name)) {
				_ = s.app.MoveCategory(cat.ID, at+1)
			}
			if label := s.say.SettingsCategoryEdit(); settingsLink(c, t, key+"-edit", label, s.say.A11yCategoryEdit(name)).Clicked() {
				s.categorySheet().openEditCategory(cat)
			}
			if cat.Builtin == "" && s.categoryTool(c, t, key+"-trash", theme.IconTrash, s.say.A11yCategoryDelete(name)) {
				s.cats.confirm = cat.ID
			}
		})
	}, lead)
}

// categoryTool is a 24 pt icon button of a category row; it reports a click.
func (s *Settings) categoryTool(c *ui.Context, t *theme.Tokens, key string, icon *ui.SVG, label string) bool {
	btn := ui.ButtonBase(c.Key(key)).Size(theme.HitTarget, theme.HitTarget).Shrink(0).Center().Radius(theme.RadiusControl).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	ink := t.TextTertiary
	if btn.Hovered() {
		btn.Background(t.BgHover)
		ink = t.TextSecondary
	}
	btn.Children(func() { ui.Icon(c, icon).Size(theme.IconSmall, theme.IconSmall).TextColor(ink) })
	return btn.Clicked()
}

// deleteCategory deletes at once, its tasks become unsorted, and the group says so with an Undo that
// puts the category back with its icon, its color and its place.
func (s *Settings) deleteCategory(id int) {
	s.cats.confirm = 0
	d, err := s.app.DeleteCategory(id)
	if err != nil {
		return
	}
	s.cats.deleted, s.cats.deletedAt = d, s.app.Now()
}

// deletedLine is the row that says which category was deleted, with Undo.
func (s *Settings) deletedLine(c *ui.Context, t *theme.Tokens, first bool) {
	d := s.cats.deleted
	text := s.say.CategoryDeletedEmpty(d.Category.Name)
	if d.Tasks > 0 {
		text = s.say.CategoryDeleted(d.Category.Name, d.Tasks)
	}
	settingsRow(c, t, "category-deleted", text, "", first, func() {
		if label := s.say.ConvoUndo(); settingsLink(c, t, "category-undo", label, label).Clicked() {
			_ = s.app.RestoreCategory(d)
			s.cats.deletedAt = time.Time{}
		}
	})
}
