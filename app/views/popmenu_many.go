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
		same := func(p data.Priority) bool {
			return len(tasks) > 0 && len(only(tasks, func(k data.Task) bool { return k.Priority != p })) == 0
		}
		m.priorityStripOf(c, t, same, true, func(p data.Priority) {
			m.closeMenu()
			m.bulkPriority(c, ids, p)
		})
	case pageMove:
		m.movePageOf(c, t, func() []popSpec {
			return m.projectChoices(
				func(project int) bool {
					return len(tasks) > 0 && len(only(tasks, func(k data.Task) bool { return !inProject(k, project) })) == 0
				},
				func(project int) {
					m.closeMenu()
					m.bulkMove(c, ids, project)
				})
		})
	default:
		m.manyMainPage(c, t, ids)
	}
}

// inProject says whether the task is in the project, or in the Inbox for project 0.
func inProject(t data.Task, project int) bool {
	return (t.ProjectID == nil && project == 0) || (t.ProjectID != nil && *t.ProjectID == project)
}

// manyMainPage is the whole menu of the selection.
func (m *Main) manyMainPage(c *ui.Context, t *theme.Tokens, ids []int) {
	pm := &m.pop
	theme.Text(c, theme.Meta, t.TextTertiary, m.say.MenuManyHeader(len(ids))).Padding(6, 8, 4)
	m.popRow(c, t, popSpec{key: "many-done", icon: theme.IconCheck, label: m.say.BulkMarkDone(), hint: "⌘↵", run: func() {
		m.closeMenu()
		m.bulkSetStatus(c, ids, data.Done)
	}})
	m.popRow(c, t, popSpec{key: "many-cancel", icon: theme.IconClose, label: m.say.BulkCancel(), run: func() {
		m.closeMenu()
		m.bulkSetStatus(c, ids, data.Cancelled)
	}})
	menuRule(c, t)
	m.popRow(c, t, popSpec{key: "many-move", icon: theme.IconMove, label: m.say.MenuMoveTo(), hint: "M", run: func() {
		pm.page, pm.back, pm.hl, pm.filter = pageMove, true, 0, ""
	}})
	m.popRow(c, t, popSpec{key: "many-priority", icon: theme.IconFlag, label: m.say.PropPriority(), hint: "P", run: func() {
		pm.page, pm.back, pm.hl = pagePriority, true, -1
	}})
	m.popRow(c, t, popSpec{key: "many-hand", icon: theme.IconAgent, label: m.say.BulkHandOff(), hint: "⌘⇧A", run: func() {
		m.closeMenu()
		m.handOffMany(c, ids)
	}})
	menuRule(c, t)
	m.popRow(c, t, popSpec{key: "many-delete", icon: theme.IconTrash, label: m.say.BulkDeleteMenu(len(ids)), hint: "⌘⌫", danger: true, run: func() {
		m.closeMenu()
		m.askBulkDelete(ids)
	}})
}

// manyLetters are the keys of the selection's whole menu.
func (m *Main) manyLetters(c *ui.Context, panel ui.Element) {
	pm := &m.pop
	ids := pm.many
	switch {
	case panel.OverlayShortcut(0, ui.KeyP):
		pm.page, pm.back, pm.hl = pagePriority, true, -1
	case panel.OverlayShortcut(0, ui.KeyM):
		pm.page, pm.back, pm.hl, pm.filter = pageMove, true, 0, ""
	case panel.OverlayShortcut(ui.Cmd, ui.KeyEnter):
		m.closeMenu()
		m.bulkSetStatus(c, ids, data.Done)
	case panel.OverlayShortcut(ui.Cmd|ui.Shift, ui.KeyA):
		m.closeMenu()
		m.handOffMany(c, ids)
	case panel.OverlayShortcut(ui.Cmd, ui.KeyBackspace):
		m.closeMenu()
		m.askBulkDelete(ids)
	}
}
