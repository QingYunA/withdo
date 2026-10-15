package views

import (
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// Lines of receipts a turn shows before "and N more".
const maxReceiptLines = 3

// answerLineHeight is one line of the answer, receipt or sentence.
const answerLineHeight float32 = 34

// answerView is what came back, under the input, and nothing else: the user's own words are not
// shown again and there is no divider. A turn that changed something is one line per task, a green
// check, what was done and Undo at the right; one that changed nothing is a grey line with the
// assistant's sentence. Both fade out on their own. A question stays until it is answered, and an
// error stays with Retry. In the inspector (compact) the same lines sit above its input.
func (m *Main) answerView(c *ui.Context, t *theme.Tokens, compact bool) {
	cv := &m.convo
	if !cv.open || len(cv.turns) == 0 {
		return
	}
	fade, gone := m.answerFade(c)
	if gone {
		cv.open = false // the turn is kept for Cmd+Z, only its line goes
		return
	}
	turn, left := cv.turns[len(cv.turns)-1], float32(16)
	if compact {
		left = 0
	}
	ui.Column(c.Key("answer")).Shrink(0).Margin(8, 0, 0).Padding(0, 12, 0, left).Opacity(fade).Children(func() {
		switch {
		case m.convo.thinking && turn.reply == "":
			// The composer already says Thinking….
		case turn.question != nil:
			m.questionView(c, t, *turn.question, !cv.thinking)
		case len(turn.receipts) > 0:
			m.receiptLines(c, t, turn, compact)
		case turn.reply != "" && turn.err == "":
			m.sentenceLine(c, t, turn.reply)
		}
		if turn.stopped {
			theme.Text(c, theme.Meta, t.TextTertiary, m.say.ConvoStopped()).Margin(4, 0, 0)
		}
		if turn.err != "" {
			m.errorView(c, t, turn.err)
		}
	})
}

// answerFade says how visible the line of a turn that asked nothing still is, and when it is gone:
// it shows for 6 s, then fades in 400 ms, and goes at once when the composer it came from holds
// something else than it did then (typing again). A question or an error never fades.
func (m *Main) answerFade(c *ui.Context) (opacity float32, gone bool) {
	cv := &m.convo
	if !cv.sheetDone() {
		return 1, false
	}
	if m.cmp[cv.owner].buf != cv.typed {
		return 0, true
	}
	age := m.app.Now().Sub(cv.doneAt)
	switch {
	case age >= answerFor:
		return 0, true
	case age >= answerFor-answerFadeFor:
		c.After(40 * time.Millisecond)
		return float32(answerFor-age) / float32(answerFadeFor), false
	}
	c.After(answerFor - answerFadeFor - age)
	return 1, false
}

// receiptLines is one line per task, at most three, then "and N more".
func (m *Main) receiptLines(c *ui.Context, t *theme.Tokens, turn convoTurn, compact bool) {
	shown := turn.receipts
	if len(shown) > maxReceiptLines {
		shown = shown[:maxReceiptLines]
	}
	for _, r := range shown {
		// Beside the task's own title in the inspector the line need not say it again.
		m.receiptLine(c, t, r, compact && r.TaskID == m.inspector.lastOpen)
	}
	if more := len(turn.receipts) - len(shown); more > 0 {
		theme.Text(c, theme.Compact, t.TextTertiary, m.say.ResultMore(more)).Margin(0, 0, 0, 22).Height(24)
	}
}

// receiptLine is "✓ Added title · Fri · P1" (the check is green) with Undo at the right; once undone the title is struck
// through and the line says so. Clicking it opens the task. Without its title (the inspector shows
// it already) the rest follows the verb directly.
func (m *Main) receiptLine(c *ui.Context, t *theme.Tokens, r data.Receipt, noTitle bool) {
	undone := m.convo.undone[r.ID]
	verb, tail := m.receiptWords(r, m.hereName())
	ink, quiet := t.Text, t.TextSecondary
	if undone {
		ink, quiet = t.TextTertiary, t.TextTertiary
	}
	row := ui.Row(c.Key("receipt-" + itoa(r.ID))).Height(answerLineHeight).Shrink(0).Gap(6).AlignItems(ui.Center).
		Cursor(ui.CursorPointer)
	row.Children(func() {
		ui.Icon(c, theme.IconCheck).Size(14, 14).TextColor(t.Status(data.Done)).Shrink(0)
		if verb != "" {
			theme.Text(c, theme.Compact, quiet, verb).Shrink(0)
		}
		if !noTitle {
			title := theme.Text(c, theme.CompactStrong, ink, r.Title).Shrink(1).MinWidth(0).Ellipsis("…")
			if undone {
				title.Strikethrough()
			}
		}
		if tail != "" {
			if !noTitle {
				tail = "· " + tail
			}
			words := theme.Text(c, theme.Compact, quiet, tail).Shrink(2).MinWidth(0).Ellipsis("…")
			if undone && noTitle {
				words.Strikethrough()
			}
		}
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

// hereName is where the composer's page is, so a receipt names a new task's project only when it
// landed somewhere else: the project's name, the Inbox, or nothing on Now.
func (m *Main) hereName() string {
	if m.convo.owner == surfaceInspector {
		return ""
	}
	switch m.nav.screen {
	case screenInbox:
		return m.say.NavInbox()
	case screenProject:
		return m.app.projectName(&m.nav.projectID)
	}
	return ""
}

// sentenceLine is what the assistant said when it changed nothing: one grey line with a small
// sparkle, "✦ That one is already in adhd-todo."
func (m *Main) sentenceLine(c *ui.Context, t *theme.Tokens, text string) {
	ui.Row(c).Shrink(0).Padding(5, 0).Gap(8).AlignItems(ui.Start).Children(func() {
		ui.Icon(c, theme.IconSparkle).Size(14, 14).TextColor(t.TextTertiary).Shrink(0).Margin(3, 0, 0)
		theme.Paragraph(c, theme.Compact, t.TextSecondary, text, 0).Shrink(1).MinWidth(0)
	})
}

// undoButton is the quiet Undo at the right of a receipt, in the accent color.
func (m *Main) undoButton(c *ui.Context, t *theme.Tokens, r data.Receipt) ui.Element {
	btn := ui.ButtonBase(c.Key("undo-"+itoa(r.ID))).Padding(3, 8).Margin(0, -8, 0, 0).Radius(theme.RadiusControl).Shrink(0).
		Label(m.say.A11yUndo(r.Title)).Cursor(ui.CursorPointer)
	if btn.Hovered() {
		btn.Background(t.BgSelected)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.Text(c, theme.Meta, t.Accent, m.say.ConvoUndo()) })
	return btn
}

// questionView is a question in body text with its options as the small tinted buttons of the
// Needs you rows. Typing a free answer into the composer works too.
func (m *Main) questionView(c *ui.Context, t *theme.Tokens, q data.Question, live bool) {
	theme.Paragraph(c, theme.Body, t.Text, q.Text, 0).Margin(2, 0, 0)
	if len(q.Options) == 0 || !live {
		return
	}
	ui.Row(c.Key("options")).Shrink(0).Margin(8, 0, 0).Gap(8).Wrap().Children(func() {
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

// errorView says what went wrong in neutral words, with Retry.
func (m *Main) errorView(c *ui.Context, t *theme.Tokens, text string) {
	ui.Row(c).Shrink(0).Margin(4, 0, 0).Gap(10).AlignItems(ui.Center).Children(func() {
		theme.Paragraph(c, theme.Body, t.TextSecondary, text, 0).Shrink(1).MinWidth(0)
		if link(c, t, m.say.ConvoRetry()).Shrink(0).Clicked() {
			m.retry()
		}
	})
}
