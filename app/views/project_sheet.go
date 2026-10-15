package views

import (
	"errors"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// projectSheet is the New project sheet: a name and an optional GitHub repository.
type projectSheet struct {
	open       bool
	name, repo string
	problem    error // what the last Create found wrong with the repository
}

func (m *Main) openNewProject() {
	m.manage.project = projectSheet{open: true}
}

// newProjectSheet draws the sheet while it is open.
func (m *Main) newProjectSheet(c *ui.Context, t *theme.Tokens) {
	s := &m.manage.project
	if !s.open {
		return
	}
	sheet(c, t, &s.open, func() {
		theme.Text(c, theme.Title, t.Text, m.say.ManageNewProjectTip())
		labeled(c, t, m.say.ManageFieldName(), func() {
			wellInput(c, t, "project-name", &s.name, m.say.ManageFieldName(), "", theme.Body, true, m.projectSheetKey)
		})
		labeled(c, t, m.say.ManageFieldRepo(), func() {
			wellInput(c, t, "project-repo", &s.repo, m.say.ManageFieldRepo(), m.say.ManageRepoPlaceholder(), theme.Mono, false, m.projectSheetKey)
			theme.Paragraph(c, theme.Meta, t.TextSecondary, m.repoLine(), 0)
		})
		cancelled, create := sheetButtons(c, m.say.ManageCancel(), m.say.ManageCreate(), strings.TrimSpace(s.name) != "", false)
		if cancelled {
			s.open = false
		}
		if create {
			m.createProject()
		}
	})
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
	p, err := m.app.CreateProject(data.NewProject{Name: s.name, RepoSlug: s.repo})
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
