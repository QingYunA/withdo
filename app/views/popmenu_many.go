package views

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The task menu of the rows selected together: the same popover, with a header that says how many
// tasks it acts on and the actions of the bulk bar, plus Cancel. Its Priority and Move to pages are
// the single task's, applied to all of them.

// openManyMenu shows the menu for the rows ids, hanging from the point (x, y) of the window.
func (m *Main) openManyMenu(ids []int, page menuPage, x, y float32, back bool) {
	m.pop = popMenu{
		open: true, task: ids[0], many: slices.Clone(ids), page: page, back: back, x: x, y: y, hl: -1,
		frame: m.pop.frame, at: m.pop.frame,
	}
	if page == pageMove {
		m.pop.hl = 0
	}
}

// openManyMenuAbove opens the menu over a button of the bulk bar, which sits at the bottom of the
// window: it hangs from the button's top, so the menu rises above the bar.
func (m *Main) openManyMenuAbove(el ui.Element, ids []int, page menuPage) {
	b := el.Bounds()
	m.openManyMenu(ids, page, b.X, b.Y-6, false)
}

// rowMenu is the right-click on a row: on one of the rows selected together the menu is for all of
// them; on any other row it is for that row alone, and the selection stays as it is.
func (m *Main) rowMenu(row ui.Element, id int) {
	if m.multi.has(id) && m.multi.count() >= 2 {
		m.openManyMenuAtPointer(row, m.selectedRows())
		return
	}
	m.openMenuAtPointer(row, id)
}

// openManyMenuAtPointer is the right-click on a selected row.
func (m *Main) openManyMenuAtPointer(el ui.Element, ids []int) {
	px, py, _ := el.PointerPosition()
	x, y := el.LocalToWindow(px, py)
	m.openManyMenu(ids, pageMain, x, y, false)
}

// manyPage builds the page the menu is on.
func (m *Main) manyPage(c *ui.Context, t *theme.Tokens) {
	pm := &m.pop
	ids, tasks := pm.many, m.tasksOf(pm.many)
	switch pm.page {
	case pagePriority:
		m.stripLabel(c, t, "priority", m.say.PropPriority(), m.say.MenuManyHeader(len(ids)))
		m.priorityStripOf(c, t, func(p data.Priority) bool { return allPriority(tasks, p) }, true, func(p data.Priority) {
			m.closeMenu()
			m.bulkPriority(c, ids, p)
		})
	case pageMove:
		m.movePageOf(c, t, finder{placeholder: m.say.MenuFindProject(), noMatch: m.say.MenuNoProjectMatch()}, func() []popSpec {
			return m.projectChoices(
				func(project int) bool { return allIn(tasks, project) },
				func(project int) {
					m.closeMenu()
					m.bulkMove(c, ids, project)
				})
		})
	default:
		m.manyMainPage(c, t, ids)
	}
}

// openPage opens a piece of the selection's whole menu: Move to starts on its first choice, the
// strips on none.
func (pm *popMenu) openPage(page menuPage) {
	pm.page, pm.back, pm.filter, pm.hl = page, true, "", -1
	if page == pageMove {
		pm.hl = 0
	}
}

// manyMainPage is the whole menu of the selection: the actions in their groups.
func (m *Main) manyMainPage(c *ui.Context, t *theme.Tokens, ids []int) {
	theme.Text(c, theme.Meta, t.TextTertiary, m.say.MenuManyHeader(len(ids))).Padding(6, 8, 4)
	group := 0
	for _, a := range actionsFor(m.tasksOf(ids)) {
		if a.Group != group {
			menuRule(c, t)
			group = a.Group
		}
		m.popRow(c, t, actionSpec(a, "many-", m.menuLabel(a, len(ids)), func() { m.chooseAction(c, a, ids) }))
	}
}

// chooseAction is an action chosen in the selection's menu, by its line or its key: a piece of the
// menu opens, or the action is done and the menu closes.
func (m *Main) chooseAction(c *ui.Context, a taskAction, ids []int) {
	if a.Page != pageMain {
		m.pop.openPage(a.Page)
		return
	}
	m.closeMenu()
	m.runAction(c, a.Kind, ids)
}

// manyLetters are the keys of the selection's whole menu.
func (m *Main) manyLetters(c *ui.Context, panel ui.Element) {
	acts := actionsFor(m.tasksOf(m.pop.many))
	if a, ok := pressedAction(acts, func(b binding) bool { return b.inMenu(panel) }); ok {
		m.chooseAction(c, a, m.pop.many)
	}
}

// manyTaskKeys are the keys of the rows selected together, in place of a single row's: P and M open
// the menu on that piece, under the keyboard row, and ⌘⌫ asks before deleting them all.
func (m *Main) manyTaskKeys(c *ui.Context) {
	ids := m.selectedRows()
	r := m.selRect
	a, ok := pressedAction(actionsFor(m.tasksOf(ids)), func(b binding) bool { return b.down(c) }, actMove, actPriority, actDelete)
	switch {
	case !ok:
	case a.Page != pageMain:
		m.openManyMenu(ids, a.Page, r.X+24, r.Y+r.H, false)
	default:
		m.runAction(c, a.Kind, ids)
	}
}
