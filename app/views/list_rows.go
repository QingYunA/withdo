package views

import (
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dragRow is the value a row carries while it is dragged to the other column: the user's task to the
// Agents tray, to hand it over; an agent's to Mine, to take it back.
type dragRow struct {
	id         int
	fromAgents bool
}

// rowSlot is how far a row's highlight bleeds past its column; the slot of a dragged row bleeds the
// same, so its dashed outline lines up with it.
const rowSlot float32 = 10

// dragCardWidth is how wide a row is while it is carried (v7b draws it 230 pt, plus the bleed).
const dragCardWidth float32 = 250

// myRow is a task of mine: its title, and at the right a P0 or P1 badge and when it is due. No
// avatar: the column says whose it is; the title still lines up with the Agents column's titles. A task that is not finished drags to the Agents tray.
func (m *Main) myRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.Status.Terminal() {
		m.doneRow(c, t, sideMine, task)
		return
	}
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	ot := m.app.overviewTask(task)
	m.draggableRow(c, t, task, false, opacity, func(row ui.Element) {
		row.AlignItems(ui.Center).Children(func() {
			ui.Box(c.Key("indent")).Width(listIndent - rowGap).Shrink(0)
			body(c, func() { m.titleLine(c, t, task, "", false) })
			right(c, func() { m.listRight(c, t, task, ot) })
		})
	})
}

// agentRow is an agent's task: its avatar, the title with the live dot and the age at the right,
// and under it what the agent last said, two lines at most, led by the pull request. No branch:
// the inspector has it.
func (m *Main) agentRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.Status.Terminal() {
		m.doneRow(c, t, sideAgents, task)
		return
	}
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	ot := m.app.overviewTask(task)
	m.draggableRow(c, t, task, true, opacity, func(row ui.Element) {
		row.AlignItems(ui.Start).Children(func() {
			m.agentAvatar(c, m.app.taskWho(task))
			body(c, func() {
				m.agentTitle(c, t, task, func() { m.listRight(c, t, task, ot) })
				agentLine(c, t, agentSaid(task, ot))
			})
		})
	})
}

// waitingRow is a task the user handed to an agent that has not reported on it yet: the agent's
// avatar, the title and how long ago, and under it who has it and how it was started.
func (m *Main) waitingRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	m.draggableRow(c, t, task, true, opacity, func(row ui.Element) {
		row.AlignItems(ui.Start).Children(func() {
			m.agentAvatar(c, theme.AgentAvatar(task.HandedTo))
			body(c, func() {
				m.agentTitle(c, t, task, func() { plainText(c, t, m.clock.Age(handedAt(task), m.app.Now())) })
				agentLine(c, t, m.handedLine(task))
			})
		})
	})
}

// handedLine is "Handed to Claude Code · opened in Terminal", or without the second half when this
// run does not know how it was handed over (the app was restarted since).
func (m *Main) handedLine(task data.Task) string {
	agent := agentName(task.HandedTo)
	launched, known := m.ho.how[task.ID]
	switch {
	case !known:
		return m.say.ListHandedTo(agent)
	case launched:
		return m.say.ListHandedToOpened(agent)
	}
	return m.say.ListHandedToCopied(agent)
}

// agentAvatar is the avatar at the start of an agent's row, level with its title.
func (m *Main) agentAvatar(c *ui.Context, who theme.Who) {
	theme.AvatarIcon(c.Key("author"), who, theme.AvatarRow).Margin(2, listIndent-theme.AvatarRow-rowGap, 0, 0)
}

// agentTitle is the title line of an agent's row, and what goes at its right.
func (m *Main) agentTitle(c *ui.Context, t *theme.Tokens, task data.Task, at func()) {
	ui.Row(c).Gap(10).MinWidth(0).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.BodyStrong, t.Text, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		ui.Spacer(c)
		right(c, at)
	})
}

