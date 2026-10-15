package views

import (
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// confirmation is the question before a project is deleted with its tasks. A task is deleted at
// once, with an Undo toast, and needs none.
type confirmation struct {
	open bool
	id   int
}

func (m *Main) askDeleteProject(id int) {
	m.manage.confirm = confirmation{open: true, id: id}
}

// sheetOpen reports whether a sheet or the confirmation floats over the window, taking the keys.
func (m *Main) sheetOpen() bool {
	return m.manage.project.open || m.manage.milestone.open || m.manage.confirm.open || m.pop.open
}

// confirmSheet asks before a delete. Cancel has the focus, so Enter does not delete; Esc cancels.
func (m *Main) confirmSheet(c *ui.Context, t *theme.Tokens) {
	cf := &m.manage.confirm
	if !cf.open {
		return
	}
	title, body, ok := m.confirmWords()
	if !ok {
		cf.open = false // what it was about is gone
		return
	}
	sheet(c, t, &cf.open, func() {
		theme.Paragraph(c, theme.BodyStrong, t.Text, title, 0)
		theme.Paragraph(c, theme.Body, t.TextSecondary, body, 0)
		cancelled, chosen := sheetButtons(c, m.say.ManageCancel(), m.say.ManageDeleteConfirm(), true, true)
		if cancelled {
			cf.open = false
		}
		if chosen {
			cf.open = false
			m.deleteConfirmed(*cf)
		}
	})
}

// confirmWords are the title and body of the question, and false when its subject no longer exists.
func (m *Main) confirmWords() (title, body string, ok bool) {
	cf := m.manage.confirm
	p, found := m.app.Project(cf.id)
	if n := m.app.taskCount(cf.id); n > 0 {
		return m.say.ManageDeleteProjectTitle(p.Name), m.say.ManageDeleteProjectBody(n), found
	}
	return m.say.ManageDeleteProjectTitle(p.Name), m.say.ManageDeleteProjectEmpty(), found
}

// deleteConfirmed does what the question asked, and leaves the screen the deleted project was on.
func (m *Main) deleteConfirmed(cf confirmation) {
	if m.app.DeleteProject(cf.id) == nil {
		m.leaveProject(cf.id)
	}
}

// leaveProject goes to Now when the project on screen is no longer in the sidebar.
func (m *Main) leaveProject(id int) {
	if m.nav.screen == screenProject && m.nav.projectID == id {
		m.goTo(nav{})
	}
	delete(m.milestone, id)
	delete(m.board, id)
}
