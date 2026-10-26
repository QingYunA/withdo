package views

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/page"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// dockPad is the room around the composer's dock: above it, and under it at the window's edge.
const (
	dockTop    float32 = 8
	dockBottom float32 = 18
)

// pageComposer is the composer, pinned at the bottom of a project, in the column of the page above it. The "Added to …" confirmation and the answer to the
// request it sent sit directly above the box and push the page up as they appear. The page scrolls
// above the dock, so its last row is never behind the box.
func (m *Main) pageComposer(c *ui.Context, t *theme.Tokens) {
	if !m.pg.Composer || m.pg.Hero {
		m.dockHeight = 0
		return
	}
	pad := m.sidePad(c)
	maxWidth := theme.ContentMax
	switch m.pg.Kind {
	case page.KindCalendar:
		maxWidth = theme.CalendarMax
	case page.KindColumns:
		maxWidth = theme.BoardMax + 2*pad
	case page.KindBoard:
		maxWidth = m.boardLayout(c).head + 2*pad
	}
	dock := ui.Column(c.Key("composer-dock")).FillWidth().Shrink(0).AlignItems(ui.Center)
	dock.Children(func() {
		ui.Column(c).FillWidth().MaxWidth(maxWidth).Padding(dockTop, pad, dockBottom).Children(func() {
			if m.replyIn(surfacePage, 0) {
				m.answerView(c, t, false)
			}
			m.voiceLine(c, t)
			m.notice(c, t)
			m.pictureNotice(c, t, surfacePage)
			ui.Box(c).Height(8).Shrink(0)
			m.composerBox(c, t, surfacePage, theme.Input)
		})
	})
	// The toasts and the bulk bar ride above the dock. Bounds would be the previous frame's height,
	// a frame behind with nothing to build another; painting has this frame's, and a change asks
	// for the frame that moves them.
	dock.Draw(func(_ *ui.Painter, r ui.Rect) {
		if r.H != m.dockHeight {
			m.dockHeight = r.H
			c.AnimationFrame()
		}
	})
}

// composerBox is the one input: a page-colored box with a hairline border, the same at rest and
// focused. Focus (or text, or the assistant at work) changes only the border and the ring; the
// controls do not move. Left of the text: the assistant's avatar without a model, + for images with
// one, nothing once a second row shows (the + is in it) unless the assistant works on a request from
// the box and the line is empty, then the pulsing avatar. At the right: the model chip, the mic,
// and ⌘K on a page while the line is empty, or the send button once there is text and no model. It
// grows with its text up to six lines. With a model connected a second row shows under the text once
// there is something to send, and while the assistant works on a request from the box and its answer
// shows: + for images at the left, the model chip and the round send (or stop) button at the right;
// images attached sit above the text as thumbnails. The place is named by the page's title, not by
// the input.
func (m *Main) composerBox(c *ui.Context, t *theme.Tokens, s surface, style theme.Style) {
	cs := &m.cmp[s]
	ready := m.assistReady()
	focused, typed := cs.ref.Focused(c), cs.buf != ""
	holds := typed || (ready && len(cs.pics) > 0) // something to send
	// The second row shows once there is something to send and while the assistant works on a request
	// from this box, answer included, so focusing an empty box moves nothing.
	working := ready && m.convo.owner == s && (m.convo.thinking || m.replyIn(s, cs.task))
	bar := ready && (holds || working)
	active := focused || holds || working
	key := "composer"
	if s == surfaceInspector {
		key = "inspector-card"
	}
	box := ui.Column(c.Key(key)).Shrink(0).MinHeight(theme.ComposerHeight).Padding(10, 10, 10, 16).Gap(8).
		Justify(ui.Center).Radius(theme.ComposerRadius)
	cs.dragOver = ready && box.FileDragOver()
	switch {
	case cs.dragOver:
		box.Background(t.Accent.Alpha(0.06)).Border(theme.Hairline, t.Accent).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.1))
	case active:
		box.Background(t.Bg).Border(theme.Hairline, t.Accent.Alpha(0.5)).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.1))
	case box.Hovered():
		box.Background(t.Bg).Border(theme.Hairline, t.Border.Mix(t.Text, 0.25))
	default:
		box.Background(t.Bg).Border(theme.Hairline, t.Border)
	}
	box.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	if ready {
		if files := box.DroppedFiles(); len(files) > 0 {
			m.attachFiles(s, files)
			m.refocus(s)
		}
	}
	box.Children(func() {
		if ready {
			m.thumbnails(c, t, s)
		}
		ui.Row(c.Key(key + "-line")).Shrink(0).Gap(10).AlignItems(ui.Center).Children(func() {
			switch {
			case !ready:
				if !typed {
					m.assistantMark(c, s)
				}
			case !bar:
				m.attachButton(c, t, s)
			case working && !typed:
				m.assistantMark(c, s)
			}
			m.composerField(c, t, s, style)
			if !bar {
				m.modelChip(c, t, s, m.compactChip(s)) // nothing without a model
			}
			m.voiceButton(c, t, s) // nothing in the inspector, or where nothing can record
			switch {
			case bar:
			case typed:
				m.sendButton(c, t, s)
			case s == surfacePage:
				theme.Text(c, theme.Meta, t.TextSecondary, m.say.ComposerHintIdle(keyHint("⌘K"))).Shrink(0)
			}
		})
		if bar {
			m.composerBar(c, t, s)
		}
	})
	if box.Clicked() {
		cs.ref.Focus(c) // a click on the box, not on a control in it, is a click in the field
	}
}

