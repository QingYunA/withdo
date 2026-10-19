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
	// n is how many rows go with it, and stack their ids (see stackKey): a row of several selected
	// ones takes them all.
	n     int
	stack string
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
	ph := m.rowPhase(c, task)
	ot := m.app.overviewTask(task)
	m.draggableRow(c, t, task, false, ph, func(row ui.Element, dragging bool) {
		row.AlignItems(ui.Center).Children(func() {
			if dragging { // the copy that follows the pointer carries the sprite of the user
				m.carrier(c, m.ho.walk(c, row, m.app.Now()))
				m.stackBadge(c, t)
			}
			m.groupGlyph(c, t, row, task, ph, false)
			m.indentSlot(c, t, task.ID)
			body(c, func() { m.titleLine(c, t, task, "", false) })
			right(c, func() { m.listRight(c, t, task, ot) })
		})
	})
}

// agentRow is an agent's task: its avatar, the title with the live dot and the age at the right
// (the buttons of its next step while it is hovered or selected, see agentActions), and under it what
// the agent last said, two lines at most, led by the pull request. No branch: the inspector has it.
func (m *Main) agentRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	if task.Status.Terminal() {
		m.doneRow(c, t, sideAgents, task)
		return
	}
	ph := m.rowPhase(c, task)
	ot := m.app.overviewTask(task)
	m.draggableRow(c, t, task, true, ph, func(row ui.Element, _ bool) {
		row.AlignItems(ui.Start).Children(func() {
			m.groupGlyph(c, t, row, task, ph, true)
			m.agentAvatar(c, t, task.ID, m.app.taskWho(task))
			body(c, func() {
				m.agentTitle(c, t, task, func() { m.agentRight(c, t, row, task, ot, ph) })
				agentLine(c, t, agentSaid(task, ot))
			})
		})
	})
}

// waitingRow is a task the user handed over that no agent has reported on yet: the robot of the
// Agents header (it is any agent's), the title and how long ago, and under it how to have one
// pick it up.
func (m *Main) waitingRow(c *ui.Context, t *theme.Tokens, task data.Task) {
	ph := m.rowPhase(c, task)
	m.draggableRow(c, t, task, true, ph, func(row ui.Element, _ bool) {
		row.AlignItems(ui.Start).Children(func() {
			m.agentAvatar(c, t, task.ID, theme.AccentBot(t.Accent))
			body(c, func() {
				m.agentTitle(c, t, task, func() {
					m.rowThumb(c, t, task.ID)
					plainText(c, t, m.clock.Age(handedAt(task), m.app.Now()))
				})
				agentLine(c, t, m.say.ListWaitingHint(task.ID))
			})
		})
	})
}

// agentAvatar is the avatar at the start of an agent's row, level with its title.
func (m *Main) agentAvatar(c *ui.Context, t *theme.Tokens, id int, who theme.Who) {
	m.lead(c, t, "author", id, who, theme.AvatarRow).Margin(2, listIndent-theme.AvatarRow-rowGap, 0, 0)
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

// notTheTitle is what an agent said, or nothing when that only repeats the task's title: a task
// made by its first report takes the summary as its title.
func notTheTitle(task data.Task, said string) string {
	if strings.TrimSpace(said) == strings.TrimSpace(task.Title) {
		return ""
	}
	return said
}

// agentSaid is what an agent last said, led by its pull request: "#12 · Board and lists are in". A
// blocked task says what it waits for.
func agentSaid(task data.Task, ot data.OverviewTask) string {
	var parts []string
	if task.PRNumber > 0 {
		parts = append(parts, "#"+itoa(task.PRNumber))
	}
	if r := ot.LatestReport; r != nil {
		said := notTheTitle(task, r.Summary)
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
	ph := m.rowPhase(c, task)
	row := m.rowShell(c, t, task.ID, plainRow, ph).AlignItems(ui.Center)
	row.Children(func() {
		if s == sideAgents {
			m.authorSlot(c, t, task)
		} else {
			m.indentSlot(c, t, task.ID)
		}
		body(c, func() {
			theme.Text(c, theme.Body, t.TextSecondary, task.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		})
		right(c, func() {
			m.rowThumb(c, t, task.ID)
			plainText(c, t, m.clock.Day(doneAt(task), m.app.Now()))
		})
	})
	if m.rowClicked(row, task.ID) {
		m.openRow(task.ID)
	}
}

// draggableRow is a row of a column that drags to the other one. While it is dragged it lifts (a
// shadow, a 2 degree tilt) and leaves an empty dashed slot where it was; fill builds the row.
func (m *Main) draggableRow(c *ui.Context, t *theme.Tokens, task data.Task, fromAgents bool, ph phase, fill func(row ui.Element, dragging bool)) {
	slot := ui.Column(c.Key("slot-"+itoa(task.ID))).Shrink(0).Margin(0, -rowSlot).Padding(0, rowSlot)
	dragging := false
	slot.Children(func() {
		row := m.rowShell(c, t, task.ID, plainRow, ph)
		stack := []int{task.ID}
		if !fromAgents {
			stack = m.trayStack(task)
		}
		row.Drag(dragRow{id: task.ID, fromAgents: fromAgents, n: len(stack), stack: stackKey(stack)})
		dragging = row.Dragging()
		if dragging {
			m.startedDrag(task.ID, stack)
		} else if m.fadesInStack(task.ID) {
			row.Opacity(stackFade)
		}
		m.liftRow(c, t, row, task.ID, dragging)
		fill(row, dragging)
		if m.rowClicked(row, task.ID) {
			m.openRow(task.ID)
		}
	})
	if dragging {
		slot.Radius(theme.RadiusCard).Border(1, t.Border).BorderStyle(ui.BorderDashed)
	}
}

// liftRow is the look of a row in flight, as a board card's (the copy that follows the pointer
// ignores the opacity; the original leaves an empty slot), and the accent wash of a row that just
// landed in Waiting for an agent.
func (m *Main) liftRow(c *ui.Context, t *theme.Tokens, row ui.Element, id int, dragging bool) {
	if wash := m.ho.rowWash(id, m.app.Now()); wash > 0 && !dragging && !row.Hovered() && m.sel != id && m.openID != id {
		row.Background(t.Accent.Alpha(wash))
	}
	if !c.Preferences().ReduceMotion { // with Reduce Motion the lift is the shadow alone
		tilt := row.AnimateTo("tilt", boolTo(dragging), ui.Snappy)
		row.Rotate(2*tilt).Scale(1+0.025*tilt, 1+0.025*tilt)
	}
	if dragging { // the copy in flight is a card, not the width of a column
		row.MaxWidth(dragCardWidth).Radius(theme.RadiusCard).Border(1, t.TextTertiary).
			Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.5)).Shadow(0, 16, 32, 0, t.Shadow).Opacity(0)
		if n := len(m.ho.stack); n > 1 { // cards fan out behind it, painted with its own
			lifted := !c.Preferences().ReduceMotion
			row.Background(ui.Transparent).Border(1, ui.Transparent).
				Draw(func(p *ui.Painter, r ui.Rect) { stackPaint(p, r, t, n, theme.RadiusCard, lifted) })
		} else {
			row.Background(t.Bg)
		}
	}
}
