package views

import (
	"fmt"

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

// card is one task on the board. It drags to another column; while it is dragged the card lifts
// (a shadow, a 2 degree tilt) and leaves an empty slot where it was.
func (m *Main) card(c *ui.Context, t *theme.Tokens, task data.Task) {
	fill, opacity, _ := m.app.completionPhase(c, task.ID)
	slot := ui.Column(c.Key(task.ID)).Transition(cardMotion)
	dragging := false
	slot.Children(func() {
		card := ui.Column(c.Key("card")).Padding(9, 12, 12, 8).Radius(theme.RadiusCard).Cursor(ui.CursorPointer)
		card.Drag(dragCard{id: task.ID})
		card.ContextMenu(func(mn *ui.Menu) { m.taskMenu(mn, task) })
		dragging = card.Dragging()
		m.styleCard(c, t, card, task, dragging, opacity)
		card.Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Start).Children(func() {
				m.app.glyphButton(c, t, task, fill)
				m.cardText(c, t, task)
			})
		})
		if card.Clicked() {
			m.openID = task.ID
			m.inspector.reset()
		}
	})
	if dragging {
		m.dragging = task.ID
		slot.Radius(theme.RadiusCard).Border(1, t.Border).BorderStyle(ui.BorderDashed)
	}
}

func (m *Main) styleCard(c *ui.Context, t *theme.Tokens, card ui.Element, task data.Task, dragging bool, opacity float32) {
	bg, border := t.Bg, t.Border
	switch {
	case dragging:
		border = t.TextTertiary
	case m.openID == task.ID || m.sel == task.ID:
		bg, border = t.Bg.Mix(t.Accent, 0.07), t.Accent.Alpha(0.6)
	case card.Hovered():
		border = t.TextTertiary
	}
	card.Background(bg).Border(1, border)
	card.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if !c.Preferences().ReduceMotion { // with Reduce Motion the lift is the shadow alone
		tilt := card.AnimateTo("tilt", boolTo(dragging), ui.Snappy)
		card.Rotate(2*tilt).Scale(1+0.025*tilt, 1+0.025*tilt)
	}
	if dragging {
		// The copy that follows the pointer ignores this opacity; the original leaves an empty slot.
		card.Shadow(0, 2, 6, 0, t.Shadow.Alpha(0.5)).Shadow(0, 16, 32, 0, t.Shadow).Opacity(0)
	} else if opacity < 1 {
		card.Opacity(opacity)
	}
}

func boolTo(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// cardText is the title and, below it, whatever else is worth a glance.
func (m *Main) cardText(c *ui.Context, t *theme.Tokens, task data.Task) {
	ui.Column(c).Grow(1).MinWidth(0).Margin(3, 0, 0).Gap(4).Children(func() {
		theme.Paragraph(c, theme.BodyStrong, t.Text, task.Title, 3)
		m.cardMeta(c, t, task)
	})
}

// cardMeta is the second line: priority bars when it is not the default, the branch, the pull
// request, and on the right the agent's age, a stale label, or when I wanted to be reminded.
func (m *Main) cardMeta(c *ui.Context, t *theme.Tokens, task data.Task) {
	right := m.metaRight(task)
	if task.Priority == data.P2 && task.Branch == "" && task.PRNumber == 0 && right == nil {
		return
	}
	ui.Row(c).Gap(8).MinHeight(16).AlignItems(ui.Center).Children(func() {
		if task.Priority != data.P2 {
			theme.PriorityBars(c, t, task.Priority).Shrink(0)
		}
		if task.Branch != "" {
			pill(c, t, task.Branch)
		}
		if task.PRNumber > 0 {
			theme.Text(c, theme.Mono, t.TextSecondary, fmt.Sprintf("#%d", task.PRNumber)).Shrink(0)
		}
		ui.Spacer(c)
		if right != nil {
			right(c, t)
		}
	})
}

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
			theme.Text(c, theme.Meta, t.Stale, "! "+m.clock.Stale(hours)).Shrink(0)
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
