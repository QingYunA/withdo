package views

import (
	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// convoSheet is the conversation under the ask field. It is part of the page, not a window or a
// chat: no bubbles, no avatars. My line in secondary text, the reply in body text, one row per
// change in the list-row style, and a question with its options as small tinted buttons. The
// sections below move down to make room, and a hairline marks where it ends.
func (m *Main) convoSheet(c *ui.Context, t *theme.Tokens) {
	cv := &m.convo
	if !cv.open || len(cv.turns) == 0 {
		return
	}
	if m.sheetExpired(c) {
		m.closeSheet()
		return
	}
	ui.Column(c.Key("convo")).Shrink(0).Margin(14, 0, 0).Children(func() {
		for i, turn := range cv.turns {
			m.turnView(c, t, i, turn, i == len(cv.turns)-1)
		}
		ui.Box(c.Key("convo-rule")).Height(1).Shrink(0).Margin(14, 0, 0).Background(t.Border)
	})
}

// sheetExpired closes the sheet 4 s after a turn that asked nothing, and schedules the redraw
// that gets it there.
func (m *Main) sheetExpired(c *ui.Context) bool {
	cv := &m.convo
	if !cv.sheetDone() {
		return false
	}
	age := m.app.Now().Sub(cv.doneAt)
	if age >= sheetClosesAfter {
		return true
	}
	c.After(sheetClosesAfter - age)
	return false
}

// turnView draws one exchange. Only the last turn's question can still be answered.
func (m *Main) turnView(c *ui.Context, t *theme.Tokens, i int, turn convoTurn, last bool) {
	ui.Column(c.Key("turn-"+itoa(i))).Shrink(0).Margin(sheetTurnGap(i), 0, 0).Children(func() {
		theme.Paragraph(c, theme.Body, t.TextSecondary, turn.user, 0)
		switch {
		case m.convo.thinking && last && turn.reply == "":
			// The ask field already says Thinking…; the sheet only shows what was asked.
		case turn.reply != "":
			theme.Paragraph(c, theme.Body, t.Text, turn.reply, 0).Margin(6, 0, 0)
		}
		for _, r := range turn.receipts {
			m.receiptRow(c, t, r)
		}
		if turn.question != nil {
			m.questionView(c, t, i, *turn.question, last && !m.convo.thinking)
		}
		if turn.stopped {
			theme.Text(c, theme.Meta, t.TextTertiary, m.say.ConvoStopped()).Margin(6, 0, 0)
		}
		if turn.err != "" {
			m.errorView(c, t, turn.err, last)
		}
	})
}

func sheetTurnGap(i int) float32 {
	if i == 0 {
		return 0
	}
	return 16
}

// receiptRow is one change, shaped like a list row: glyph, title, a quiet detail, and Undo.
func (m *Main) receiptRow(c *ui.Context, t *theme.Tokens, r data.Receipt) {
	undone := m.convo.undone[r.ID]
	row := ui.Row(c.Key("receipt-"+itoa(r.ID))).Shrink(0).Margin(4, -plainRow.horz, 0).Padding(7, plainRow.horz).Gap(12).
		AlignItems(ui.Center).Radius(theme.RadiusCard).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(t.BgHover)
	}
	row.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	row.Children(func() {
		if r.Bell {
			ui.Icon(c, theme.IconBell).Size(16, 16).TextColor(t.Accent).Shrink(0)
		} else {
			theme.StatusGlyph(c, t, r.Status, 0)
		}
		ink := t.Text
		if undone {
			ink = t.TextTertiary
		}
		title := theme.Text(c, theme.BodyStrong, ink, r.Title).Shrink(1).MinWidth(0).Ellipsis("…")
		if undone {
			title.Strikethrough()
		}
		theme.Text(c, theme.Meta, t.TextTertiary, m.detailText(r.Detail)).Shrink(2).MinWidth(0).Ellipsis("…")
		ui.Spacer(c)
		if undone {
			theme.Text(c, theme.Meta, t.TextTertiary, m.say.ConvoUndone()).Shrink(0)
		} else if m.undoButton(c, t, r).Clicked() {
			m.undo(r)
		}
	})
	if row.Clicked() && !undone {
		m.openRow(r.TaskID)
	}
}

// undoButton is the quiet Undo at the right of a receipt.
func (m *Main) undoButton(c *ui.Context, t *theme.Tokens, r data.Receipt) ui.Element {
	btn := ui.ButtonBase(c.Key("undo-"+itoa(r.ID))).Padding(3, 8).Radius(theme.RadiusControl).Shrink(0).
		Label(m.say.A11yUndo(r.Title)).Cursor(ui.CursorPointer)
	ink := t.TextSecondary
	if btn.Hovered() {
		btn.Background(t.BgSelected)
		ink = t.Accent
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.Text(c, theme.MetaStrong, ink, m.say.ConvoUndo()) })
	return btn
}

// questionView is a question in body text with its options as the small tinted buttons of the
// Needs you rows. Typing a free answer into the ask field works too.
func (m *Main) questionView(c *ui.Context, t *theme.Tokens, turn int, q data.Question, live bool) {
	theme.Paragraph(c, theme.Body, t.Text, q.Text, 0).Margin(6, 0, 0)
	if len(q.Options) == 0 || !live {
		return
	}
	ui.Row(c.Key("options-"+itoa(turn))).Shrink(0).Margin(10, 0, 0).Gap(8).Wrap().Children(func() {
		for _, o := range q.Options {
			if optionButton(c, t, o).Clicked() {
				m.pick(o)
			}
		}
	})
}

func optionButton(c *ui.Context, t *theme.Tokens, text string) ui.Element {
	btn := ui.ButtonBase(c.Key("option-"+text)).Padding(3, 9).Radius(theme.RadiusControl).Shrink(0).
		Background(t.BgSelected).Label(text).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.Accent.Alpha(0.2))
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.Text(c, theme.MetaStrong, t.Accent, text) })
	return btn
}

// errorView says what went wrong in neutral words, with Retry on the last turn.
func (m *Main) errorView(c *ui.Context, t *theme.Tokens, text string, last bool) {
	ui.Row(c).Shrink(0).Margin(6, 0, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.Paragraph(c, theme.Body, t.TextSecondary, text, 0).Shrink(1).MinWidth(0)
		if last && link(c, t, m.say.ConvoRetry()).Shrink(0).Clicked() {
			m.retry()
		}
	})
}