// agentLine is the quiet lines under an agent's title: two at most.
func agentLine(c *ui.Context, t *theme.Tokens, text string) {
	if text != "" {
		theme.Paragraph(c, theme.Meta, t.TextSecondary, text, 2).Margin(2, 0, 0)
	}
}

// agentSaid is what an agent last said, led by its pull request: "#12 · Board and lists are in". A
// blocked task says what it waits for.
func agentSaid(task data.Task, ot data.OverviewTask) string {
	var parts []string
	if task.PRNumber > 0 {
		parts = append(parts, "#"+itoa(task.PRNumber))
	}
	if r := ot.LatestReport; r != nil {
		said := r.Summary
		if task.Status == data.Blocked && r.NextStep != "" {
			said = r.NextStep
		}
		if said != "" {
			parts = append(parts, said)
		}
	}
	return strings.Join(parts, " · ")
}

// doneRow is a finished task of a column: its title in text.secondary and when it was finished.
func (m *Main) doneRow(c *ui.Context, t *theme.Tokens, s side, task data.Task) {
	_, opacity, _ := m.app.completionPhase(c, task.ID)
	row := m.rowShell(c, t, task.ID, plainRow, opacity).AlignItems(ui.Center)
	row.Children(func() {
		if s == sideAgents {
			m.authorSlot(c, task)
		} else {
			ui.Box(c.Key("indent")).Width(listIndent - rowGap).Shrink(0)
		}
		body(c, func() {
			theme.Text(c, theme.Body, t.TextSecondary, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		})
		right(c, func() { plainText(c, t, m.clock.Day(doneAt(task), m.app.Now())) })
	})
	if row.Clicked() {
		m.openRow(task.ID)
	}
}

// draggableRow is a row of a column that drags to the other one. While it is dragged it lifts (a
// shadow, a 2 degree tilt) and leaves an empty dashed slot where it was; fill builds the row. A
// row waiting for an agent that has just opened shows only as the hand-off lets it.
func (m *Main) draggableRow(c *ui.Context, t *theme.Tokens, task data.Task, fromAgents bool, opacity float32, fill func(row ui.Element)) {
	opacity *= m.ho.rowOpacity(task.ID)
	slot := ui.Column(c.Key("slot-"+itoa(task.ID))).Shrink(0).Margin(0, -rowSlot).Padding(0, rowSlot)
	dragging := false
	slot.Children(func() {
		row := m.rowShell(c, t, task.ID, plainRow, opacity)
		row.Drag(dragRow{id: task.ID, fromAgents: fromAgents})
		dragging = row.Dragging()
		m.liftRow(c, t, row, task.ID, dragging)
		fill(row)
		if row.Clicked() {
			m.openRow(task.ID)
		}
	})
	if dragging {
		m.ho.dragged = dragged{id: task.ID, rect: slot.Bounds()}
		slot.Radius(theme.RadiusCard).Border(1, t.Border).BorderStyle(ui.BorderDashed)
	}
}

// liftRow is the look of a row in flight, as a board card's (the copy that follows the pointer
// ignores the opacity; the original leaves an empty slot), and the accent wash of a row the
// hand-off just opened.
func (m *Main) liftRow(c *ui.Context, t *theme.Tokens, row ui.Element, id int, dragging bool) {
	if wash := m.ho.rowWash(id); wash > 0 && !dragging && !row.Hovered() && m.sel != id && m.openID != id {
		row.Background(t.Accent.Alpha(wash))
	}
	if !c.Preferences().ReduceMotion { // with Reduce Motion the lift is the shadow alone
		tilt := row.AnimateTo("tilt", boolTo(dragging), ui.Snappy)
		row.Rotate(2*tilt).Scale(1+0.025*tilt, 1+0.025*tilt)
	}
	if dragging { // the copy in flight is a card, not the width of a column
		row.MaxWidth(dragCardWidth).Background(t.Bg).Border(1, t.TextTertiary).
			Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.5)).Shadow(0, 16, 32, 0, t.Shadow).Opacity(0)
	}
}
