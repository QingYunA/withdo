package views

import (
	"errors"
	"log"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
)

// OpenWith is where Review and Answer on a handed-over row take the user: to the agent's app, or
// to the folder the agent works in, in a terminal, an editor or Finder.
type OpenWith string

// The places an agent's work can be opened.
const (
	OpenAgentApp OpenWith = "agentApp"
	OpenTerminal OpenWith = "terminal"
	OpenEditor   OpenWith = "editor"
	OpenFinder   OpenWith = "finder"
)

// openWiths lists the places in the order Settings and the task menu show them.
var openWiths = []OpenWith{OpenAgentApp, OpenTerminal, OpenEditor, OpenFinder}

// What OpenIn cannot do, for the views to word in the user's language.
var (
	// ErrNoDir says there is no folder to open: the agent never said where it works.
	ErrNoDir = errors.New("no folder to open")
	// ErrNoEditor says none of the editors OpenIn knows is installed.
	ErrNoEditor = errors.New("no editor installed")
)

// openFailure is the last OpenIn that failed, kept to draw one error line under that task's row.
type openFailure struct {
	id  int
	err error
}

// OpenWith is where Review and Answer go: the choice saved in Settings, else the agent's app.
func (a *App) OpenWith() OpenWith {
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	a.loadPrefs()
	for _, o := range openWiths {
		if OpenWith(a.prefs.OpenWith) == o {
			return o
		}
	}
	return OpenAgentApp
}

// SetOpenWith chooses where Review and Answer go, and keeps the choice.
func (a *App) SetOpenWith(o OpenWith) {
	a.savePrefs(func(p *data.Prefs) { p.OpenWith = string(o) })
}

// workDir is where the agent of a task works: the folder its newest report named, else its
// project's, else "".
func (a *App) workDir(t data.Task) string {
	path := ""
	if t.ProjectID != nil {
		if p, ok := a.Project(*t.ProjectID); ok {
			path = p.RepoPath
		}
	}
	return data.WorkDir(a.snap.details[t.ID].Reports, path)
}

// canOpen reports whether a task has anywhere to be opened in: a terminal, an editor and Finder
// need its folder, the agent's app needs the agent to have one.
func (a *App) canOpen(how OpenWith, t data.Task) bool {
	return a.workDir(t) != "" || how == OpenAgentApp && agentApp(a.taskAgent(t)) != ""
}

// openWork opens where the agent of a task works, in the place asked for. A failure leaves one
// error line under the task's row until the next success or until the row is opened.
func (m *Main) openWork(how OpenWith, t data.Task) {
	err := m.app.Act.OpenIn(how, agentApp(m.app.taskAgent(t)), m.app.workDir(t))
	if err != nil {
		log.Printf("open %s: %v", how, err)
		m.openFail = openFailure{id: t.ID, err: err}
		return
	}
	m.openFail = openFailure{}
}

// openFailText words the error line under a row.
func (m *Main) openFailText(err error) string {
	switch {
	case errors.Is(err, ErrNoDir):
		return m.say.RowOpenNoFolder()
	case errors.Is(err, ErrNoEditor):
		return m.say.RowOpenNoEditor()
	}
	return m.say.RowOpenFailed(err.Error())
}

// openWithName is a place as the interface words it.
func openWithName(say locales.Messages, o OpenWith) string {
	switch o {
	case OpenTerminal:
		return say.OpenWithTerminal()
	case OpenEditor:
		return say.OpenWithEditor()
	case OpenFinder:
		if systemGOOS == "windows" {
			return say.OpenWithExplorer()
		}
		return say.OpenWithFinder()
	}
	return say.OpenWithAgentApp()
}
