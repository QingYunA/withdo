package views

import (
	"errors"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// projectSheet is the New project sheet: a name and an optional GitHub repository, under an optional
// parent project (which is how a sub-project is made, from the parent's menu).
type projectSheet struct {
	open       bool
	name, repo string
	repoOpen   bool  // the repository field is unfolded; it is folded until the user asks for it
	parent     int   // the project the new one hangs under; 0 for a top-level project
	problem    error // what the last Create found wrong with the repository
}

// openNewProject opens the sheet; parent 0 makes a top-level project.
func (m *Main) openNewProject(parent int) {
	m.manage.project = projectSheet{open: true, parent: parent}
}

// newProjectSheet draws the sheet while it is open.
func (m *Main) newProjectSheet(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.project
	if !s.open {
		return
	}
	sheet(c, t, &s.open, func() {
		title := m.say.ManageNewProjectTip()
		if s.parent != 0 {
			title = m.say.ManageNewSubprojectTitle()
		}
		theme.Text(c, theme.Title, t.Text, title)
		if parent, ok := m.app.Project(s.parent); ok {
			theme.Paragraph(c, theme.Meta, t.TextSecondary, m.say.ManageSubprojectUnder(parent.Name), 0)
		}
		labeled(c, t, m.say.ManageFieldName(), func() {
			wellInput(c, t, "project-name", &s.name, m.say.ManageFieldName(), "", theme.Body, true, m.projectSheetKey)
		})
		if s.repoOpen {
			labeled(c, t, m.say.ManageFieldRepo(), func() {
				wellInput(c, t, "project-repo", &s.repo, m.say.ManageFieldRepo(), m.say.ManageRepoPlaceholder(), theme.Mono, false, m.projectSheetKey)
				theme.Paragraph(c, theme.Meta, t.TextSecondary, m.repoLine(), 0)
			})
		} else {
			m.repoFold(c, t)
		}
		cancelled, create := sheetButtons(c, m.say.ManageCancel(), m.say.ManageCreate(), strings.TrimSpace(s.name) != "", false)
		if cancelled {
			s.open = false
		}
		if create {
			m.createProject()
		}
	})
}

// repoFold is the folded repository field: a chevron and the field's label; a click unfolds it.
func (m *Main) repoFold(c *ui.Context, t *theme.Tokens) {
	row := ui.Row(c.Key("project-repo-fold")).Padding(3, 4).Gap(4).Radius(7).AlignItems(ui.Center).Shrink(0).
		Cursor(ui.CursorPointer).Label(m.say.ManageFieldRepo())
	ink := t.TextTertiary
	if row.Hovered() {
		row.Background(t.BgHover)
		ink = t.TextSecondary
	}
	row.Children(func() {
		ui.Icon(c, theme.IconForward).Size(theme.IconSmall, theme.IconSmall).TextColor(ink).Shrink(0)
		theme.Text(c, theme.Meta, ink, m.say.ManageFieldRepo())
	})
	if row.Clicked() {
		m.manage.project.repoOpen = true
	}
}

// repoLine is the line under the repository field: what is wrong with it, or what it is for.
func (m *Main) repoLine() string {
	var taken *data.RepoTakenError
	switch s := m.manage.project; {
	case errors.Is(s.problem, data.ErrRepoInvalid):
		return m.say.ManageRepoInvalid()
	case errors.As(s.problem, &taken):
		return m.say.ManageRepoTaken(taken.Project)
	}
	return m.say.ManageRepoHelp()
}

// projectSheetKey is the keyboard of both fields: Enter creates.
func (m *Main) projectSheetKey(k ui.TextKey) bool {
	if k.Mods != 0 || k.Key != ui.KeyEnter {
		return false
	}
	m.createProject()
	return true
}

// createProject makes the project and opens it. A problem with the repository stays under the
// field; any other failure closes the sheet and shows the banner.
func (m *Main) createProject() {
	s := &m.manage.project
	if strings.TrimSpace(s.name) == "" {
		return
	}
	var parent *int
	if s.parent != 0 {
		id := s.parent
		parent = &id
	}
	p, err := m.app.CreateProject(data.NewProject{Name: s.name, RepoSlug: s.repo, ParentID: parent})
	var taken *data.RepoTakenError
	switch {
	case errors.Is(err, data.ErrRepoInvalid), errors.As(err, &taken):
		s.problem = err
	case err != nil:
		m.app.snap.err = err
		s.open = false
	default:
		s.open = false
		m.goTo(nav{screen: screenProject, projectID: p.ID})
	}
}