// composerBar is the second row of a composer with a model behind it: + and, at the right, the model
// chip and the send button, which is the stop button while the assistant works on a request from here.
func (m *Main) composerBar(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	ui.Row(c.Key("composer-bar-" + itoa(int(s)))).Shrink(0).Gap(8).AlignItems(ui.Center).Height(theme.SendSize).Children(func() {
		m.attachButton(c, t, s)
		ui.Box(c).Grow(1).MinWidth(0)
		m.modelChip(c, t, s, false)
		switch {
		case m.convo.thinking && m.convo.owner == s:
			m.stopButton(c, t)
		case strings.TrimSpace(cs.buf) != "" || len(cs.pics) > 0:
			m.sendButton(c, t, s)
		}
	})
}

// composerField is the text area of a composer with its placeholder over it.
func (m *Main) composerField(c *ui.Context, t *theme.Tokens, s surface, style theme.Style) {
	cs := &m.cmp[s]
	label, key := m.say.ComposerLabel(), "composer-input"
	if s == surfaceInspector {
		label, key = m.say.ComposerLabelTask(), "composer-input-task"
	}
	box := ui.Box(c.Key(key + "-box")).Shrink(0).MinWidth(composerFieldMin).Grow(1)
	box.Children(func() {
		field := style.Apply(ui.TextAreaBase(c.Key(key), &cs.buf).Lines(theme.ComposerMinRows, theme.ComposerMaxRows)).
			TextColor(t.Text).Label(label).Bind(&cs.ref).OnTextKey(m.composerKey(s)).OnPaste(m.pasteHook(c, s))
		if cs.caretEnd {
			cs.caretEnd = false
			n := utf8.RuneCountInString(cs.buf)
			field.SetTextSelection(n, n)
		}
		m.takeFocus(c, s)
		// An input method's composition is drawn before it reaches buf: the placeholder gives way to it.
		if cs.buf == "" && !field.Composing() {
			theme.Text(c, style, t.TextSecondary, m.placeholder(s)).Absolute().Left(0).Top(0).Right(0).Ellipsis("…").PassThrough()
		}
	})
}

