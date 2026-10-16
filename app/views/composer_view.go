package views

import (
	"math"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// pageComposer is the composer at the top of Now, a project or the Inbox, with the "Added to …"
// confirmation under it and the answer to the request it sent.
func (m *Main) pageComposer(c *ui.Context, t *theme.Tokens) {
	if !m.pageHasComposer() {
		return
	}
	m.composerBox(c, t, surfacePage, theme.Input)
	m.pictureNotice(c, t, surfacePage)
	m.notice(c, t)
	if m.replyIn(surfacePage, 0) {
		m.answerView(c, t, false)
	}
}

// composerBox is the one input: at rest a quiet line on a well (no border, no shadow) with the
// assistant's avatar at its left and ⌘K at its right; focused or holding text it turns to the page's own
// color with an accent ring and grows with its text up to six lines. With a model connected a second
// row then shows under the text: + for images at the left, the model chip and the round send button
// at the right; images attached sit above the text as thumbnails. Without a model the avatar and ⌘K
// give way to the send button in the line itself. The place is named by the page's title, not by the
// input.
func (m *Main) composerBox(c *ui.Context, t *theme.Tokens, s surface, style theme.Style) {
	cs := &m.cmp[s]
	ready := m.assistReady()
	focused, typed := cs.ref.Focused(c), cs.buf != ""
	holds := typed || (ready && len(cs.pics) > 0) // something to send
	thinking := m.convo.thinking && m.convo.owner == s
	// The second row shows once there is something to send, so focusing an empty box moves nothing;
	// an empty focused box has the + in the avatar's place instead.
	bar := ready && (holds || cs.modelOpen || cs.ctlHover)
	inline := ready && !bar && !thinking && (focused || cs.plusHover)
	if !bar {
		cs.ctlHover = false
	}
	if !inline {
		cs.plusHover = false
	}
	key := "composer"
	if s == surfaceInspector {
		key = "inspector-card"
	}
	box := ui.Column(c.Key(key)).Shrink(0).MinHeight(theme.ComposerHeight).Padding(10, 10, 10, 16).Gap(8).
		Justify(ui.Center).Radius(theme.ComposerRadius)
	cs.dragOver = ready && box.FileDragOver()
	switch {
	case cs.dragOver:
		box.Background(t.Accent.Alpha(0.06)).Border(1, t.Accent).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.1))
	case focused || holds:
		box.Background(t.Bg).Border(1, t.Accent.Alpha(0.5)).Shadow(0, 0, 0, 3, t.Accent.Alpha(0.1))
	case box.Hovered():
		box.Background(t.BgSubtle.Mix(t.Text, 0.03)).Border(1, ui.Transparent)
	default:
		box.Background(t.BgSubtle).Border(1, ui.Transparent)
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
			case inline:
				cs.plusHover = m.attachButton(c, t, s).Hovered()
			case !typed && (!focused || thinking) && !bar:
				m.assistantMark(c, s)
			}
			m.composerField(c, t, s, style)
			switch {
			case bar:
			case typed:
				m.sendButton(c, t, s)
			case !focused && s == surfacePage:
				theme.Text(c, theme.Meta, t.TextTertiary, m.say.ComposerHintIdle()).Shrink(0)
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

// composerBar is the second row of a focused composer with a model behind it.
func (m *Main) composerBar(c *ui.Context, t *theme.Tokens, s surface) {
	cs := &m.cmp[s]
	row := ui.Row(c.Key("composer-bar-" + itoa(int(s)))).Shrink(0).Gap(8).AlignItems(ui.Center).Height(theme.SendSize)
	cs.ctlHover = row.Hovered()
	row.Children(func() {
		m.attachButton(c, t, s)
		ui.Box(c).Grow(1).MinWidth(0)
		m.modelChip(c, t, s)
		if strings.TrimSpace(cs.buf) != "" || len(cs.pics) > 0 {
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
	box := ui.Box(c.Key(key + "-box")).Shrink(0).MinWidth(0).Grow(1)
	box.Children(func() {
		field := style.Apply(ui.TextAreaBase(c.Key(key), &cs.buf).Lines(theme.ComposerMinRows, theme.ComposerMaxRows)).
			TextColor(t.Text).Label(label).Bind(&cs.ref).OnTextKey(m.composerKey(s)).OnPaste(m.pasteHook(c, s))
		m.takeFocus(c, s)
		// An input method's composition is drawn before it reaches buf: the placeholder gives way to it.
		if cs.buf == "" && !field.Composing() {
			theme.Text(c, style, t.TextTertiary, m.placeholder(s)).Absolute().Left(0).Top(0).Right(0).Ellipsis("…").PassThrough()
		}
	})
}

// placeholder is what the empty field says: where it is, or that the assistant is working, or that
// it waits for an answer.
func (m *Main) placeholder(s surface) string {
	cv := &m.convo
	switch {
	case cv.thinking && cv.owner == s:
		return m.say.AskThinking()
	case cv.open && cv.owner == s && len(cv.turns) > 0 && cv.turns[len(cv.turns)-1].question != nil:
		return m.say.AskAnswerHint()
	}
	if s == surfacePage && m.nav.screen == screenAgents {
		return m.say.ComposerPlaceholderAgents()
	}
	switch m.placeOf(s).Kind {
	case data.PlaceCalendar:
		return m.say.CalendarPlaceholder(m.clock.CalendarDay(m.calendarDay()))
	case data.PlaceProject, data.PlaceInbox:
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

// assistantMark is the assistant's 18 pt avatar at the left of the input. While it works on a
// request from s it pulses, unless Reduce Motion is on.
func (m *Main) assistantMark(c *ui.Context, s surface) {
	icon := theme.AvatarIcon(c.Key("assistant"), theme.Assistant, theme.AvatarRow).Shrink(0)
	if !m.convo.thinking || m.convo.owner != s || c.Preferences().ReduceMotion {
		return
	}
	phase := float64(m.app.Now().UnixMilli()%pulsePeriod) / pulsePeriod
	icon.Opacity(float32(0.4 + 0.6*(0.5+0.5*math.Cos(2*math.Pi*phase))))
	c.After(40 * time.Millisecond)
}

// pulsePeriod is one breath of the assistant's avatar, in milliseconds.
const pulsePeriod = 1400

// notice is the line under the input after a plain task was added without a model: "✓ Added to
// Inbox", 1.5 s, fading at the end. When there is none, the quiet way to set a model up takes its
// place while the field is focused and empty.
func (m *Main) notice(c *ui.Context, t *theme.Tokens) {
	cs := &m.cmp[surfacePage]
	age := m.app.Now().Sub(cs.noticeAt)
	if cs.noticeAt.IsZero() || age >= noticeFor {
		m.setUpLine(c, t)
		return
	}
	c.After(noticeFor - age)
	opacity := float32(1)
	if age > noticeFor-300*time.Millisecond {
		opacity = float32(noticeFor-age) / float32(300*time.Millisecond)
	}
	ui.Row(c.Key("notice")).Height(answerLineHeight).Shrink(0).Margin(8, 0, 0).Padding(0, 12, 0, 16).Gap(6).
		Opacity(opacity).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, theme.IconCheck).Size(theme.IconSmall, theme.IconSmall).TextColor(t.Status(data.Done)).Shrink(0)
		theme.Text(c, theme.Compact, t.TextSecondary, cs.noticeText)
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
	ui.Row(c.Key("set-up")).Height(answerLineHeight).Shrink(0).Margin(8, 0, 0).Padding(0, 12, 0, 16).AlignItems(ui.Center).Children(func() {
		l := link(c, t, m.say.ComposerChipSetUp())
		cs.setUpHover = l.Hovered()
		if l.Clicked() {
			m.openSettings(PaneAssistant)
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
