package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// cardMotion lets cards settle into a column with the spring, and close the gap they leave.
var cardMotion = ui.ElementTransition{
	Animation: &ui.Snappy,
	Enter:     &ui.Motion{Scale: 0.94},
	Exit:      &ui.Motion{Collapse: true},
}

// card is one task on the board: its title and one line of meta. It drags to another lane; while
// it is dragged the card lifts (a shadow, a 2 degree tilt) and leaves an empty slot where it was.
func (m *Main) card(c *ui.Context, t *theme.Tokens, task data.Task) {
	ph := m.app.completionPhase(c, task)
	slot := ui.Column(c.Key(task.ID)).Transition(cardMotion)
	dragging := false
	slot.Children(func() {
		card := ui.Column(c.Key("card")).Padding(12).Gap(4).Radius(theme.RadiusTile).Cursor(ui.CursorPointer)
		card.Drag(dragCard{id: task.ID})
		dragging = card.Dragging()
		if m.sel == task.ID {
			m.selRect = card.Bounds()
		}
		m.styleCard(c, t, card, task, dragging)
		card.Children(func() {
			if task.Status == data.Done {
				theme.Paragraph(c, theme.Body, t.TextSecondary, task.Title, 3)
			} else {
				theme.Paragraph(c, theme.BodyStrong, t.Text, task.Title, 3)
			}
			m.cardMeta(c, t, task, m.showsActions(card, task.ID, ph), ph)
		})
		if card.RightClicked() {
			m.openMenuAtPointer(card, task.ID)
		}
		if card.Clicked() {
			m.openID = task.ID
			m.inspector.reset()
		}
	})
	if dragging {
		slot.Radius(theme.RadiusTile).Border(1, t.Border).BorderStyle(ui.BorderDashed)
	}
}

// styleCard is the card's look: white with a hairline ring and a soft shadow; a finished task's is
// a flat well. The open, selected and menu'd card is tinted with the accent.
func (m *Main) styleCard(c *ui.Context, t *theme.Tokens, card ui.Element, task data.Task, dragging bool) {
	bg, border := t.Bg, t.Border
	flat := task.Status == data.Done
	if flat {
		bg, border = t.BgSubtle, ui.Transparent
	}
	switch {
	case dragging:
		border = t.TextTertiary
	case m.openID == task.ID || m.sel == task.ID || (m.pop.open && m.pop.task == task.ID):
		bg, border = t.Bg.Mix(t.Accent, 0.07), t.Accent.Alpha(0.6)
	case card.Hovered():
		border = t.TextTertiary
	}
	card.Background(bg).Border(1, border)
	if !flat {
		card.Shadow(0, 1, 2, 0, t.Shadow.Alpha(0.14))
	}
	card.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if !c.Preferences().ReduceMotion { // with Reduce Motion the lift is the shadow alone
		tilt := card.AnimateTo("tilt", boolTo(dragging), ui.Snappy)
		card.Rotate(2*tilt).Scale(1+0.025*tilt, 1+0.025*tilt)
	}
	if dragging {
		// The copy that follows the pointer ignores this opacity; the original leaves an empty slot.
		card.Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.5)).Shadow(0, 16, 32, 0, t.Shadow).Opacity(0)
	}
}

func boolTo(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// cardMeta is the card's one line: who wrote it (a 12 pt avatar), the blocked mark, a P0 or P1 badge, the branch, and at the
// right the due label or the age (or the stale label). A lane is narrow, so the branch shows only
// when no mark or badge competes with it: cut down to "fix…" it says nothing. The author's avatar is
// always there, so the line always is.
func (m *Main) cardMeta(c *ui.Context, t *theme.Tokens, task data.Task, acting bool, ph phase) {
	right := m.metaRight(task)
	var under func() // the buttons, when two do not fit beside the avatar in a narrow lane
	branch := task.Branch
	if actions := m.agentActions(task); acting && len(actions) > 0 {
		buttons := func() { m.actionButtons(c, t, task.ID, actions, ph) }
		branch, right = "", nil
		if len(actions) > 1 && m.boardLayout(c).col < cardActionsInlineFrom {
			under = buttons
		} else {
			right = func(*ui.Context, *theme.Tokens) { buttons() }
		}
	}
	badge := theme.ShowsPriority(task.Priority) && !task.Status.Terminal()
	blocked := task.Status == data.Blocked
	if badge || blocked {
		branch = ""
	}
	ui.Row(c).Gap(8).MinHeight(18).AlignItems(ui.Center).Children(func() {
		theme.AvatarIcon(c.Key("author"), m.app.taskWho(task), theme.AvatarSmall).Shrink(0)
		if blocked {
			theme.StatusGlyphSized(c, t, data.Blocked, theme.IconSmall, 0)
		}
		if badge {
			theme.PriorityBadge(c, t, task.Priority)
		}
		if branch != "" {
			theme.Text(c, theme.Mono, t.TextSecondary, branch).Shrink(4).MinWidth(0).Ellipsis("…")
		}
		ui.Spacer(c)
		if right != nil {
			right(c, t)
		}
	})
	if under != nil {
		ui.Row(c).Gap(8).Margin(4, 0, 0).Children(under)
	}
}

// cardActionsInlineFrom is the lane width from which Open PR and Accept fit in a card's meta line;
// in a narrower lane they take a line of their own under it.
const cardActionsInlineFrom float32 = 200

// metaRight builds the right end of the second line, or returns nil when there is nothing to say.
func (m *Main) metaRight(task data.Task) func(*ui.Context, *theme.Tokens) {
	now := m.app.Now()
	switch {
	case task.Status.Terminal():
		at := doneAt(task)
		return plainMeta(m.clock.Age(at, now))
	case task.Source == data.SourceUser:
		words := m.clock.DueLabel(task, now)
		if words == "" {
			return nil
		}
		reminder := task.RemindAt != nil && task.RemindedAt == nil
		return func(c *ui.Context, t *theme.Tokens) {
			ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Meta, t.TextSecondary, words)
				if reminder {
					theme.ClockGlyph(c, t.TextSecondary)
				}
			})
		}
	}
	ot := m.app.overviewTask(task)
	if ot.Stale && task.Status.Active() {
		hours := ot.HoursSinceActivity
		return func(c *ui.Context, t *theme.Tokens) {
			theme.Text(c, theme.Meta, t.Stale, m.clock.Stale(hours)).Shrink(0)
		}
	}
	since := task.UpdatedAt
	if ot.LatestReport != nil {
		since = ot.LatestReport.CreatedAt
	}
	return plainMeta(m.clock.Age(since, now))
}

func plainMeta(text string) func(*ui.Context, *theme.Tokens) {
	return func(c *ui.Context, t *theme.Tokens) {
		theme.Text(c, theme.Meta, t.TextTertiary, text).Shrink(0)
	}
}

func (m *Main) milestoneLabel(ms data.Milestone) string {
	switch {
	case !ms.Open:
		return m.say.MilestoneClosed(ms.Title)
	case ms.DueAt != nil:
		return m.say.MilestoneDue(ms.Title, m.clock.DayInline(*ms.DueAt, m.app.Now()))
	}
	return ms.Title
}