// composerFieldMin is the room the field keeps in a one-line box, so the placeholder stays readable:
// in the inspector's narrow box the model chip gives way first and ellipsizes.
const composerFieldMin float32 = 150

// placeholder is what the empty field says: where it is, or that the assistant is working, or that
// it waits for an answer.
func (m *Main) placeholder(s surface) string {
	cv := &m.convo
	switch {
	case cv.open && cv.owner == s && len(cv.turns) > 0 && cv.turns[len(cv.turns)-1].question != nil:
		return m.say.AskAnswerHint()
	}
	if s == surfacePage && m.pg.Hero {
		return m.say.ComposerPlaceholderHome()
	}
	switch m.placeOf(s).Kind {
	case data.PlaceProject:
		return m.say.ComposerPlaceholderAdd()
	case data.PlaceTask:
		return m.say.ComposerPlaceholderTask()
	}
	return m.say.ComposerPlaceholder()
}

// sendButton is the round button at the right of the box, drawn once there is text: accent when it
// can be sent, neutral while the assistant is still working on the last request.
func (m *Main) sendButton(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	can := (strings.TrimSpace(cs.buf) != "" || len(cs.pics) > 0) && !m.convo.thinking && cs.pending == 0
	btn := ui.ButtonBase(c.Key("send-"+itoa(int(s)))).Size(theme.SendSize, theme.SendSize).Shrink(0).Center().
		AlignSelf(ui.End).Radius(theme.SendSize / 2).Label(m.say.ComposerSend())
	switch {
	case can && btn.Hovered():
		btn.Background(t.Accent.Mix(t.Text, 0.12)).Cursor(ui.CursorPointer)
	case can:
		btn.Background(t.Accent).Cursor(ui.CursorPointer)
	default:
		btn.Background(t.SendIdle())
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { ui.Icon(c, theme.IconSend).Size(theme.IconRow, theme.IconRow).TextColor(t.AccentText) })
	if btn.Clicked() && can {
		m.submit(s)
	}
}

// stopButton is the round button that takes the send button's place while the assistant works on a
// request from the box: neutral, with the pixel stop mark. A click cancels the request, as Esc does.
func (m *Main) stopButton(c *ui.Context, t *theme.Tokens) {
	btn := ui.ButtonBase(c.Key("stop")).Size(theme.SendSize, theme.SendSize).Shrink(0).Center().
		AlignSelf(ui.End).Radius(theme.SendSize / 2).Cursor(ui.CursorPointer).Label(m.say.ComposerStop())
	if btn.Hovered() {
		btn.Background(t.StopWell().Mix(t.Text, 0.1))
	} else {
		btn.Background(t.StopWell())
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.PixelStop(c, theme.IconRow, t.TextSecondary) })
	if btn.Clicked() {
		m.cancelAsk()
	}
}

// assistantMark is the assistant's 18 pt avatar at the left of the input. While it works on a
// request from s it pulses, unless Reduce Motion is on.
func (m *Main) assistantMark(c *ui.Context, s surface) {
	m.pulse(c, theme.AvatarIcon(c.Key("assistant"), theme.Assistant, theme.AvatarRow).Shrink(0), s)
}

// pulse makes the assistant's avatar breathe while it works on a request from s.
func (m *Main) pulse(c *ui.Context, icon ui.Element, s surface) {
	if !m.convo.thinking || m.convo.owner != s || c.Preferences().ReduceMotion {
		return
	}
	phase := float64(m.app.Now().UnixMilli()%pulsePeriod) / pulsePeriod
	icon.Opacity(float32(0.4 + 0.6*(0.5+0.5*math.Cos(2*math.Pi*phase))))
	c.After(40 * time.Millisecond)
}

// pulsePeriod is one breath of the assistant's avatar, in milliseconds.
const pulsePeriod = 1400

