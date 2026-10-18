package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// The buttons of an agent's task are the same wherever it is listed: on Mine, in a project's Agents
// column, in a list and on a board card.

func (m *Main) openPRAction(task data.Task) rowAction {
	return rowAction{text: m.say.ActionOpenPR(), label: m.say.A11yOpenPR(itoa(task.PRNumber)), run: func() { m.app.Act.OpenURL(task.PRURL) }}
}

// goSeeAction takes the user to where the agent is, the way Settings says, rather than to the inspector.
func (m *Main) goSeeAction(text string, task data.Task) rowAction {
	return rowAction{text: text, label: m.say.A11yOpenTask(task.Title), run: func() { m.openWork(m.app.OpenWith(), task) }}
}

func (m *Main) acceptAction(task data.Task) rowAction {
	return rowAction{text: m.say.ActionAccept(), label: m.say.A11yAcceptTask(task.Title), accepts: true, run: func() { m.app.Accept(task) }}
}

// openPageAction opens the first page the agent attached to its latest report.
func (m *Main) openPageAction(task data.Task, link data.Link) rowAction {
	return rowAction{text: m.say.ActionOpenPage(), label: m.say.A11yOpenPage(task.Title), icon: theme.IconFile, run: func() { m.openLink(link) }}
}

// agentActions are the buttons of an agent's task that waits for the user: blocked, Answer; in
// review, Open page when the latest report attached one, Open PR (or Review, without a pull request
// and without a page) and Accept. Any other task has none.
func (m *Main) agentActions(task data.Task) []rowAction {
	if task.Source != data.SourceAgent {
		return nil
	}
	if task.Status == data.Blocked {
		return []rowAction{m.goSeeAction(m.say.ActionAnswer(), task)}
	}
	if task.Status != data.Review {
		return nil
	}
	var actions []rowAction
	if r := m.app.latestReport(task.ID); r != nil && len(r.Links) > 0 {
		actions = append(actions, m.openPageAction(task, r.Links[0]))
	}
	switch {
	case task.PRURL != "":
		actions = append(actions, m.openPRAction(task))
	case len(actions) == 0:
		actions = append(actions, m.goSeeAction(m.say.ActionReview(), task))
	}
	return append(actions, m.acceptAction(task))
}

// showsActions says whether a hovered or selected row, or one being accepted, shows its buttons
// where it shows the age. While its task menu is open the menu has the choices, not the row.
func (m *Main) showsActions(row ui.Element, id int, ph phase) bool {
	if m.pop.open && m.pop.task == id && ph.stamp < 0 {
		return false
	}
	return row.Hovered() || m.sel == id || ph.stamp >= 0
}

// actionButtons draws the buttons of a row; once the task is being accepted its Accept button is
// stamped.
func (m *Main) actionButtons(c *ui.Context, t *theme.Tokens, id int, actions []rowAction, ph phase) {
	for i, a := range actions {
		stamp := -1
		if a.accepts {
			stamp = ph.stamp
		}
		if stampButton(c, t, "action-"+itoa(i)+"-"+itoa(id), a.text, a.label, stamp, a.icon).Clicked() {
			a.run()
		}
	}
}

// agentRight is the right end of an agent's row in a list or a column: the buttons of its next step
// while the row is hovered or selected, else the age and what goes with it.
func (m *Main) agentRight(c *ui.Context, t *theme.Tokens, row ui.Element, task data.Task, ot data.OverviewTask, ph phase) {
	if actions := m.agentActions(task); len(actions) > 0 && m.showsActions(row, task.ID, ph) {
		m.actionButtons(c, t, task.ID, actions, ph)
		return
	}
	m.listRight(c, t, task, ot)
}
