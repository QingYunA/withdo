package views

import (
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Handing a task to an agent: the user drags a row of Mine onto the Agents tray. The view words the
// prompt in the app's language, asks the store to hand the task over, copies the prompt when no agent
// could be started, and says so in a toast with Undo (and ⌘Z). Dragging an agent's row onto Mine
// takes it back. The animation is in handoff_anim.go.

// handOffToast is the one toast of the hand-off, so a second one replaces the first.
const handOffToast = "hand-off"

// HandOff hands a task to an agent and refreshes. An error goes to the banner.
func (a *App) HandOff(id int, prompt string) (data.HandOffResult, error) {
	var res data.HandOffResult
	err := a.do(func() (err error) { res, err = a.Store.HandOff(id, prompt); return })
	return res, a.fail(err)
}

// TakeBack makes an agent's task the user's again, and refreshes.
func (a *App) TakeBack(id int) (data.Task, error) {
	var task data.Task
	err := a.do(func() (err error) { task, err = a.Store.TakeBack(id); return })
	return task, a.fail(err)
}

// RestoreHandOff puts a task's Source and hand-off back as they were, which starts nothing.
func (a *App) RestoreHandOff(id int, agent string, at *time.Time, source data.Source) error {
	return a.fail(a.do(func() error { _, err := a.Store.RestoreHandOff(id, agent, at, source); return err }))
}

// handOffPrompt is what the agent starts with, in the app's language: the task, its notes when it
// has any, and how to report.
func (m *Main) handOffPrompt(task data.Task) string {
	if notes := strings.TrimSpace(task.Notes); notes != "" {
		return m.say.HandOffPromptNotes(task.ID, task.Title, notes)
	}
	return m.say.HandOffPrompt(task.ID, task.Title)
}

// handOffAgentName is the agent a hand-off would go to, as far as the view can tell: the one that
// reported last in the project, else Claude Code. The store picks the agent itself; this is the name
// the tray shows while a row is over it.
func (m *Main) handOffAgentName() string {
	var newest *data.Report
	for _, t := range m.app.snap.tasks {
		if t.ProjectID == nil || *t.ProjectID != m.nav.projectID {
			continue
		}
		if r := m.app.latestReport(t.ID); r != nil && (newest == nil || r.CreatedAt.After(newest.CreatedAt)) {
			newest = r
		}
	}
	if newest == nil || newest.Agent == "" || newest.Agent == unnamedAgent {
		return agentName("claude")
	}
	return agentName(newest.Agent)
}

// handOff is the drop on the tray: the task becomes the agent's and waits for it, the prompt goes to
// the clipboard when no agent started, and the hand-off plays.
func (m *Main) handOff(c *ui.Context, id int) {
	task, ok := m.app.Task(id)
	if !ok || task.Source != data.SourceUser || task.Status.Terminal() {
		return
	}
	prompt := m.handOffPrompt(task)
	res, err := m.app.HandOff(id, prompt)
	if err != nil {
		return
	}
	if m.ho.how == nil {
		m.ho.how = map[int]bool{}
	}
	m.ho.how[id] = res.Launched
	title := m.say.HandOffOpened(res.AgentName)
	if !res.Launched {
		c.WriteClipboard(prompt)
		title = m.say.HandOffCopied()
	}
	m.ho.begin(m.handOffPlan(c, task), task, res, m.app.Now())
	m.offerUndo(c, title, m.undoHandOff(id, task.Status))
}

// undoHandOff takes the task back and gives it the status it had. It cannot close the Terminal
// window the agent was started in.
func (m *Main) undoHandOff(id int, was data.Status) func() {
	return func() {
		if _, err := m.app.TakeBack(id); err != nil {
			return
		}
		_ = m.app.Update(id, data.Patch{Status: data.StatusPtr(was)})
		delete(m.ho.how, id)
		if m.ho.anim != nil && m.ho.anim.id == id {
			m.ho.anim = nil
		}
	}
}

// takeBack is the drop of an agent's row on Mine: the task is the user's again, with no animation.
func (m *Main) takeBack(c *ui.Context, id int) {
	task, ok := m.app.Task(id)
	if !ok || task.Source != data.SourceAgent || task.Status.Terminal() {
		return
	}
	if _, err := m.app.TakeBack(id); err != nil {
		return
	}
	delete(m.ho.how, id)
	m.offerUndo(c, m.say.HandOffTakenBack(), func() {
		_ = m.app.RestoreHandOff(id, task.HandedTo, task.HandedAt, task.Source)
	})
}

// offerUndo says what happened in a toast with Undo, which ⌘Z does as well for as long as it shows.
func (m *Main) offerUndo(c *ui.Context, title string, undo func()) {
	m.ho.undoSeq++
	seq := m.ho.undoSeq
	m.ho.undo = undo
	c.AddToast(ui.Toast{
		ID: handOffToast, Title: title, Action: m.say.MenuUndo(), Timeout: undoWithin, OnAction: undo,
		OnClose: func() {
			if m.ho.undoSeq == seq {
				m.ho.undo = nil
			}
		},
	})
}

// undoHandOffKey is ⌘Z while the toast of a hand-off shows.
func (m *Main) undoHandOffKey(c *ui.Context) {
	undo := m.ho.undo
	m.ho.undo = nil
	undo()
	c.CloseToast(handOffToast)
}