// notice is the line under the input after tasks were added without a model: "✓ Added to Inbox"
// for a plain one (1.5 s), else the receipt of what the local rules read, with Undo (6 s), fading
// at the end. When there is none, the quiet way to set a model up takes its place while the field
// is focused and empty.
func (m *Main) notice(c *ui.Context, t *theme.Tokens) {
	cs := &m.cmp[surfacePage]
	length := cs.noticeLen
	if length == 0 {
		length = noticeFor
	}
	age := m.app.Now().Sub(cs.noticeAt)
	if cs.noticeAt.IsZero() || age >= length {
		m.setUpLine(c, t)
		return
	}
	c.After(length - age)
	opacity := float32(1)
	if age > length-300*time.Millisecond {
		opacity = float32(length-age) / float32(300*time.Millisecond)
	}
	top, left := m.lineEdge()
	ui.Row(c.Key("notice")).Height(answerLineHeight).Shrink(0).Margin(top, 0, 0).Padding(0, 12, 0, left).Gap(6).
		Opacity(opacity).AlignItems(ui.Center).Children(func() {
		style := theme.Meta
		if !m.homeLine {
			ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Status(data.Done)).Shrink(0)
			style = theme.Compact
		}
		theme.Text(c, style, t.TextSecondary, cs.noticeText).Shrink(0).MinWidth(0).Ellipsis("…")
		if m.undos.cur == nil || m.undos.seq != cs.undoTicket {
			return
		}
		if link(c, t, m.say.MenuUndo()).Clicked() {
			e := m.undos.cur
			m.undos.clear()
			cs.noticeAt = time.Time{}
			e.undo()
		}
	})
}

// setUpLine is the one extra line of a composer with no model behind it: while the field is
// focused and empty, a quiet way to Settings > Assistant. Enter adds a plain task meanwhile.
func (m *Main) setUpLine(c *ui.Context, t *theme.Tokens) {
	cs := &m.cmp[surfacePage]
	if m.app.Assistant == nil || m.assistReady() || cs.buf != "" || !(cs.ref.Focused(c) || cs.setUpHover) {
		cs.setUpHover = false
		return
	}
	// A press on the line takes the focus from the field: the line stays while the pointer is on it,
	// so the click is not lost.
	top, left := m.lineEdge()
	ui.Row(c.Key("set-up")).Height(answerLineHeight).Shrink(0).Margin(top, 0, 0).Padding(0, 12, 0, left).AlignItems(ui.Center).Children(func() {
		l := link(c, t, m.say.ComposerChipSetUp())
		cs.setUpHover = l.Hovered()
		if l.Clicked() {
			m.openSettings(PaneAdvanced)
		}
	})
}

// inspectorComposer is the compact composer at the bottom of the inspector: it edits the open
// task in plain words. The reply of its request shows above it. It is not drawn while no model is
// connected.
func (m *Main) inspectorComposer(c *ui.Context, t *theme.Tokens, task data.Task) {
	if !m.assistReady() {
		return
	}
	cs := &m.cmp[surfaceInspector]
	if cs.task != task.ID {
		cs.task, cs.buf, cs.frames = task.ID, "", 0
	}
	hairline(c, t) // the properties scroll above it; this stack stays where it is
	ui.Column(c.Key("inspector-composer")).Shrink(0).Padding(12, 14, 14).Children(func() {
		if m.replyIn(surfaceInspector, task.ID) {
			// The words line up with the properties above (24 pt); a receipt's well reaches out to the box's edge.
			ui.Scroll(c.Key("inspector-reply")).MaxHeight(inspectorReplyMax).Margin(0, 0, 10).Children(func() {
				ui.Column(c).Padding(0, 10).Children(func() { m.answerView(c, t, true) })
			})
		}
		m.composerBox(c, t, surfaceInspector, theme.Body)
		m.pictureNotice(c, t, surfaceInspector)
	})
}

// inspectorReplyMax keeps a long reply from pushing the properties out of the inspector.
const inspectorReplyMax float32 = 240
